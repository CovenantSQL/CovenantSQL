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
	"flag"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	version = "unknown"
	commit  = "unknown"
	branch  = "unknown"
)

var (
	// config
	configFile    string
	password      string
	listenAddr    string
	checkInterval time.Duration
)

func init() {
	flag.StringVar(&configFile, "config", "./config.yaml", "config file path")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:4665", "listen address for http explorer api")
	flag.DurationVar(&checkInterval, "interval", time.Second*2, "new block check interval for explorer")
	flag.StringVar(&password, "password", "", "master key password for covenantsql")
}

func main() {
	// set random
	rand.Seed(time.Now().UnixNano())
	log.SetLevel(log.DebugLevel)
	flag.Parse()
	flag.Visit(func(f *flag.Flag) {
		log.Infof("Args %s : %v", f.Name, f.Value)
	})

	// init client
	var err error
	if err = client.Init(configFile, []byte(password)); err != nil {
		log.Fatalf("init node failed: %v", err)
		return
	}

	// start service
	var service *Service
	if service, err = NewService(checkInterval); err != nil {
		log.Fatalf("init service failed: %v", err)
		return
	}

	// start api
	var httpServer *http.Server
	if httpServer, err = startAPI(service, listenAddr); err != nil {
		log.Fatalf("start explorer api failed: %v", err)
		return
	}

	// start subscription
	if err = service.start(); err != nil {
		log.Fatalf("start service failed: %v", err)
		return
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(
		signalCh,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	signal.Ignore(syscall.SIGHUP, syscall.SIGTTIN, syscall.SIGTTOU)

	<-signalCh

	// stop explorer api
	if err = stopAPI(httpServer); err != nil {
		log.Fatalf("stop explorer api failed: %v", err)
		return
	}

	// stop subscription
	if err = service.stop(); err != nil {
		log.Fatalf("stop service failed: %v", err)
		return
	}

	log.Info("explorer stopped")
}
