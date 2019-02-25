package jsonrpc

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
	wsstream "github.com/sourcegraph/jsonrpc2/websocket"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// WebsocketServer is a websocket server providing JSON-RPC API service.
type WebsocketServer struct {
	http.Server
	RPCHandler jsonrpc2.Handler
}

// Serve accepts incoming connections and serve each.
func (ws *WebsocketServer) Serve() error {
	var (
		mux      = http.NewServeMux()
		upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		handler  = ws.RPCHandler
	)

	if handler == nil {
		handler = defaultHandler
	}

	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			log.WithError(err).Error("jsonrpc: upgrade http connection to websocket failed")
			http.Error(rw, errors.WithMessage(err, "could not upgrade to websocket").Error(), http.StatusBadRequest)
			return
		}
		defer conn.Close()

		// TODO: add metric for the connections
		<-jsonrpc2.NewConn(
			context.Background(),
			wsstream.NewObjectStream(conn),
			handler,
		).DisconnectNotify()
	})

	addr := ws.Addr
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrapf(err, "couldn't bind to address %q", addr)
	}

	ws.Handler = mux
	return ws.Server.Serve(listener)
}

// Shutdown gracefully shuts down the server.
func (ws *WebsocketServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := ws.Server.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
