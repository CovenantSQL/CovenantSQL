// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on ioutil Go package, modified to add
// file extension option to File() and renamed to temp
// Lefteris Zafiris 2017

package temp

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// Random number state.
// We generate random temporary file names so that there's a good
// chance the file doesn't exist yet - keeps the number of tries in
// TempFile to a minimum.
var rand uint32
var randmu sync.Mutex

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextSuffix() string {
	randmu.Lock()
	r := rand
	if r == 0 {
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	rand = r
	randmu.Unlock()
	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

// File creates a new temporary file with the provided extension
// in the directory dir with a name beginning with prefix,
// opens the file for reading and writing, and returns the resulting *os.File.
// If dir is the empty string, File uses the default directory
// for temporary files (see os.TempDir).
// Multiple programs calling File simultaneously
// will not choose the same file.  The caller can use f.Name()
// to find the pathname of the file.  It is the caller's responsibility
// to remove the file when no longer needed.
func File(dir, prefix, extension string) (f *os.File, err error) {
	if dir == "" {
		dir = os.TempDir()
	}

	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(dir, prefix+nextSuffix())
		if extension != "" {
			name += "." + extension
		}
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		break
	}
	return
}

// Dir creates a new temporary directory in the directory dir
// with a name beginning with prefix and returns the path of the
// new directory.  If dir is the empty string, Dir uses the
// default directory for temporary files (see os.TempDir).
// Multiple programs calling Dir simultaneously
// will not choose the same directory.  It is the caller's responsibility
// to remove the directory when no longer needed.
func Dir(dir, prefix string) (name string, err error) {
	if dir == "" {
		dir = os.TempDir()
	}

	nconflict := 0
	for i := 0; i < 10000; i++ {
		try := filepath.Join(dir, prefix+nextSuffix())
		err = os.Mkdir(try, 0700)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		if os.IsNotExist(err) {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return "", err
			}
		}
		if err == nil {
			name = try
		}
		break
	}
	return
}
