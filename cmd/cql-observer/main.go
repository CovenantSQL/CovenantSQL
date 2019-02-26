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
	"runtime"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/sqlchain/observer"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const name = "cql-observer"

var (
	version = "unknown"

	// config
	configFile  string
	listenAddr  string
	showVersion bool
	logLevel    string
)

func init() {
	flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file path")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&asymmetric.BypassSignature, "bypass-signature", false,
		"Disable signature sign and verify, for testing")
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

	if err = initNode(); err != nil {
		return
	}

	service, httpServer, err := observer.StartObserver(listenAddr, version)
	if err != nil {
		log.WithError(err).Fatal("start observer failed")
	}

	<-utils.WaitForExit()

	_ = observer.StopObserver(service, httpServer)
	log.Info("observer stopped")

	return
}
