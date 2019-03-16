package internal

import (
	"context"
	"net/http"
	"time"

	"github.com/CovenantSQL/CovenantSQL/sqlchain/adapter"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

var (
	adapterAddr string // adapter listen addr

	adapterHTTPServer *http.Server
)

var CmdAdapter = &Command{
	UsageLine:   "cql adapter [-tmp-path path] [-bg-log-level level] [address]",
	Description: "Adapter command serve a database chain adapter, e.g. :7784",
}

func init() {
	CmdAdapter.Run = runAdapter

	addCommonFlags(CmdAdapter)
	addBgServerFlag(CmdAdapter)
}

func runAdapter(cmd *Command, args []string) {
	configInit()
	bgServerInit()

	if len(args) != 1 {
		ConsoleLog.Error("Adapter command need listern address as param")
		SetExitStatus(1)
		return
	}
	adapterAddr = args[0]

	adapterHTTPServer, err := adapter.NewHTTPAdapter(adapterAddr, configFile)
	if err != nil {
		ConsoleLog.WithError(err).Error("init adapter failed")
		SetExitStatus(1)
		return
	}

	if err = adapterHTTPServer.Serve(); err != nil {
		ConsoleLog.WithError(err).Error("start adapter failed")
		SetExitStatus(1)
		return
	}

	ConsoleLog.Infof("adapter started on %s", adapterAddr)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		adapterHTTPServer.Shutdown(ctx)
		ConsoleLog.Info("stopped adapter")
	}()

	ConsoleLog.Printf("Ctrl + C to stop adapter server on %s", adapterAddr)
	<-utils.WaitForExit()
}
