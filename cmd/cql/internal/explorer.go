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
	"net/http"

	"github.com/CovenantSQL/CovenantSQL/sqlchain/observer"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

var (
	explorerAddr string // Explorer addr

	explorerService    *observer.Service
	explorerHTTPServer *http.Server
)

// CmdExplorer is cql explorer command.
var CmdExplorer = &Command{
	UsageLine: "cql explorer [common params] [-tmp-path path] [-bg-log-level level] listen_address",
	Short:     "start a SQLChain explorer server",
	Long: `
Explorer serves a SQLChain web explorer.
e.g.
    cql explorer 127.0.0.1:8546
`,
}

func init() {
	CmdExplorer.Run = runExplorer

	addCommonFlags(CmdExplorer)
	addBgServerFlag(CmdExplorer)
}

func startExplorerServer(explorerAddr string) func() {
	var err error
	explorerService, explorerHTTPServer, err = observer.StartObserver(explorerAddr, Version)
	if err != nil {
		ConsoleLog.WithError(err).Error("start explorer failed")
		SetExitStatus(1)
		return nil
	}

	ConsoleLog.Infof("explorer server started on %s", explorerAddr)

	return func() {
		_ = observer.StopObserver(explorerService, explorerHTTPServer)
		ConsoleLog.Info("explorer stopped")
	}
}

func runExplorer(cmd *Command, args []string) {
	configInit(cmd)
	bgServerInit()

	if len(args) != 1 {
		ConsoleLog.Error("Explorer command need listen address as param")
		SetExitStatus(1)
		return
	}
	explorerAddr = args[0]

	cancelFunc := startExplorerServer(explorerAddr)
	ExitIfErrors()
	defer cancelFunc()

	ConsoleLog.Printf("Ctrl + C to stop explorer server on %s\n", explorerAddr)
	<-utils.WaitForExit()
}
