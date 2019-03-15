/*
 * Copyright 2016-2018 Kenneth Shaw.
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

//	// Shard chain explorer/adapter stuff
//	tmpPath      string // background observer and explorer block and log file path
//	bgLogLevel   string // background log level
//	explorerAddr string // explorer Web addr
//	adapterAddr  string // adapter listen addr
//
//	service    *observer.Service
//	httpServer *http.Server
)

func init() {
	// // Explorer/Adapter
	// flag.StringVar(&tmpPath, "tmp-path", "", "Background service temp file path, use os.TempDir for default")
	// flag.StringVar(&bgLogLevel, "bg-log-level", "", "Background service log level") // flag.StringVar(&explorerAddr, "web", "", "Address to serve a database chain explorer, e.g. :8546")
	// flag.StringVar(&adapterAddr, "adapter", "", "Address to serve a database chain adapter, e.g. :7784")

	internal.CqlCommands = []*internal.Command{
		internal.CmdConsole,
		internal.CmdVersion,
		internal.CmdBalance,
		internal.CmdCreate,
		internal.CmdDrop,
		internal.CmdPermission,
		internal.CmdTransfer,
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
	//
	//	if tmpPath == "" {
	//		tmpPath = os.TempDir()
	//	}
	//	logPath := filepath.Join(tmpPath, "covenant_service.log")
	//	bgLog, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	//	if err != nil {
	//		fmt.Fprintf(os.Stderr, "open log file failed: %s, %v", logPath, err)
	//		os.Exit(-1)
	//	}
	//	log.SetOutput(bgLog)
	//	log.SetStringLevel(bgLogLevel, log.InfoLevel)
	//
	//	if explorerAddr != "" {
	//		service, httpServer, err = observer.StartObserver(explorerAddr, internal.Version)
	//		if err != nil {
	//			log.WithError(err).Fatal("start explorer failed")
	//		} else {
	//			internal.ConsoleLog.Infof("explorer started on %s", explorerAddr)
	//		}
	//
	//		defer func() {
	//			_ = observer.StopObserver(service, httpServer)
	//			log.Info("explorer stopped")
	//		}()
	//	}
	//
	//	if adapterAddr != "" {
	//		server, err := adapter.NewHTTPAdapter(adapterAddr, configFile)
	//		if err != nil {
	//			log.WithError(err).Fatal("init adapter failed")
	//		}
	//
	//		if err = server.Serve(); err != nil {
	//			log.WithError(err).Fatal("start adapter failed")
	//		} else {
	//			internal.ConsoleLog.Infof("adapter started on %s", adapterAddr)
	//
	//			defer func() {
	//				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	//				defer cancel()
	//				server.Shutdown(ctx)
	//				log.Info("stopped adapter")
	//			}()
	//		}
	//	}
	//
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

	// if web flag is enabled
	//if explorerAddr != "" || adapterAddr != "" {
	//	fmt.Printf("Ctrl + C to stop explorer on %s and adapter on %s\n", explorerAddr, adapterAddr)
	//	<-utils.WaitForExit()
	//	return
	//}
}

func mainUsage() {
	//TODO(laodouya) print stderr main usage
	os.Exit(2)
}
