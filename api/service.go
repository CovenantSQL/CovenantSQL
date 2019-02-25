package api

import (
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/api/models"
	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/rpc/jsonrpc"
)

var (
	rpc    = jsonrpc.NewHandler()
	server *Service
)

func init() {
	server = &Service{
		WebsocketServer: jsonrpc.WebsocketServer{
			Server: http.Server{
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 60 * time.Second,
			},
		},
	}
	// Like handshake, the client must call this method after a successful connection.
	// Then it will be able to call methods with write permissions through the websocket proxy.
	rpc.RegisterMethod("__register", jsonrpc.RegisterClient, jsonrpc.RegisterClientParams{})
}

// Serve creates a Service and serve.
func Serve(addr, chainFile string, chain *bp.Chain) error {
	server.Addr = addr
	server.ChainDBFile = chainFile
	server.ChainRPC = bp.NewChainRPCService(chain)

	return server.Serve()
}

// Shutdown gracefully shuts down the API server.
func Shutdown() error {
	return server.Shutdown()
}

// Service is the API service.
type Service struct {
	jsonrpc.WebsocketServer
	ChainRPC    *bp.ChainRPCService
	ChainDBFile string
}

// Serve runs an API server on the specified address and database file.
func (s *Service) Serve() error {
	// setup database
	if err := models.InitModels(s.ChainDBFile); err != nil {
		return errors.WithMessage(err, "api: init models failed")
	}
	s.RPCHandler = rpc
	return s.WebsocketServer.Serve()
}
