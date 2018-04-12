package server

import (
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/rqlite/rqlite/tcp"
	"path/filepath"
	"github.com/rqlite/rqlite/store"
	"fmt"
	"strings"
	"time"
	"github.com/rqlite/rqlite/cluster"
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
}

func NewService(conf *ServiceConfig) (service *Service, err error) {
	s := &Service{
		Conf: conf,
	}

	if err = s.listenRaft(); err != nil {
		return
	}
	if err = s.listenApi(); err != nil {
		defer s.RaftListener.Close()
		return
	}
	if err = s.initStore(); err != nil {
		defer s.RaftListener.Close()
		return
	}
}

func (s *Service) listenRaft() (err error) {
	addr := net.JoinHostPort(s.Conf.BindAddr, fmt.Sprint(s.Conf.RaftPort))
	ln, err := net.Listen("tcp", addr)

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

	if mux, err := tcp.NewMux(ln, adv); err != nil {
		log.Fatalf("failed to create raft mux %s", err.Error())
		return
	} else {
		// skip certificate verify
		mux.InsecureSkipVerify = true
		s.RaftMux = mux
	}
}

func (s *Service) listenApi() error {

}

func (s *Service) initStore() (err error) {
	if dataPath, err := filepath.Abs(s.Conf.DataPath); err != nil {
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

func (s *Service) Serve() (err error) {
	// start server
	go s.RaftMux.Serve()

	// open raft fsm
	metaTn := s.RaftMux.Listen(MuxMetaHeader)
	s.ClusterService = cluster.NewService(metaTn, s.DBStore)
	if err := s.ClusterService.Open(); err != nil {
		return
	}

	return
}
