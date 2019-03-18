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
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const name = "cql-faucet"

var (
	version     = "unknown"
	listenAddr  string
	configFile  string
	password    string
	showVersion bool
)

func init() {
	flag.StringVar(&listenAddr, "listen", "", "API listen addr (will override settings in config file")
	flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Configuration file for covenantsql")
	flag.StringVar(&password, "password", "", "Master key password for covenantsql")
	flag.BoolVar(&asymmetric.BypassSignature, "bypass-signature", false,
		"Disable signature sign and verify, for testing")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
}

func main() {
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
		log.WithError(err).Error("init covenantsql client failed")
		os.Exit(-1)
		return
	}

	// load faucet config from same config file
	var cfg *Config

	if cfg, err = LoadConfig(listenAddr, configFile); err != nil {
		log.WithError(err).Error("read faucet config failed")
		os.Exit(-1)
		return
	}

	// init persistence
	var p *Persistence
	if p, err = NewPersistence(cfg); err != nil {
		log.WithError(err).Error("init persistence storage failed")
		return
	}

	// init faucet api
	var server *http.Server
	if server, err = startAPI(p, cfg.ListenAddr); err != nil {
		return
	}

	log.Info("started faucet")

	<-utils.WaitForExit()

	// stop faucet api
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	server.Shutdown(ctx)
	log.Info("stopped faucet")
}
