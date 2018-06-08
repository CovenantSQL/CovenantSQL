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
	"github.com/thunderdb/ThunderDB/common"
	"github.com/thunderdb/ThunderDB/conf"
	"github.com/thunderdb/ThunderDB/rpc"
	"github.com/thunderdb/ThunderDB/utils"
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
	minPort      int
	maxPort      int
	bindAddr     string
	publicAddr   string
	expvar       bool
	pprofEnabled bool
	dsn          string
	onDisk       bool

	// raft
	raftHeartbeatTimeout time.Duration
	raftApplyTimeout     time.Duration
	raftOpenTimeout      time.Duration
	raftSnapThreshold    uint64
	initPeers            string

	// api
	publishPeersTimeout time.Duration
	publishPeersDelay   time.Duration

	// profile
	cpuProfile string
	memProfile string

	// other
	noLogo      bool
	showVersion bool

	// rpc server
	rpcServer *rpc.Server
)

// Role indicates the role of current server playing
const Role = common.Miner

const name = `thunderdbd`
const desc = `ThunderDB is a database`

func init() {
	flag.BoolVar(&noLogo, "nologo", false, "Do not print logo")
	flag.IntVar(&minPort, "min-port", 10000, "Minimum port to allocate")
	flag.IntVar(&maxPort, "max-port", 20000, "Maximum port to allocate")
	flag.StringVar(&bindAddr, "bind-addr", "0.0.0.0", "Addr to bind in local interfaces")
	flag.StringVar(&publicAddr, "public-addr", "", "Addr to be accessed by internet")
	flag.BoolVar(&expvar, "expvar", true, "Serve expvar data on api server")
	flag.BoolVar(&pprofEnabled, "pprof", true, "Serve pprof data on api server")
	flag.StringVar(&dsn, "dsn", "", `SQLite DSN parameters. E.g. "cache=shared&mode=memory"`)
	flag.BoolVar(&onDisk, "on-disk", false, "Use an on-disk SQLite database")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.DurationVar(&raftHeartbeatTimeout, "raft-timeout", time.Second, "Raft heartbeat timeout")
	flag.DurationVar(&raftApplyTimeout, "raft-apply-timeout", time.Second*10, "Raft apply timeout")
	flag.DurationVar(&raftOpenTimeout, "raft-open-timeout", time.Second*120, "Time for initial Raft logs to be applied. Use 0s duration to skip wait")
	flag.Uint64Var(&raftSnapThreshold, "raft-snap", 8192, "Number of outstanding log entries that trigger snapshot")
	flag.DurationVar(&publishPeersTimeout, "publish-peers-timeout", time.Second*30, "Timeout for peers to publish")
	flag.DurationVar(&publishPeersDelay, "publish-peers-delay", time.Second, "Interval for peers publishing retry")
	flag.StringVar(&cpuProfile, "cpu-profile", "", "Path to file for CPU profiling information")
	flag.StringVar(&memProfile, "mem-profile", "", "Path to file for memory profiling information")
	flag.StringVar(&initPeers, "init-peers", "", "Init peers to join")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n%s\n\n", desc)
		fmt.Fprintf(os.Stderr, "Usage: %s [arguments] <data directory>\n", name)
		flag.PrintDefaults()
	}
}

func initLogs() {
	log.SetOutput(os.Stderr)
	log.SetLevel(log.InfoLevel)

	log.Infof("%s starting, version %s, commit %s, branch %s", name, version, commit, branch)
	log.Infof("%s, target architecture is %s, operating system target is %s", runtime.Version(), runtime.GOARCH, runtime.GOOS)
	log.Infof("role: %s", conf.Role)
}

func main() {
	flag.Parse()

	// init log
	initLogs()

	if showVersion {
		log.Info("%s %s %s %s %s (commit %s, branch %s)",
			name, version, runtime.GOOS, runtime.GOARCH, runtime.Version(), commit, branch)
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if !noLogo {
		fmt.Print(logo)
	}

	// validate args
	if minPort >= maxPort {
		log.Error("a valid port range is required")
		os.Exit(2)
	} else if maxPort-minPort < 1 {
		log.Error("at least two port in random port range is required")
		os.Exit(2)
	}

	// set random
	rand.Seed(time.Now().UnixNano())

	// init profile, if cpuProfile, memProfile length is 0, nothing will be done
	utils.StartProfile(cpuProfile, memProfile)
	defer utils.StopProfile()

	log.Info("server stopped")
}
