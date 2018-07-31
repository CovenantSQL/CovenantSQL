// +build !nomeminfo

package metric

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func (c *meminfoCollector) getMemInfo() (map[string]float64, error) {
	file, err := os.Open(procFilePath("meminfo"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := parseMemInfo(file)
	if _, ok := info["MemAvailable_bytes"]; !ok {
		if err == nil {
			free, ok := info["MemFree_bytes"]
			if ok {
				buf, ok := info["Buffers_bytes"]
				if ok {
					cache, ok := info["Cached_bytes"]
					if ok {
						info["MemAvailable_bytes"] = free + buf + cache
					}
				}
			}
		}
	}
	return info, err
}

func parseMemInfo(r io.Reader) (map[string]float64, error) {
	var (
		memInfo = map[string]float64{}
		scanner = bufio.NewScanner(r)
		re      = regexp.MustCompile(`\((.*)\)`)
	)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		fv, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value in meminfo: %s", err)
		}
		key := parts[0][:len(parts[0])-1] // remove trailing : from key
		// Active(anon) -> Active_anon
		key = re.ReplaceAllString(key, "_${1}")
		switch len(parts) {
		case 2: // no unit
		case 3: // has unit, we presume kB
			fv *= 1024
			key = key + "_bytes"
		default:
			return nil, fmt.Errorf("invalid line in meminfo: %s", line)
		}
		memInfo[key] = fv
	}

	return memInfo, scanner.Err()
}
