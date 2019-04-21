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
	"bytes"
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
		ConsoleLog.Debugf("cql build: %s\n", version)
	}

	return version
}

func runVersion(cmd *Command, args []string) {
	fmt.Print(PrintVersion(false))
}

func runHelp(cmd *Command, args []string) {
	if l := len(args); l != 1 {
		if l > 1 {
			// Don't support multiple commands
			SetExitStatus(2)
		}
		MainUsage()
	}

	cmdName := args[0]
	for _, cmd := range CqlCommands {
		if cmd.Name() != cmdName {
			continue
		}
		fmt.Fprintf(os.Stdout, "usage: %s\n", cmd.UsageLine)
		fmt.Fprintf(os.Stdout, cmd.Long)
		fmt.Fprintf(os.Stdout, "\nParams:\n")
		cmd.Flag.SetOutput(os.Stdout)
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

    cql <command> [params] [arguments]

The commands are:

`
	helpCommon := `
The common params for commands (except help and version) are:

`

	helpTail := `
Use "cql help <command>" for more information about a command.
`

	output := bytes.NewBuffer(nil)
	output.WriteString(helpHead)
	for _, cmd := range CqlCommands {
		if cmd.Name() == "help" {
			continue
		}
		fmt.Fprintf(output, "\t%-10s\t%s\n", cmd.Name(), cmd.Short)
	}

	addCommonFlags(CmdHelp)
	fmt.Fprint(output, helpCommon)
	CmdHelp.Flag.SetOutput(output)
	CmdHelp.Flag.PrintDefaults()

	fmt.Fprint(output, helpTail)
	fmt.Fprintf(os.Stdout, output.String())
	Exit()
}
