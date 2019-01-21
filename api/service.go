package api

import (
	"net/http"
	"time"

	"github.com/CovenantSQL/CovenantSQL/api/models"
	"github.com/CovenantSQL/CovenantSQL/rpc/jsonrpc"
	"github.com/pkg/errors"
)

var (
	rpc    = jsonrpc.NewHandler()
	server *jsonrpc.WebsocketServer
)

// Serve runs an API server on the specified address and database file.
func Serve(addr, dbFile string) error {
	// setup database
	if err := models.InitModels(dbFile); err != nil {
		return errors.WithMessage(err, "api: init models failed")
	}

	server = &jsonrpc.WebsocketServer{
		Server: http.Server{
			Addr:         addr,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		RPCHandler: rpc,
	}

	return server.Serve()
}

// StopService stops the API server.
func StopService() {
	server.Stop()
}
