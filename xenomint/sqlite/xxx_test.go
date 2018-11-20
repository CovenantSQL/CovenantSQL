/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sqlite

import (
	"io/ioutil"
	"math/rand"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	testingDataDir string
)

func setup() {
	const minNoFile uint64 = 4096
	var (
		err error
		lmt syscall.Rlimit
	)

	if testingDataDir, err = ioutil.TempDir("", "CovenantSQL"); err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())

	if err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lmt); err != nil {
		panic(err)
	}
	if lmt.Max < minNoFile {
		panic("insufficient max RLIMIT_NOFILE")
	}
	lmt.Cur = lmt.Max
	if err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lmt); err != nil {
		panic(err)
	}

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func teardown() {
	if err := os.RemoveAll(testingDataDir); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	os.Exit(func() int {
		setup()
		defer teardown()
		return m.Run()
	}())
}
