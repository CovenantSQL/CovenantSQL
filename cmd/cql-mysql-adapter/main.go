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
	"os"
	"os/signal"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"golang.org/x/sys/unix"
)

var (
	configFile string
	password   string

	listenAddr    string
	mysqlUser     string
	mysqlPassword string
)

func init() {
	flag.StringVar(&configFile, "config", "./config.yaml", "config file for mysql adapter")
	flag.StringVar(&password, "password", "", "master key password")
	flag.BoolVar(&asymmetric.BypassSignature, "bypassSignature", false,
		"Disable signature sign and verify, for testing")

	flag.StringVar(&listenAddr, "listen", "127.0.0.1:4664", "listen address for mysql adapter")
	flag.StringVar(&mysqlUser, "mysql-user", "root", "mysql user for adapter server")
	flag.StringVar(&mysqlPassword, "mysql-password", "calvin", "mysql password for adapter server")
}

func main() {
	flag.Parse()
	flag.Visit(func(f *flag.Flag) {
		log.Infof("Args %#v : %#v", f.Name, f.Value)
	})

	// init client
	if err := client.Init(configFile, []byte(password)); err != nil {
		log.WithError(err).Fatal("init covenantsql client failed")
		return
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, unix.SIGTERM)

	server, err := NewServer(listenAddr, mysqlUser, mysqlPassword)
	if err != nil {
		log.WithError(err).Fatal("init server failed")
		return
	}

	go server.Serve()

	log.Info("start mysql adapter")

	<-stop

	server.Shutdown()

	log.Info("stopped mysql adapter")
}
