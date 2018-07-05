/*
 * Copyright 2018 The ThunderDB Authors.
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

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/common"
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/utils"
)

const logo = `
 _______ _                     _           _____  ____  
|__   __| |                   | |         |  __ \|  _ \ 
   | |  | |__  _   _ _ __   __| | ___ _ __| |  | | |_) |
   | |  | '_ \| | | | '_ \ / _| |/ _ \ '__| |  | |  _ <
   | |  | | | | |_| | | | | (_| |  __/ |  | |__| | |_) |
   |_|  |_| |_|\__,_|_| |_|\__,_|\___|_|  |_____/|____/

  _____ ______   ___  ________   _______   ________     
  |\   _ \  _   \|\  \|\   ___  \|\  ___ \ |\   __  \    
  \ \  \\\__\ \  \ \  \ \  \\ \  \ \   __/|\ \  \|\  \   
   \ \  \\|__| \  \ \  \ \  \\ \  \ \  \_|/_\ \   _  _\  
    \ \  \    \ \  \ \  \ \  \\ \  \ \  \_|\ \ \  \\  \| 
     \ \__\    \ \__\ \__\ \__\\ \__\ \_______\ \__\\ _\ 
      \|__|     \|__|\|__|\|__| \|__|\|_______|\|__|\|__|
                                                       
`

var (
	version = "1"
	commit  = "unknown"
	branch  = "unknown"
)

var (
	// database
	minPort  int
	maxPort  int
	bindAddr string

	// config
	configFile string

	// profile
	cpuProfile string
	memProfile string

	// other
	noLogo      bool
	showVersion bool
)

// Role indicates the role of current server playing
const Role = common.Miner

const name = `thunderminerd`
const desc = `ThunderDB is a Distributed Database running on BlockChain`

func init() {
	flag.BoolVar(&noLogo, "nologo", false, "Do not print logo")
	flag.StringVar(&bindAddr, "bind-addr", "0.0.0.0:6699", "Addr and port to bind, In honor of Napster")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.StringVar(&configFile, "config", "./config.yaml", "Config file path")

	flag.StringVar(&cpuProfile, "cpu-profile", "", "Path to file for CPU profiling information")
	flag.StringVar(&memProfile, "mem-profile", "", "Path to file for memory profiling information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n%s\n\n", desc)
		fmt.Fprintf(os.Stderr, "Usage: %s [arguments]\n", name)
		flag.PrintDefaults()
	}
}

func initLogs() {
	log.Infof("%s starting, version %s, commit %s, branch %s", name, version, commit, branch)
	log.Infof("%s, target architecture is %s, operating system target is %s", runtime.Version(), runtime.GOARCH, runtime.GOOS)
	log.Infof("role: %s", conf.Role)
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
	log.Debugf("config:\n%#v", conf.GConf)

	// init log
	initLogs()

	if showVersion {
		log.Infof("%s %s %s %s %s (commit %s, branch %s)",
			name, version, runtime.GOOS, runtime.GOARCH, runtime.Version(), commit, branch)
		os.Exit(0)
	}

	if !noLogo {
		fmt.Print(logo)
	}

	// init profile, if cpuProfile, memProfile length is 0, nothing will be done
	utils.StartProfile(cpuProfile, memProfile)
	defer utils.StopProfile()

	//reg := metric.StartMetricCollector()

	log.Info("miner stopped")
}
