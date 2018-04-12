package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	log "github.com/sirupsen/logrus"
	"runtime/pprof"
	"github.com/thunderdb/ThunderDB/utils"
	"math/rand"
	"time"
)

const logo = `
 _____ _                     _           ____  ____
|_   _| |__  _   _ _ __   __| | ___ _ __|  _ \| __ )
  | | | '_ \| | | | '_ \ / _' |/ _ \ '__| | | |  _ \
  | | | | | | |_| | | | | (_| |  __/ |  | |_| | |_) |
  |_| |_| |_|\__,_|_| |_|\__,_|\___|_|  |____/|____/

`

var (
	version   = "1"
	commit    = "unknown"
	branch    = "unknown"
	buildtime = "unknown"
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

	fmt.Println(apiPort, raftPort, dataPath)
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
