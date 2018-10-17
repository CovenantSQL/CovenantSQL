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
	"time"
)

var (
	version = "unknown"
	commit  = "unknown"
	branch  = "unknown"
)

var (
	// config
	configFile    string
	listenAddr    string
	checkInterval time.Duration
)

func init() {
	flag.StringVar(&configFile, "config", "./config.yaml", "config file path")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:4665", "listen address for http explorer api")
	flag.DurationVar(&checkInterval, "interval", time.Second*2, "new block check interval for explorer")
}

func main() {

}
