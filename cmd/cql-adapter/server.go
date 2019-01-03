/*
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
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql-adapter/api"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-adapter/config"
	"github.com/gorilla/handlers"
)

// HTTPAdapter is a adapter for covenantsql/alternative sqlite3 service.
type HTTPAdapter struct {
	server *http.Server
}

// NewHTTPAdapter creates adapter to service.
func NewHTTPAdapter(configFile string, password string) (adapter *HTTPAdapter, err error) {
	adapter = new(HTTPAdapter)

	// load config file
	var cfg *config.Config
	if cfg, err = config.LoadConfig(configFile, password); err != nil {
		return
	}

	// init server
	handler := handlers.CORS()(api.GetRouter())

	adapter.server = &http.Server{
		TLSConfig: cfg.TLSConfig,
		Addr:      cfg.ListenAddr,
		Handler:   handler,
	}

	return
}

// Serve defines adapter serve logic.
func (adapter *HTTPAdapter) Serve() (err error) {
	// start server
	cfg := config.GetConfig()

	// bind port, start tls listener
	var listener net.Listener
	if listener, err = net.Listen("tcp", cfg.ListenAddr); err != nil {
		return
	}

	if cfg.TLSConfig != nil {
		listener = tls.NewListener(listener, cfg.TLSConfig)
	}

	// serve the connection
	go adapter.server.Serve(listener)

	return
}

// Shutdown shutdown the service.
func (adapter *HTTPAdapter) Shutdown(ctx context.Context) {
	if adapter.server != nil {
		adapter.server.Shutdown(ctx)
	}
}
