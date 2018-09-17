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
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"golang.org/x/sys/unix"
)

var (
	configFile string
	password   string
)

func init() {
	flag.StringVar(&configFile, "config", "config.yaml", "configuration file for covenantsql")
	flag.StringVar(&password, "password", "", "master key password for covenantsql")
}

func main() {
	// init client
	var err error
	if err = client.Init(configFile, []byte(password)); err != nil {
		log.Errorf("init covenantsql client failed: %v", err)
		os.Exit(-1)
		return
	}

	// load faucet config from same config file
	var cfg *Config

	if cfg, err = LoadConfig(configFile); err != nil {
		log.Errorf("read faucet config failed: %v", err)
		os.Exit(-1)
		return
	}

	// init persistence
	var p *Persistence
	if p, err = NewPersistence(cfg); err != nil {
		log.Errorf("")
		return
	}

	// init verifier
	var v *Verifier
	if v, err = NewVerifier(cfg, p); err != nil {
		return
	}

	// start verifier
	go v.run()

	// init faucet api
	var server *http.Server
	if server, err = startAPI(v, p, cfg.ListenAddr); err != nil {
		return
	}

	log.Info("started faucet")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, unix.SIGTERM)

	<-stop

	// stop verifier
	v.stop()

	// stop faucet api
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	server.Shutdown(ctx)
	log.Info("stopped faucet")
}
