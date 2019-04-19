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
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	waitTxConfirmationMaxDuration time.Duration

	// ConsoleLog is logging for console.
	ConsoleLog *logrus.Logger

	// CqlCommands initialized in package main
	CqlCommands []*Command
)

func init() {
	ConsoleLog = logrus.New()
	ConsoleLog.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})
}

// A Command is an implementation of a cql command
// like cql create or cql transfer.
type Command struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(cmd *Command, args []string)

	// UsageLine is the one-line usage message.
	// The first word in the line is taken to be the command name.
	UsageLine string

	// Short is the short description shown in the 'cql help' output.
	Short string

	// Long is the long message shown in the 'cql help <this-command>' output.
	Long string

	// Flag is a set of flags specific to this command.
	Flag flag.FlagSet
}

// LongName returns the command's long name: all the words in the usage line between "cql" and a flag or argument,
func (c *Command) LongName() string {
	name := c.UsageLine
	if i := strings.Index(name, " ["); i >= 0 {
		name = name[:i]
	}
	if name == "cql" {
		return ""
	}
	return strings.TrimPrefix(name, "cql ")
}

// Name returns the command's short name: the last word in the usage line before a flag or argument.
func (c *Command) Name() string {
	name := c.LongName()
	if i := strings.LastIndex(name, " "); i >= 0 {
		name = name[i+1:]
	}
	return name
}

// Usage print base usage help info.
func (c *Command) Usage() {
	fmt.Fprintf(os.Stdout, "usage: %s\n", c.UsageLine)
	fmt.Fprintf(os.Stdout, "Run 'cql help %s' for details.\n", c.LongName())
	os.Exit(2)
}

// Runnable reports whether the command can be run; otherwise
// it is a documentation pseudo-command such as importpath.
func (c *Command) Runnable() bool {
	return c.Run != nil
}

var atExitFuncs []func()

// AtExit will register function to be executed before exit.
func AtExit(f func()) {
	atExitFuncs = append(atExitFuncs, f)
}

// Exit will run all exit funcs and then return with exitStatus
func Exit() {
	for _, f := range atExitFuncs {
		f()
	}
	os.Exit(exitStatus)
}

// ExitIfErrors will call Exit() if exitStatus is not 0
func ExitIfErrors() {
	if exitStatus != 0 {
		Exit()
	}
}

var exitStatus = 0
var exitMu sync.Mutex

// SetExitStatus provide thread safe set exit status func.
func SetExitStatus(n int) {
	exitMu.Lock()
	if exitStatus < n {
		exitStatus = n
	}
	exitMu.Unlock()
}
