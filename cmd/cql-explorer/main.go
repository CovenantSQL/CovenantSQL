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
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const name = "cql-explorer"

var (
	version = "unknown"
)

var (
	// config
	configFile    string
	password      string
	listenAddr    string
	checkInterval time.Duration
	showVersion   bool
)

func init() {
	flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file path")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:4665", "Listen address for http explorer api")
	flag.DurationVar(&checkInterval, "interval", time.Second*2, "New block check interval for explorer")
	flag.StringVar(&password, "password", "", "Master key password for covenantsql")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
}

func main() {
	// set random
	rand.Seed(time.Now().UnixNano())
	log.SetLevel(log.DebugLevel)
	flag.Parse()
	if showVersion {
		fmt.Printf("%v %v %v %v %v\n",
			name, version, runtime.GOOS, runtime.GOARCH, runtime.Version())
		os.Exit(0)
	}

	configFile = utils.HomeDirExpand(configFile)

	flag.Visit(func(f *flag.Flag) {
		log.Infof("args %#v : %s", f.Name, f.Value)
	})

	// init client
	var err error
	if err = client.Init(configFile, []byte(password)); err != nil {
		log.WithError(err).Fatal("init node failed")
		return
	}

	// start service
	var service *Service
	if service, err = NewService(checkInterval); err != nil {
		log.WithError(err).Fatal("init service failed")
		return
	}

	// start api
	var httpServer *http.Server
	if httpServer, err = startAPI(service, listenAddr); err != nil {
		log.WithError(err).Fatal("start explorer api failed")
		return
	}

	// start subscription
	if err = service.start(); err != nil {
		log.WithError(err).Fatal("start service failed")
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
		log.WithError(err).Fatal("stop explorer api failed")
		return
	}

	// stop subscription
	if err = service.stop(); err != nil {
		log.WithError(err).Fatal("stop service failed")
		return
	}

	log.Info("explorer stopped")
}
