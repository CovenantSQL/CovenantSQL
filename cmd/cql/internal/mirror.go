/*
 * Copyright 2019 The CovenantSQL Authors.
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
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/sqlchain/mirror"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

var (
	mirrorDatabase string // mirror database id
	mirrorAddr     string // mirror server rpc addr

	mirrorService *mirror.Service
)

// CmdMirror is cql mirror command.
var CmdMirror = &Command{
	UsageLine: "cql mirror [common params] [-tmp-path path] [-bg-log-level level] dsn listen_address",
	Short:     "start a SQLChain database mirror server",
	Long: `
Mirror subscribes database updates and serves a read-only database mirror.
e.g.
    cql mirror dsn 127.0.0.1:9389
`,
}

func init() {
	CmdMirror.Run = runMirror

	addCommonFlags(CmdMirror)
	addBgServerFlag(CmdMirror)
}

func startMirrorServer(mirrorDatabase string, mirrorAddr string) func() {
	var err error
	mirrorService, err = mirror.StartMirror(mirrorDatabase, mirrorAddr)
	if err != nil {
		ConsoleLog.WithError(err).Error("start mirror failed")
		SetExitStatus(1)
		return nil
	}

	ConsoleLog.Infof("mirror server started on %s", mirrorAddr)
	// TODO(): print sample command for cql to connect

	return func() {
		mirror.StopMirror(mirrorService)
		ConsoleLog.Info("mirror stopped")
	}
}

func runMirror(cmd *Command, args []string) {
	configInit(cmd)
	bgServerInit()

	if len(args) != 2 {
		ConsoleLog.Error("Missing args, run `cql help mirror` for help")
		SetExitStatus(1)
		return
	}

	dsn := args[0]
	mirrorAddr = args[1]

	cfg, err := client.ParseDSN(dsn)
	if err != nil {
		// not a dsn/dbid
		ConsoleLog.WithField("db", dsn).WithError(err).Error("Not a valid dsn")
		SetExitStatus(1)
		return
	}

	mirrorDatabase = cfg.DatabaseID

	cancelFunc := startMirrorServer(mirrorDatabase, mirrorAddr)
	ExitIfErrors()
	defer cancelFunc()

	ConsoleLog.Printf("Ctrl + C to stop mirror server on %s\n", mirrorAddr)
	<-utils.WaitForExit()
}
