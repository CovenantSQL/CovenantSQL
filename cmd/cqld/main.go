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
	"github.com/CovenantSQL/CovenantSQL/metric"
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
	metricWeb  string

	// other
	noLogo      bool
	showVersion bool
	configFile  string

	wsapiAddr string

	logLevel string
)

const name = `cqld`
const desc = `CovenantSQL is a Distributed Database running on BlockChain`

func init() {
	flag.BoolVar(&noLogo, "nologo", false, "Do not print logo")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&asymmetric.BypassSignature, "bypass-signature", false,
		"Disable signature sign and verify, for testing")
	flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file path")

	flag.StringVar(&cpuProfile, "cpu-profile", "", "Path to file for CPU profiling information")
	flag.StringVar(&memProfile, "mem-profile", "", "Path to file for memory profiling information")
	flag.StringVar(&metricWeb, "metric-web", "", "Address and port to get internal metrics")

	flag.StringVar(&wsapiAddr, "wsapi", "", "Address of the websocket JSON-RPC API, run as API Node")
	flag.StringVar(&logLevel, "log-level", "", "Service log level")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "\n%s\n\n", desc)
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s [arguments]\n", name)
		flag.PrintDefaults()
	}
}

func initLogs() {
	log.Infof("%#v starting, version %#v, commit %#v, branch %#v", name, version, commit, branch)
	log.Infof("%#v, target architecture is %#v, operating system target is %#v",
		runtime.Version(), runtime.GOARCH, runtime.GOOS)
	log.Infof("role: %#v", conf.RoleTag)
}

func main() {
	flag.Parse()
	log.SetStringLevel(logLevel, log.InfoLevel)
	// set random
	rand.Seed(time.Now().UnixNano())

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
	log.Debugf("config:\n%#v", conf.GConf)
	// BP Never Generate new key pair
	conf.GConf.GenerateKeyPair = false

	// init log
	initLogs()

	if !noLogo {
		fmt.Print(logo)
	}

	if len(metricWeb) > 0 {
		err = metric.InitMetricWeb(metricWeb)
		if err != nil {
			log.Errorf("start metric web server on %s failed: %v", metricWeb, err)
			os.Exit(-1)
		}
	}
	// init profile, if cpuProfile, memProfile length is 0, nothing will be done
	_ = utils.StartProfile(cpuProfile, memProfile)
	defer utils.StopProfile()

	if err := runNode(conf.GConf.ThisNodeID, conf.GConf.ListenAddr); err != nil {
		log.WithError(err).Fatal("run kayak failed")
	}

	log.Info("server stopped")
}
