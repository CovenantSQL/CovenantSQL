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
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const name = "cql-observer"

var (
	version = "unknown"

	// config
	configFile    string
	dbID          string
	listenAddr    string
	resetPosition string
	showVersion   bool
	logLevel      string
)

func init() {
	flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file path")
	flag.StringVar(&dbID, "database", "", "Database to listen for observation")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&asymmetric.BypassSignature, "bypass-signature", false,
		"Disable signature sign and verify, for testing")
	flag.StringVar(&resetPosition, "reset", "", "Reset subscribe position")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:4663", "Listen address for http explorer api")
	flag.StringVar(&logLevel, "log-level", "", "Service log level")
}

func main() {
	flag.Parse()
	// set random
	rand.Seed(time.Now().UnixNano())
	log.SetStringLevel(logLevel, log.InfoLevel)
	if showVersion {
		fmt.Printf("%v %v %v %v %v\n",
			name, version, runtime.GOOS, runtime.GOARCH, runtime.Version())
		os.Exit(0)
	}

	configFile = utils.HomeDirExpand(configFile)

	flag.Visit(func(f *flag.Flag) {
		log.Infof("args %#v : %s", f.Name, f.Value)
	})

	var err error
	conf.GConf, err = conf.LoadConfig(configFile)
	if err != nil {
		log.WithField("config", configFile).WithError(err).Fatal("load config failed")
	}

	kms.InitBP()

	// init node
	if err = initNode(); err != nil {
		log.WithError(err).Fatal("init node failed")
	}

	// start service
	var service *Service
	if service, err = startService(); err != nil {
		log.WithError(err).Fatal("start observation failed")
	}

	// start explorer api
	httpServer, err := startAPI(service, listenAddr)
	if err != nil {
		log.WithError(err).Fatal("start explorer api failed")
	}

	// register node
	if err = registerNode(); err != nil {
		log.WithError(err).Fatal("register node failed")
	}

	// start subscription
	var cfg *Config
	if cfg, err = loadConfig(configFile); err != nil {
		log.WithError(err).Fatal("failed to load config")
	}
	if cfg != nil {
		for _, v := range cfg.Databases {
			if err = service.subscribe(proto.DatabaseID(v.ID), v.Position); err != nil {
				log.WithError(err).Fatal("init subscription failed")
			}
		}
	}
	// Process command arguments after config file so that you can reset subscribe on startup
	// without changing the config.
	if dbID != "" {
		if err = service.subscribe(proto.DatabaseID(dbID), resetPosition); err != nil {
			log.WithError(err).Fatal("init subscription failed")
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
		log.WithError(err).Fatal("stop explorer api failed")
	}

	// stop subscriptions
	if err = stopService(service); err != nil {
		log.WithError(err).Fatal("stop service failed")
	}

	log.Info("observer stopped")
}
