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

package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
)

// DupConf duplicate conf file using random new listen addr to avoid failure on concurrent test cases.
func DupConf(confFile string, newConfFile string) (err error) {
	// replace port in confFile
	var fileBytes []byte
	if fileBytes, err = ioutil.ReadFile(confFile); err != nil {
		return
	}

	var ports []int
	if ports, err = GetRandomPorts("127.0.0.1", 4000, 6000, 1); err != nil {
		return
	}

	newConfBytes := bytes.Replace(fileBytes, []byte(":2230"), []byte(fmt.Sprintf(":%v", ports[0])), -1)

	return ioutil.WriteFile(newConfFile, newConfBytes, 0644)
}
