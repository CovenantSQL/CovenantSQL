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
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"

	graphite "github.com/cyberdelia/go-metrics-graphite"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/metric"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	_ "github.com/CovenantSQL/CovenantSQL/utils/log/debug"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
	"github.com/CovenantSQL/CovenantSQL/worker"
)

const logo = `
   ______                                  __  _____ ____    __ 
  / ____/___ _   _____  ____  ____ _____  / /_/ ___// __ \  / / 
 / /   / __ \ | / / _ \/ __ \/ __  / __ \/ __/\__ \/ / / / / /
/ /___/ /_/ / |/ /  __/ / / / /_/ / / / / /_ ___/ / /_/ / / /___
\____/\____/|___/\___/_/ /_/\__,_/_/ /_/\__//____/\___\_\/_____/

   _____ ______   ___  ________   _______   ________     
  |\   _ \  _   \|\  \|\   ___  \|\  ___ \ |\   __  \    
  \ \  \\\__\ \  \ \  \ \  \\ \  \ \   __/|\ \  \|\  \   
   \ \  \\|__| \  \ \  \ \  \\ \  \ \  \_|/_\ \   _  _\  
    \ \  \    \ \  \ \  \ \  \\ \  \ \  \_|\ \ \  \\  \| 
     \ \__\    \ \__\ \__\ \__\\ \__\ \_______\ \__\\ _\ 
      \|__|     \|__|\|__|\|__| \|__|\|_______|\|__|\|__|
                                                       
`

var (
	version = "unknown"
)

var (
	// config
	configFile string
	genKeyPair bool
	metricLog  bool
	metricWeb  string

	// profile
	cpuProfile     string
	memProfile     string
	profileServer  string
	metricGraphite string
	traceFile      string

	// other
	noLogo      bool
	showVersion bool
	logLevel    string
)

const name = `cql-minerd`
const desc = `CovenantSQL is a Distributed Database running on BlockChain`

func init() {
	flag.BoolVar(&noLogo, "nologo", false, "Do not print logo")
	flag.BoolVar(&metricLog, "metric-log", false, "Print metrics in log")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.BoolVar(&genKeyPair, "gen-keypair", false, "Gen new key pair when no private key found")
	flag.BoolVar(&asymmetric.BypassSignature, "bypass-signature", false,
		"Disable signature sign and verify, for testing")

	flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file path")

	flag.StringVar(&profileServer, "profile-server", "", "Profile server address, default not started")
	flag.StringVar(&cpuProfile, "cpu-profile", "", "Path to file for CPU profiling information")
	flag.StringVar(&memProfile, "mem-profile", "", "Path to file for memory profiling information")
	flag.StringVar(&metricGraphite, "metric-graphite-server", "", "Metric graphite server to push metrics")
	flag.StringVar(&metricWeb, "metric-web", "", "Address and port to get internal metrics")

	flag.StringVar(&traceFile, "trace-file", "", "Trace profile")
	flag.StringVar(&logLevel, "log-level", "", "Service log level")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "\n%s\n\n", desc)
		_, _ = fmt.Fprintf(os.Stderr, "Usage: %s [arguments]\n", name)
		flag.PrintDefaults()
	}
}

func initLogs() {
	log.Infof("%#v starting, version %#v", name, version)
	log.Infof("%#v, target architecture is %#v, operating system target is %#v", runtime.Version(), runtime.GOARCH, runtime.GOOS)
	log.Infof("role: %#v", conf.RoleTag)
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

	if conf.GConf.Miner == nil {
		log.Fatal("miner config does not exists")
	}
	if conf.GConf.Miner.ProvideServiceInterval.Seconds() <= 0 {
		log.Fatal("miner metric collect interval is invalid")
	}
	if conf.GConf.Miner.MaxReqTimeGap.Seconds() <= 0 {
		log.Fatal("miner request time gap is invalid")
	}

	kms.InitBP()
	log.Debugf("config:\n%#v", conf.GConf)

	// init log
	initLogs()

	if !noLogo {
		fmt.Print(logo)
	}

	// init profile, if cpuProfile, memProfile length is 0, nothing will be done
	_ = utils.StartProfile(cpuProfile, memProfile)

	// set generate key pair config
	conf.GConf.GenerateKeyPair = genKeyPair

	// start rpc
	var (
		server *mux.Server
		direct *rpc.Server
	)
	if server, direct, err = initNode(); err != nil {
		log.WithError(err).Fatal("init node failed")
	}

	// stop channel for all daemon routines
	stopCh := make(chan struct{})
	defer close(stopCh)

	if len(profileServer) > 0 {
		go func() {
			log.Println(http.ListenAndServe(profileServer, nil))
		}()
	}

	if len(metricWeb) > 0 {
		err = metric.InitMetricWeb(metricWeb)
		if err != nil {
			log.Errorf("start metric web server on %s failed: %v", metricWeb, err)
			os.Exit(-1)
		}
	}

	// start prometheus collector
	reg := metric.StartMetricCollector()

	// start period provide service transaction generator
	go func() {
		tick := time.NewTicker(conf.GConf.Miner.ProvideServiceInterval)
		defer tick.Stop()

		for {
			sendProvideService(reg)

			select {
			case <-stopCh:
				return
			case <-tick.C:
			}
		}
	}()

	// start dbms
	var dbms *worker.DBMS
	if dbms, err = startDBMS(server, direct, func() {
		sendProvideService(reg)
	}); err != nil {
		log.WithError(err).Fatal("start dbms failed")
	}

	defer dbms.Shutdown()

	// start rpc server
	go func() {
		server.Serve()
	}()
	defer server.Stop()

	// start direct rpc server
	if direct != nil {
		go func() {
			direct.Serve()
		}()
		defer direct.Stop()
	}

	if metricLog {
		go metrics.Log(metrics.DefaultRegistry, 5*time.Second, log.StandardLogger())
	}

	if metricGraphite != "" {
		addr, err := net.ResolveTCPAddr("tcp", metricGraphite)
		if err != nil {
			log.WithError(err).Error("resolve metric graphite server addr failed")
			return
		}
		minerName := fmt.Sprintf("miner-%s", conf.GConf.ThisNodeID[len(conf.GConf.ThisNodeID)-5:])
		go graphite.Graphite(metrics.DefaultRegistry, 5*time.Second, minerName, addr)
	}

	if traceFile != "" {
		f, err := os.Create(traceFile)
		if err != nil {
			log.WithError(err).Fatal("failed to create trace output file")
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.WithError(err).Fatal("failed to close trace file")
			}
		}()

		if err := trace.Start(f); err != nil {
			log.WithError(err).Fatal("failed to start trace")
		}
		defer trace.Stop()
	}

	<-utils.WaitForExit()
	utils.StopProfile()

	log.Info("miner stopped")
}
