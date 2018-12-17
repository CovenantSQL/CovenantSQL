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
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const logo = `
   ______                                  __  _____ ____    __ 
  / ____/___ _   _____  ____  ____ _____  / /_/ ___// __ \  / / 
 / /   / __ \ | / / _ \/ __ \/ __  / __ \/ __/\__ \/ / / / / /
/ /___/ /_/ / |/ /  __/ / / / /_/ / / / / /_ ___/ / /_/ / / /___
\____/\____/|___/\___/_/ /_/\__,_/_/ /_/\__//____/\___\_\/_____/

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

	logLevel = log.DebugLevel
)

const name = `cqld`
const desc = `CovenantSQL is a Distributed Database running on BlockChain`

func init() {
	flag.BoolVar(&noLogo, "nologo", false, "Do not print logo")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&asymmetric.BypassSignature, "bypassSignature", false,
		"Disable signature sign and verify, for testing")
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
	log.Infof("%#v starting, version %#v, commit %#v, branch %#v", name, version, commit, branch)
	log.Infof("%#v, target architecture is %#v, operating system target is %#v", runtime.Version(), runtime.GOARCH, runtime.GOOS)
	log.Infof("role: %#v", conf.RoleTag)
}

func main() {
	// set random
	rand.Seed(time.Now().UnixNano())
	log.SetLevel(logLevel)
	flag.Parse()

	if showVersion {
		fmt.Printf("%v %v %v %v %v\n",
			name, version, runtime.GOOS, runtime.GOARCH, runtime.Version())
		os.Exit(0)
	}

	flag.Visit(func(f *flag.Flag) {
		log.Infof("Args %#v : %s", f.Name, f.Value)
	})

	var err error
	conf.GConf, err = conf.LoadConfig(configFile)
	if err != nil {
		log.WithField("config", configFile).WithError(err).Fatal("load config failed")
	}

	kms.InitBP()
	log.Debugf("config:\n%#v", conf.GConf)
	// BP DO NOT Generate new key pair
	conf.GConf.GenerateKeyPair = false

	// init log
	initLogs()

	if !noLogo {
		fmt.Print(logo)
	}

	// init profile, if cpuProfile, memProfile length is 0, nothing will be done
	utils.StartProfile(cpuProfile, memProfile)
	defer utils.StopProfile()

	if clientMode {
		if err := runClient(conf.GConf.ThisNodeID); err != nil {
			log.WithError(err).Fatal("run client failed")
		} else {
			log.Info("run client success")
		}
		return
	}

	if err := runNode(conf.GConf.ThisNodeID, conf.GConf.ListenAddr); err != nil {
		log.WithError(err).Fatal("run kayak failed")
	}

	log.Info("server stopped")
}
