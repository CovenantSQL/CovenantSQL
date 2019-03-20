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
	"os"
	"runtime"
)

const name = "cql"

var (
	// Version of command, set by main func of version
	Version = "unknown"
)

// CmdVersion is cql version command entity.
var CmdVersion = &Command{
	UsageLine: "cql version",
	Short:     "show build version infomation",
	Long: `
Use "cql help <command>" for more information about a command.
`,
}

// CmdHelp is cql help command entity.
var CmdHelp = &Command{
	UsageLine: "cql help [command]",
	Short:     "show help of sub commands",
	Long: `
Use "cql help <command>" for more information about a command.
`,
}

func init() {
	CmdVersion.Run = runVersion
	CmdHelp.Run = runHelp
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

func runHelp(cmd *Command, args []string) {
	if len(args) != 1 {
		MainUsage()
	}

	cmdName := args[0]
	for _, cmd := range CqlCommands {
		if cmd.Name() != cmdName {
			continue
		}
		fmt.Fprintf(os.Stderr, "usage: %s\n", cmd.UsageLine)
		fmt.Fprintf(os.Stderr, cmd.Long)
		fmt.Fprintf(os.Stderr, "\nParams:\n")
		cmd.Flag.PrintDefaults()
		return
	}

	//Not support command
	SetExitStatus(2)
	MainUsage()
}

// MainUsage prints cql base help
func MainUsage() {
	helpHead := `cql is a tool for managing CovenantSQL database.

Usage:

    cql <command> [-params] [arguments]

The commands are:

`
	helpTail := `
Use "cql help <command>" for more information about a command.
`

	helpMsg := helpHead
	for _, cmd := range CqlCommands {
		if cmd.Name() == "help" {
			continue
		}
		cmdName := cmd.Name()
		for len(cmdName) < 10 {
			cmdName += " "
		}
		helpMsg += "\t" + cmdName + "\t" + cmd.Short + "\n"
	}
	helpMsg += helpTail

	fmt.Fprintf(os.Stderr, helpMsg)
	Exit()
}
