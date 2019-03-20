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
	webAddr string // Web addr

	webService    *observer.Service
	webHTTPServer *http.Server
)

// CmdWeb is cql web command.
var CmdWeb = &Command{
	UsageLine: "cql web [-config file] [-tmp-path path] [-bg-log-level level] [address]",
	Short:     "start a SQLChain web explorer",
	Long: `
Web command serves a SQLChain web explorer.
e.g.
    cql web 127.0.0.1:8546
`,
}

func init() {
	CmdWeb.Run = runWeb

	addCommonFlags(CmdWeb)
	addBgServerFlag(CmdWeb)
}

func startWebServer(webAddr string) func() {
	var err error
	webService, webHTTPServer, err = observer.StartObserver(webAddr, Version)
	if err != nil {
		ConsoleLog.WithError(err).Error("start explorer failed")
		SetExitStatus(1)
		return nil
	}

	ConsoleLog.Infof("web server started on %s", webAddr)

	return func() {
		_ = observer.StopObserver(webService, webHTTPServer)
		ConsoleLog.Info("explorer stopped")
	}
}

func runWeb(cmd *Command, args []string) {
	configInit()
	bgServerInit()

	if len(args) != 1 {
		ConsoleLog.Error("Web command need listern address as param")
		SetExitStatus(1)
		return
	}
	webAddr = args[0]

	cancelFunc := startWebServer(webAddr)
	ExitIfErrors()
	defer cancelFunc()

	ConsoleLog.Printf("Ctrl + C to stop web server on %s\n", webAddr)
	<-utils.WaitForExit()
}
