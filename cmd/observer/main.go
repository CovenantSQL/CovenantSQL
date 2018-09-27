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
	"time"

	"os"
	"os/signal"
	"syscall"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
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
	dbID          string
	listenAddr    string
	resetPosition string
)

func init() {
	flag.StringVar(&configFile, "config", "./config.yaml", "Config file path")
	flag.StringVar(&dbID, "database", "", "database to listen for observation")
	flag.StringVar(&resetPosition, "reset", "", "reset subscribe position")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:4663", "listen address for http explorer api")
}

func main() {
	// set random
	rand.Seed(time.Now().UnixNano())
	log.SetLevel(log.DebugLevel)
	flag.Parse()

	var err error
	conf.GConf, err = conf.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("load config from %s failed: %s", configFile, err)
	}

	kms.InitBP()

	// start rpc
	var server *rpc.Server
	if server, err = initNode(); err != nil {
		log.Fatalf("init node failed: %v", err)
	}

	// start service
	var service *Service
	if service, err = startService(server); err != nil {
		log.Fatalf("start observation failed: %v", err)
	}

	// start explorer api
	httpServer, err := startAPI(service, listenAddr)
	if err != nil {
		log.Fatalf("start explorer api failed: %v", err)
	}

	// register node
	if err = registerNode(); err != nil {
		log.Fatalf("register node failed: %v", err)
	}

	// start subscription
	var cfg *Config
	if cfg, err = loadConfig(configFile); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if cfg != nil {
		for _, v := range cfg.Databases {
			if err = service.subscribe(proto.DatabaseID(v.ID), v.Position); err != nil {
				log.Fatalf("init subscription failed: %v", err)
			}
		}
	}
	// Process command arguments after config file so that you can reset subscribe on startup
	// without changing the config.
	if dbID != "" {
		if err = service.subscribe(proto.DatabaseID(dbID), resetPosition); err != nil {
			log.Fatalf("init subscription failed: %v", err)
		}
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
	}

	// stop subscriptions
	if err = stopService(service, server); err != nil {
		log.Fatalf("stop service failed: %v", err)
	}

	log.Info("observer stopped")
}
