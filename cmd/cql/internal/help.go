/*
 * Copyright 2018-2019 The CovenantSQL Authors.
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

package internal

import (
	"fmt"
	"runtime"
)

const name = "cql"

var (
	// Version of command, set by main func of version
	Version = "unknown"
)

// CmdVersion is cql version command entity.
var CmdVersion = &Command{
	UsageLine:   "cql version",
	Description: "Show cql build version infomation",
}

func init() {
	CmdVersion.Run = runVersion
}

// PrintVersion prints program git version.
func PrintVersion(printLog bool) string {
	version := fmt.Sprintf("%v %v %v %v %v\n",
		name, Version, runtime.GOOS, runtime.GOARCH, runtime.Version())

	if printLog {
		ConsoleLog.Infof("cql build: %s", version)
	}

	return version
}

func runVersion(cmd *Command, args []string) {
	fmt.Print(PrintVersion(false))
}
