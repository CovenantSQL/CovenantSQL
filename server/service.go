package server

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rqlite/rqlite/cluster"
	httpd "github.com/rqlite/rqlite/http"
	"github.com/rqlite/rqlite/store"
	"github.com/rqlite/rqlite/tcp"
	log "github.com/sirupsen/logrus"
)

const (
	MuxRaftHeader = 1 // Raft consensus communications
	MuxMetaHeader = 2 // Cluster meta communications
)

type ServiceConfig struct {
	// ports, addrs
	BindAddr   string
	PublicAddr string
	ApiPort    int
	RaftPort   int
	InitPeers  []string

	// database
	DataPath string
	DSN      string
	OnDisk   bool

	// raft config
	RaftSnapshotThreshold uint64
	RaftHeartbeatTimeout  time.Duration
	RaftApplyTimeout      time.Duration
	RaftOpenTimeout       time.Duration

	// api server config
	PublishPeersDelay   time.Duration
	PublishPeersTimeout time.Duration
	EnablePprof         bool
	Expvar              bool
	BuildInfo           map[string]interface{}
}

type Service struct {
	Conf *ServiceConfig

	// database
	DataPath       string
	NodeId         string
	DBConfig       *store.DBConfig
	DBStore        *store.Store
	ClusterService *cluster.Service

	// raft/api server handlers
	RaftListener net.Listener
	RaftMux      *tcp.Mux
	ApiServer    *httpd.Service
}

func NewService(conf *ServiceConfig) (service *Service, err error) {
	s := &Service{
		Conf: conf,
	}

	if err = s.listenRaft(); err != nil {
		return
	}

	if err = s.initStore(); err != nil {
		defer s.RaftListener.Close()
		return
	}

	s.listenApi()

	service = s

	return
}

func (s *Service) listenRaft() (err error) {
	addr := net.JoinHostPort(s.Conf.BindAddr, fmt.Sprint(s.Conf.RaftPort))

	var ln net.Listener

	ln, err = net.Listen("tcp", addr)

	if err != nil {
		log.Fatalf("failed to listen on %v: %s", addr, err.Error())
		return
	}

	s.RaftListener = ln

	var adv net.Addr

	if s.Conf.PublicAddr != "" {
		advAddr := net.JoinHostPort(s.Conf.PublicAddr, fmt.Sprint(s.Conf.RaftPort))
		adv, err = net.ResolveTCPAddr("tcp", advAddr)
		if err != nil {
			defer ln.Close()
			log.Fatalf("failed to resolve public address %s: %s", advAddr, err.Error())
			return
		}
	}

	var mux *tcp.Mux

	if mux, err = tcp.NewMux(ln, adv); err != nil {
		log.Fatalf("failed to create raft mux %s", err.Error())
	} else {
		// skip certificate verify
		mux.InsecureSkipVerify = true
		s.RaftMux = mux
	}

	return
}

func (s *Service) listenApi() {
	httpAddr := net.JoinHostPort(s.Conf.BindAddr, fmt.Sprint(s.Conf.ApiPort))
	s.ApiServer = httpd.New(httpAddr, s.DBStore, nil)
	s.ApiServer.Pprof = s.Conf.EnablePprof
	s.ApiServer.Expvar = s.Conf.Expvar
	s.ApiServer.BuildInfo = s.Conf.BuildInfo

	return
}

func (s *Service) initStore() (err error) {
	var dataPath string

	if dataPath, err = filepath.Abs(s.Conf.DataPath); err != nil {
		return
	} else {
		s.DataPath = dataPath
	}

	s.DBConfig = store.NewDBConfig(s.Conf.DSN, !s.Conf.OnDisk)
	// FIXME, auto node id generator
	s.NodeId = fmt.Sprintf("%v_%v",
		strings.Replace(s.Conf.PublicAddr, ".", "_", -1),
		s.Conf.ApiPort)

	raftTn := s.RaftMux.Listen(MuxRaftHeader)
	s.DBStore = store.New(&store.StoreConfig{
		DBConf: s.DBConfig,
		Dir:    s.DataPath,
		Tn:     raftTn,
		ID:     s.NodeId,
	})

	s.DBStore.SnapshotThreshold = s.Conf.RaftSnapshotThreshold
	s.DBStore.HeartbeatTimeout = s.Conf.RaftHeartbeatTimeout
	s.DBStore.ApplyTimeout = s.Conf.RaftApplyTimeout
	s.DBStore.OpenTimeout = s.Conf.RaftOpenTimeout

	// open database
	if err = s.DBStore.Open(len(s.Conf.InitPeers) == 0); err != nil {
		return
	}

	return
}

func (s *Service) setPeers() (err error) {
	ticker := time.NewTicker(s.Conf.PublishPeersDelay)
	defer ticker.Stop()
	timer := time.NewTimer(s.Conf.PublishPeersTimeout)
	defer timer.Stop()

	raftAddr := net.JoinHostPort(s.Conf.PublicAddr, fmt.Sprint(s.Conf.RaftPort))
	apiAddr := net.JoinHostPort(s.Conf.PublicAddr, fmt.Sprint(s.Conf.ApiPort))

	for {
		select {
		case <-ticker.C:
			if err = s.ClusterService.SetPeer(raftAddr, apiAddr); err != nil {
				log.Warningf("failed to set peer for %s to %s: %s (retrying)",
					raftAddr, apiAddr, err.Error())
				continue
			}
			return
		case <-timer.C:
			err = errors.New("set peer timeout expired")
			break
		}
	}

	return
}

func (s *Service) Serve() (err error) {
	// start raft server
	go s.RaftMux.Serve()

	// open raft fsm
	metaTn := s.RaftMux.Listen(MuxMetaHeader)
	s.ClusterService = cluster.NewService(metaTn, s.DBStore)
	if err = s.ClusterService.Open(); err != nil {
		return
	}

	// join peers
	if len(s.Conf.InitPeers) > 0 {
		advAddr := net.JoinHostPort(s.Conf.PublicAddr, fmt.Sprint(s.Conf.RaftPort))
		if _, err = cluster.Join(s.Conf.InitPeers, s.NodeId, advAddr, true); err != nil {
			return
		}
	}

	// start api server
	if err = s.ApiServer.Start(); err != nil {
		return
	}

	if err = s.ApiServer.RegisterStatus("mux", s.RaftMux); err != nil {
		defer s.ApiServer.Close()
		return
	}

	// publish api address
	if err = s.setPeers(); err != nil {
		defer s.ApiServer.Close()
		return
	}

	return
}

func (s *Service) Close() (err error) {
	// stop storage
	if err = s.DBStore.Close(true); err != nil {
		return
	}

	// stop api server
	s.ApiServer.Close()

	// stop raft server
	s.RaftListener.Close()

	return
}
