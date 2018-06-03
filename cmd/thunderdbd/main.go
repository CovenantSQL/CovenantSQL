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

	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/common"
	"github.com/thunderdb/ThunderDB/conf"
	"github.com/thunderdb/ThunderDB/rpc"
	"github.com/thunderdb/ThunderDB/utils"
	"golang.org/x/crypto/ssh/terminal"
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

	// private key path
	privateKeyPath string

	// other
	noLogo      bool
	showVersion bool

	// rpc server
	rpcServer *rpc.Server
)

// Role specifies the role of this app, which can be "miner", "blockProducer"
const Role = common.BlockProducer

const name = `thunderdbd`
const desc = `ThunderDB is a database`

func init() {
	flag.BoolVar(&noLogo, "nologo", false, "Do not print logo")
	flag.BoolVar(&showVersion, "version", false, "Show version information and exit")
	flag.DurationVar(&raftHeartbeatTimeout, "raft-timeout", time.Second, "Raft heartbeat timeout")
	flag.DurationVar(&raftApplyTimeout, "raft-apply-timeout", time.Second*10, "Raft apply timeout")
	flag.DurationVar(&raftOpenTimeout, "raft-open-timeout", time.Second*120, "Time for initial Raft logs to be applied. Use 0s duration to skip wait")
	flag.Uint64Var(&raftSnapThreshold, "raft-snap", 8192, "Number of outstanding log entries that trigger snapshot")
	flag.DurationVar(&publishPeersTimeout, "publish-peers-timeout", time.Second*30, "Timeout for peers to publish")
	flag.DurationVar(&publishPeersDelay, "publish-peers-delay", time.Second, "Interval for peers publishing retry")
	flag.StringVar(&privateKeyPath, "private-key-path", "./private.key", "Path to private key file")
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
		log.Infof("%s %s %s %s %s (commit %s, branch %s)",
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

	// set random
	rand.Seed(time.Now().UnixNano())

	// init profile, if cpuProfile, memProfile length is 0, nothing will be done
	utils.StartProfile(cpuProfile, memProfile)
	defer utils.StopProfile()

	// read master key
	fmt.Print("Type in Master key to continue: ")
	masterKeyBytes, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("Failed to read Master Key: %v", err)
	}

	// start RPC server
	rpcServer := rpc.NewServer()
	err = rpcServer.InitRPCServer("0.0.0.0:2120", privateKeyPath, masterKeyBytes)
	if err != nil {
		log.Fatalf("init RPC server failed: %s", err)
	}
	go rpcServer.Serve()

	log.Info("server stopped")
}
