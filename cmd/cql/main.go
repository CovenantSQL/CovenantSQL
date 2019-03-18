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

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql/internal"
)

var (
	version = "unknown"
)

func init() {
	internal.CqlCommands = []*internal.Command{
		internal.CmdConsole,
		internal.CmdVersion,
		internal.CmdBalance,
		internal.CmdCreate,
		internal.CmdDrop,
		internal.CmdPermission,
		internal.CmdTransfer,
		internal.CmdWeb,
		internal.CmdAdapter,
	}
}

func main() {

	internal.Version = version

	// set random
	rand.Seed(time.Now().UnixNano())

	flag.Usage = mainUsage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		mainUsage()
	}

	internal.CmdName = args[0] // for error messages
	if args[0] == "help" {
		mainUsage()
		return
	}

	internal.PrintVersion(true)

	for _, cmd := range internal.CqlCommands {
		if cmd.Name() != args[0] {
			continue
		}
		if !cmd.Runnable() {
			continue
		}
		cmd.Flag.Usage = func() { cmd.Usage() }
		cmd.Flag.Parse(args[1:])
		args = cmd.Flag.Args()
		cmd.Run(cmd, args)
		internal.Exit()
		return
	}
	helpArg := ""
	if i := strings.LastIndex(internal.CmdName, " "); i >= 0 {
		helpArg = " " + internal.CmdName[:i]
	}
	fmt.Fprintf(os.Stderr, "cql %s: unknown command\nRun 'cql help%s' for usage.\n", internal.CmdName, helpArg)
	internal.SetExitStatus(2)
	internal.Exit()
}

func mainUsage() {
	//TODO(laodouya) print stderr main usage
	os.Exit(2)
}
