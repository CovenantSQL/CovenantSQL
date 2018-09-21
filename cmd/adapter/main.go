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
	"os"
	"os/signal"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"golang.org/x/sys/unix"
)

var (
	configFile string
)

func init() {
	flag.StringVar(&configFile, "config", "./config.yaml", "config file for adapter")
}

func main() {
	flag.Parse()

	server, err := NewHTTPAdapter(configFile)
	if err != nil {
		log.Fatalf("init adapter failed: %v", err)
		return
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, unix.SIGTERM)

	log.Infof("start adapter")
	if err = server.Serve(); err != nil {
		log.Fatalf("start adapter failed: %v", err)
		return
	}

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	server.Shutdown(ctx)
	log.Infof("stopped adapter")
}
