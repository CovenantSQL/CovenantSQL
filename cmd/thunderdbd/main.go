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
	"runtime/pprof"
	"time"

	"os/signal"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/server"
	"github.com/thunderdb/ThunderDB/utils"
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
	// database
	minPort      int
	maxPort      int
	bindAddr     string
	publicAddr   string
	expvar       bool
	pprofEnabled bool
	dsn          string
	onDisk       bool
	showVersion  bool

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
	noLogo bool
)

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
}

func initPorts() (apiPort int, raftPort int) {
	ports, err := utils.GetRandomPorts(bindAddr, minPort, maxPort, 2)

	if err != nil {
		log.Fatalf("could not get enough ports in port range [%v, %v]: %s\n",
			minPort, maxPort, err.Error())
		os.Exit(3)
	}

	apiPort = ports[0]
	raftPort = ports[1]

	return
}

func main() {
	flag.Parse()

	// init log
	initLogs()

	if showVersion {
		log.Fatalf("%s %s %s %s %s (commit %s, branch %s)",
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
		log.Fatal("a valid port range is required")
		os.Exit(2)
	} else if maxPort-minPort < 1 {
		log.Fatal("at least two port in random port range is required")
		os.Exit(2)
	}

	// set random
	rand.Seed(time.Now().UnixNano())

	// init profile
	startProfile(cpuProfile, memProfile)
	defer stopProfile()

	// random ports
	apiPort, raftPort := initPorts()

	// init db
	dataPath := flag.Arg(0)

	// peers to join
	var peerAddrs []string

	if initPeers != "" {
		peerAddrs = strings.Split(initPeers, ",")
	}

	s, err := server.NewService(&server.ServiceConfig{
		BindAddr:              bindAddr,
		PublicAddr:            publicAddr,
		ApiPort:               apiPort,
		RaftPort:              raftPort,
		InitPeers:             peerAddrs,
		DataPath:              dataPath,
		DSN:                   dsn,
		OnDisk:                onDisk,
		RaftSnapshotThreshold: raftSnapThreshold,
		RaftHeartbeatTimeout:  raftHeartbeatTimeout,
		RaftApplyTimeout:      raftApplyTimeout,
		RaftOpenTimeout:       raftOpenTimeout,
		PublishPeersTimeout:   publishPeersTimeout,
		PublishPeersDelay:     publishPeersDelay,
		EnablePprof:           pprofEnabled,
		Expvar:                expvar,
		BuildInfo: map[string]interface{}{
			"version": version,
			"commit":  commit,
			"branch":  branch,
		},
	})

	if err != nil {
		log.Fatalf("init service failed: %s", err.Error())
		os.Exit(4)
	}

	log.Info("server initialized")

	if err = s.Serve(); err != nil {
		log.Fatalf("start service failed: %s", err.Error())
		os.Exit(5)
	}

	log.Info("server started")

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate

	if err := s.Close(); err != nil {
		log.Fatalf("stop service failed: %s", err.Error())
		os.Exit(6)
	}

	log.Info("server stopped")
}

var prof struct {
	cpu *os.File
	mem *os.File
}

// startProfile initializes the CPU and memory profile, if specified.
func startProfile(cpuprofile, memprofile string) {
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatalf("failed to create CPU profile file at %s: %s", cpuprofile, err.Error())
		}
		log.Infof("writing CPU profile to: %s\n", cpuprofile)
		prof.cpu = f
		pprof.StartCPUProfile(prof.cpu)
	}

	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatalf("failed to create memory profile file at %s: %s", cpuprofile, err.Error())
		}
		log.Infof("writing memory profile to: %s\n", memprofile)
		prof.mem = f
		runtime.MemProfileRate = 4096
	}
}

// stopProfile closes the CPU and memory profiles if they are running.
func stopProfile() {
	if prof.cpu != nil {
		pprof.StopCPUProfile()
		prof.cpu.Close()
		log.Infof("CPU profiling stopped")
	}
	if prof.mem != nil {
		pprof.Lookup("heap").WriteTo(prof.mem, 0)
		prof.mem.Close()
		log.Infof("memory profiling stopped")
	}
}
