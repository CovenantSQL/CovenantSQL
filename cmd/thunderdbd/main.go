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

	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/utils"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

const logo = `
 _____ _                     _           ____  ____
|_   _| |__  _   _ _ __   __| | ___ _ __|  _ \| __ )
  | | | '_ \| | | | '_ \ / _' |/ _ \ '__| | | |  _ \
  | | | | | | |_| | | | | (_| |  __/ |  | |_| | |_) |
  |_| |_| |_|\__,_|_| |_|\__,_|\___|_|  |____/|____/

`

var (
	version = "1"
	commit  = "unknown"
	branch  = "unknown"
)

var (
	// profile
	cpuProfile string
	memProfile string

	// other
	noLogo      bool
	showVersion bool
	configFile  string
	genKeyPair  bool

	clientMode      bool
	clientOperation string
)

const name = `thunderdbd`
const desc = `ThunderDB is a Distributed Database running on BlockChain`

func init() {
	flag.BoolVar(&noLogo, "nologo", false, "Do not print logo")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&genKeyPair, "genKeyPair", false, "Gen new key pair when no private key found")
	flag.StringVar(&configFile, "config", "./config.yaml", "Config file path")

	flag.StringVar(&cpuProfile, "cpu-profile", "", "Path to file for CPU profiling information")
	flag.StringVar(&memProfile, "mem-profile", "", "Path to file for memory profiling information")

	flag.BoolVar(&clientMode, "client", false, "run as client")
	flag.StringVar(&clientOperation, "operation", "FindNeighbor", "client operation")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n%s\n\n", desc)
		fmt.Fprintf(os.Stderr, "Usage: %s [arguments]\n", name)
		flag.PrintDefaults()
	}
}

func initLogs() {
	log.Infof("%s starting, version %s, commit %s, branch %s", name, version, commit, branch)
	log.Infof("%s, target architecture is %s, operating system target is %s", runtime.Version(), runtime.GOARCH, runtime.GOOS)
	log.Infof("role: %s", conf.RoleTag)
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
	log.Debugf("config:\n%#v", conf.GConf)
	conf.GConf.GenerateKeyPair = genKeyPair

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

	if clientMode {
		if err := runClient(conf.GConf.ThisNodeID); err != nil {
			log.Fatalf("run client failed: %v", err.Error())
		} else {
			log.Infof("run client success")
		}
		return
	}

	if err := runNode(conf.GConf.ThisNodeID, conf.GConf.ListenAddr); err != nil {
		log.Fatalf("run kayak failed: %v", err.Error())
	}

	log.Info("server stopped")
}
