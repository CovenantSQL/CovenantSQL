package api

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/CovenantSQL/CovenantSQL/api/models"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sourcegraph/jsonrpc2"
	wsstream "github.com/sourcegraph/jsonrpc2/websocket"
)

// Service configs the API service.
type Service struct {
	WebsocketAddr string // start a websocket server
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration

	stopChan chan struct{}
}

// StartServers start API servers in a non-blocking way, fatal on errors.
func (s *Service) StartServers() {
	go s.RunServers()
}

// StopServers top API servers.
func (s *Service) StopServers() {
	close(s.stopChan)
}

// RunServers start API servers in a blocking way, fatal on errors.
func (s *Service) RunServers() {
	// setup database
	if err := models.InitModels(); err != nil {
		log.WithError(err).Fatal("api: init models failed")
		return
	}

	s.stopChan = make(chan struct{})
	wg := sync.WaitGroup{}

	if s.WebsocketAddr != "" {
		log.WithField("addr", s.WebsocketAddr).Info("api: start websocket server")
		wg.Add(1)
		go s.runWebsocketServer(&wg)
	}

	sigchan := make(chan os.Signal)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	<-sigchan
	close(s.stopChan)
	wg.Wait()
}

func (s *Service) runWebsocketServer(wg *sync.WaitGroup) {
	defer wg.Done()

	var connOpts []jsonrpc2.ConnOpt

	mux := http.NewServeMux()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			log.WithError(err).Error("api: upgrade http connection to websocket failed")
			http.Error(rw, errors.WithMessage(err, "could not upgrade to websocket").Error(), http.StatusBadRequest)
			return
		}
		defer conn.Close()

		// TODO: add metric for the connections
		log.Debug("received incoming connection")
		<-jsonrpc2.NewConn(
			context.Background(),
			wsstream.NewObjectStream(conn),
			jsonrpcHandler.Handler(),
			connOpts...,
		).DisconnectNotify()
		log.Debug("connection closed")
	})

	addr := s.WebsocketAddr
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.WithField("addr", addr).WithError(err).Fatal("api: couldn't bind to address")
		return
	}

	httpServer := &http.Server{
		Handler:      mux,
		ReadTimeout:  s.ReadTimeout,
		WriteTimeout: s.WriteTimeout,
	}

	go func() {
		if err := httpServer.Serve(listener); err != nil {
			log.WithError(err).Error("api: websocket server serve error")
		}
	}()

	<-s.stopChan

	log.Warn("api: shutdown websocket server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := httpServer.Shutdown(ctx); err != nil {
		log.WithError(err).Error("shutdown server")
	}
	cancel()
	log.Warn("api: websocket server stopped")
}
