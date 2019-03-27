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
	"time"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql/internal"
)

var (
	version = "unknown"
)

func init() {
	internal.CqlCommands = []*internal.Command{
		internal.CmdGenerate,
		internal.CmdConsole,
		internal.CmdCreate,
		internal.CmdDrop,
		internal.CmdWallet,
		internal.CmdTransfer,
		internal.CmdGrant,
		internal.CmdMirror,
		internal.CmdExplorer,
		internal.CmdAdapter,
		internal.CmdIDMiner,
		internal.CmdRPC,
		internal.CmdVersion,
		internal.CmdHelp,
	}
}

func main() {
	internal.Version = version

	// set random
	rand.Seed(time.Now().UnixNano())

	flag.Usage = internal.MainUsage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		internal.MainUsage()
	}

	if args[0] != "version" && args[0] != "help" {
		internal.PrintVersion(true)
	}

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
	fmt.Fprintf(os.Stderr, "cql %s: unknown command\nRun 'cql help' for usage.\n", args[0])
	internal.SetExitStatus(2)
	internal.Exit()
}
