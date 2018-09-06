// +build ignore

package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/knq/snaker"
)

const (
	ncursesSrc = "https://ftp.gnu.org/pub/gnu/ncurses/ncurses-6.1.tar.gz"

	capsFile = "ncurses-6.1/include/Caps"
)

var (
	flagCache = flag.String("cache", ".cache", "cache directory")
	flagOut   = flag.String("out", "capvals.go", "out file")
)

var commentRE = regexp.MustCompile(`^#.*`)

func notSpace(r rune) bool {
	return !unicode.IsSpace(r)
}

func main() {
	flag.Parse()

	var err error

	// get archive
	buf, err := get(ncursesSrc)
	if err != nil {
		log.Fatal(err)
	}

	// load caps file
	capsBuf, err := load(buf, capsFile)
	if err != nil {
		log.Fatal(err)
	}

	// process caps
	outBuf, err := processCaps(capsBuf)
	if err != nil {
		log.Fatal(err)
	}

	// write
	err = ioutil.WriteFile(*flagOut, outBuf, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// gofmt
	err = exec.Command("gofmt", "-w", "-s", *flagOut).Run()
	if err != nil {
		log.Fatal(err)
	}
}

// get retrieves a file either from the the http path, or from disk.
func get(file string) ([]byte, error) {
	err := os.MkdirAll(*flagCache, 0755)
	if err != nil {
		return nil, err
	}

	// check if the file exists
	cacheFile := filepath.Join(*flagCache, filepath.Base(file))
	fi, err := os.Stat(cacheFile)
	if err == nil && !fi.IsDir() {
		log.Printf("loading %s", cacheFile)
		return ioutil.ReadFile(cacheFile)
	}

	// retrieve
	log.Printf("retrieving %s", file)
	res, err := http.Get(file)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// read
	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// cache
	log.Printf("saving %s", cacheFile)
	err = ioutil.WriteFile(cacheFile, buf, 0644)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// load extracts a file from a tar.gz.
func load(buf []byte, file string) ([]byte, error) {
	// create gzip reader
	gr, err := gzip.NewReader(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	// walk files in tar
	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		// found file, read contents
		if h.Name == file {
			b := bytes.NewBuffer(make([]byte, h.Size))

			var n int64
			n, err = io.Copy(b, tr)
			if err != nil {
				return nil, err
			}

			// check that all bytes were copied
			if n != h.Size {
				return nil, errors.New("could not read entire file")
			}
			return b.Bytes(), nil
		}
	}

	return nil, fmt.Errorf("could not load file %s", file)
}

// processCaps processes the data in the Caps file.
func processCaps(capsBuf []byte) ([]byte, error) {
	var err error

	// create scanner
	s := bufio.NewScanner(bytes.NewReader(capsBuf))
	s.Buffer(make([]byte, 1024*1024), 1024*1024)

	// storage
	bools, nums, strs := new(bytes.Buffer), new(bytes.Buffer), new(bytes.Buffer)
	var boolCount, numCount, stringCount int
	var lastBool, lastNum, lastString string
	var boolNames, numNames, stringNames []string

	// process caps
	var n int
	for s.Scan() {
		// read line
		line := strings.TrimSpace(commentRE.ReplaceAllString(strings.Trim(s.Text(), "\x00"), ""))
		if len(line) == 0 || strings.HasPrefix(line, "capalias") || strings.HasPrefix(line, "infoalias") {
			continue
		}

		// split line's columns
		row := make([]string, 8)
		for i := 0; i < 7; i++ {
			start := strings.IndexFunc(line, unicode.IsSpace)
			end := strings.IndexFunc(line[start:], notSpace)
			row[i] = strings.TrimSpace(line[:start+end])
			line = line[start+end:]
		}
		row[7] = strings.TrimSpace(line)

		// manipulation
		var buf *bytes.Buffer
		var names *[]string
		var typ, isFirst, prefix, suffix string

		// format variable name
		name := snaker.SnakeToCamel(row[0])

		switch row[2] {
		case "bool":
			if boolCount == 0 {
				isFirst = " = iota"
			}
			buf, names, lastBool, prefix, suffix = bools, &boolNames, name, "indicates", ""
			typ = "bool"
			boolCount++

		case "num":
			if numCount == 0 {
				isFirst = " = iota"
			}
			buf, names, lastNum, prefix, suffix = nums, &numNames, name, "is", ""
			typ = "num"
			numCount++

		case "str":
			if stringCount == 0 {
				isFirst = " = iota"
			}
			buf, names, lastString, prefix, suffix = strs, &stringNames, name, "is the", ""
			typ = "string"
			stringCount++

		default:
			log.Fatal("line %d is invalid, has type: %s", n, row[2])
		}

		if isFirst == "" {
			buf.WriteString("\n")
		}
		buf.WriteString(fmt.Sprintf("// The %s [%s, %s] %s capability ", name, row[0], row[1], typ) + formatComment(row[7], prefix, suffix) + "\n" + name + isFirst + "\n")
		*names = append(*names, row[0], row[1])

		n++
	}
	err = s.Err()
	if err != nil {
		return nil, err
	}

	f := new(bytes.Buffer)
	f.WriteString(hdr)

	// add consts
	typs := []string{"Bool", "Num", "String"}
	for i, b := range []*bytes.Buffer{bools, nums, strs} {
		f.WriteString(fmt.Sprintf("// %s capabilities.\nconst (\n", typs[i]))
		b.WriteTo(f)
		f.WriteString(")\n\n")
	}

	// add counts
	f.WriteString("const (\n")
	f.WriteString(fmt.Sprintf("// CapCountBool is the count of bool capabilities.\nCapCountBool = %s+1\n\n", lastBool))
	f.WriteString(fmt.Sprintf("// CapCountNum is the count of num capabilities.\nCapCountNum = %s+1\n\n", lastNum))
	f.WriteString(fmt.Sprintf("// CapCountString is the count of string capabilities.\nCapCountString = %s+1\n", lastString))
	f.WriteString(")\n\n")

	// add names
	z := []string{"bool", "num", "string"}
	for n, s := range [][]string{boolNames, numNames, stringNames} {
		y := z[n]
		f.WriteString(fmt.Sprintf("// %sCapNames are the %s term cap names.\n", y, y))
		f.WriteString(fmt.Sprintf("var %sCapNames = [...]string{\n", y))
		for i := 0; i < len(s); i += 2 {
			f.WriteString(fmt.Sprintf(`"%s", "%s",`+"\n", s[i], s[i+1]))
		}
		f.WriteString("}\n")
	}

	return f.Bytes(), nil
}

// formatComment formats comments with prefix and suffix.
func formatComment(s, prefix, suffix string) string {
	s = strings.TrimPrefix(s, prefix)
	s = strings.TrimSuffix(s, ".")
	s = strings.TrimSuffix(s, suffix)

	return strings.TrimSpace(prefix+" "+s+" "+suffix) + "."
}

const (
	hdr = `package terminfo

	// Code generated by gen.go. DO NOT EDIT.

`
)
