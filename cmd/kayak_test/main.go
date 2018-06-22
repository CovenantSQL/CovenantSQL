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
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/consistent"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/etls"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	kt "gitlab.com/thunderdb/ThunderDB/kayak/transport"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	"gitlab.com/thunderdb/ThunderDB/twopc"
	"gitlab.com/thunderdb/ThunderDB/utils"
)

const (
	ListenHost               = "127.0.0.1"
	ListenAddrPattern        = ListenHost + ":%v"
	NodeDirPattern           = "./node_%v"
	PubKeyStoreFile          = "public.keystore"
	PrivateKeyFile           = "private.key"
	PrivateKeyMasterKey      = "abc"
	KayakServiceName         = "kayak"
	KayakInstanceTransportID = "kayak_test"
	DBServiceName            = "Database"
	DBFileName               = "storage.db"
)

var (
	genConf         bool
	clientMode      bool
	confPath        string
	minPort         int
	maxPort         int
	nodeCnt         int
	nodeOffset      int
	clientOperation string
)

type NodeInfo struct {
	Nonce          *cpuminer.NonceInfo
	PublicKeyBytes []byte
	PublicKey      *asymmetric.PublicKey
	Port           int
	Role           kayak.ServerRole
}

type LogCodec struct{}

func (m *LogCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (m *LogCodec) Decode(data []byte, v interface{}) (err error) {
	var payload storage.ExecLog
	err = json.Unmarshal(data, &payload)

	if v == nil {
		return
	}

	rv := reflect.ValueOf(v)

	if rv.Kind() == reflect.Ptr {
		rv.Elem().Set(reflect.ValueOf(&payload))
	} else {
		rv.Set(reflect.ValueOf(&payload))
	}

	return
}

type StubServer struct {
	Runtime *kayak.Runtime
	Storage *storage.Storage
	Codec   kayak.TwoPCLogCodec
}

func (s *StubServer) Write(sql string, response *string) (err error) {
	var writeData []byte

	l := &storage.ExecLog{
		Queries: []string{sql},
	}

	if writeData, err = s.Codec.Encode(l); err != nil {
		return err
	}

	err = s.Runtime.Apply(writeData)

	return
}

func (s *StubServer) Read(sql string, rows *[]map[string]interface{}) (err error) {
	var columns []string
	var result [][]interface{}
	columns, _, result, err = s.Storage.Query(context.Background(), []string{sql})

	// rebuild map
	*rows = make([]map[string]interface{}, 0, len(result))

	for _, r := range result {
		// build row map
		row := make(map[string]interface{})

		for i, c := range r {
			row[columns[i]] = c
		}

		*rows = append(*rows, row)
	}

	return
}

func init() {
	flag.BoolVar(&genConf, "generate", false, "run conf generation")
	flag.BoolVar(&clientMode, "client", false, "run as client")
	flag.StringVar(&confPath, "conf", "peers.conf", "peers conf path")
	flag.IntVar(&minPort, "minPort", 10000, "minimum port number to allocate (used in conf generation)")
	flag.IntVar(&maxPort, "maxPort", 20000, "maximum port number to allocate (used in conf generation)")
	flag.IntVar(&nodeCnt, "nodeCnt", 3, "node count for conf generation")
	flag.IntVar(&nodeOffset, "nodeOffset", 0, "node offset in peers conf")
	flag.StringVar(&clientOperation, "operation", "read", "client operation")
}

func main() {
	flag.Parse()

	if clientMode {
		if err := runClient(); err != nil {
			log.Fatalf("run client failed: %v", err.Error())
		} else {
			log.Infof("run client success")
		}
		return
	} else if genConf {
		// call conf generator
		if err := generateConf(nodeCnt, minPort, maxPort, confPath); err != nil {
			log.Fatalf("generate conf failed: %v", err.Error())
		} else {
			log.Infof("generate conf success")
		}
		return
	}

	if err := runNode(); err != nil {
		log.Fatalf("run kayak failed: %v", err.Error())
	}
}

func runClient() (err error) {
	var nodes []NodeInfo

	// load conf
	log.Infof("load conf")
	if nodes, err = loadConf(confPath); err != nil {
		return
	}

	var leader *NodeInfo
	var client *NodeInfo

	// get leader node
	for i := range nodes {
		if nodes[i].Role == kayak.Leader {
			leader = &nodes[i]
		} else if nodes[i].Role != kayak.Follower {
			client = &nodes[i]
		}
	}

	// create client key
	if err = initClientKey(client, leader); err != nil {
		return
	}

	// do client request
	if err = clientRequest(leader, clientOperation, flag.Arg(0)); err != nil {
		return
	}

	return
}

func clientRequest(leader *NodeInfo, reqType string, sql string) (err error) {
	leaderNodeID := proto.NodeID(leader.Nonce.Hash.String())
	var conn *etls.CryptoConn
	if conn, err = rpc.DialToNode(leaderNodeID); err != nil {
		return
	}

	var client *rpc.Client
	if client, err = rpc.InitClientConn(conn); err != nil {
		return
	}

	reqType = strings.Title(strings.ToLower(reqType))

	var res interface{}
	if err = client.Call(fmt.Sprintf("%v.%v", DBServiceName, reqType), sql, &res); err != nil {
		return
	}

	return
}

func initClientKey(client *NodeInfo, leader *NodeInfo) (err error) {
	clientRootDir := fmt.Sprintf(NodeDirPattern, "client")
	os.MkdirAll(clientRootDir, 0755)

	// init local key store
	pubKeyStorePath := filepath.Join(clientRootDir, PubKeyStoreFile)
	if _, err = consistent.InitConsistent(pubKeyStorePath, true); err != nil {
		return
	}

	// init client private key
	route.InitResolver()
	privateKeyStorePath := filepath.Join(clientRootDir, PrivateKeyFile)
	if err = kms.InitLocalKeyPair(privateKeyStorePath, []byte(PrivateKeyMasterKey)); err != nil {
		return
	}

	kms.SetLocalNodeIDNonce(client.Nonce.Hash.CloneBytes(), &client.Nonce.Nonce)

	// set leader key
	leaderNodeID := proto.NodeID(leader.Nonce.Hash.String())
	kms.SetPublicKey(leaderNodeID, leader.Nonce.Nonce, leader.PublicKey)

	// set route to leader
	route.SetNodeAddr(&proto.RawNodeID{Hash: leader.Nonce.Hash}, fmt.Sprintf(ListenAddrPattern, leader.Port))

	return
}

func runNode() (err error) {
	var nodes []NodeInfo

	// load conf
	log.Infof("load conf")
	if nodes, err = loadConf(confPath); err != nil {
		return
	}

	var service *kt.ETLSTransportService

	// create service
	log.Infof("create service")
	service = createService()

	var server *rpc.Server

	// create server
	log.Infof("create server")
	if server, err = createServer(nodeOffset, nodes[nodeOffset].Port, service); err != nil {
		return
	}

	var peers *kayak.Peers

	// init nodes
	log.Infof("init peers")
	if peers, err = initPeers(nodeOffset, nodes); err != nil {
		return
	}

	// init storage
	log.Infof("init storage")
	var stor *storage.Storage
	if stor, err = initStorage(nodeOffset); err != nil {
		return
	}

	// init kayak
	log.Infof("init kayak runtime")
	var kayakRuntime *kayak.Runtime
	if _, kayakRuntime, err = initKayakTwoPC(nodeOffset, &nodes[nodeOffset], peers, stor, service); err != nil {
		return
	}

	// register service rpc
	server.RegisterService(DBServiceName, &StubServer{
		Runtime: kayakRuntime,
		Storage: stor,
		Codec:   &LogCodec{},
	})

	// start server
	server.Serve()

	return
}

func initStorage(nodeOffset int) (stor *storage.Storage, err error) {
	dbFile := filepath.Join(fmt.Sprintf(NodeDirPattern, nodeOffset), DBFileName)
	return storage.New(dbFile)
}

func initKayakTwoPC(nodeOffset int, node *NodeInfo, peers *kayak.Peers, worker twopc.Worker, service *kt.ETLSTransportService) (config kayak.Config, runtime *kayak.Runtime, err error) {
	// create twopc runner
	log.Infof("create twopc runner")
	runner := kayak.NewTwoPCRunner()
	localNodeID := proto.NodeID(node.Nonce.Hash.String())

	// create transport
	log.Infof("create etls transport config")
	transportConfig := &kt.ETLSTransportConfig{
		NodeID:           localNodeID,
		TransportID:      KayakInstanceTransportID,
		TransportService: service,
		ServiceName:      KayakServiceName,
		ClientBuilder: func(ctx context.Context, nodeID proto.NodeID) (client *rpc.Client, err error) {
			log.Infof("dial to node %s", nodeID)

			var conn *etls.CryptoConn
			conn, err = rpc.DialToNode(nodeID)
			if err != nil {
				return
			}

			client, err = rpc.InitClientConn(conn)

			return
		},
	}

	log.Infof("create etls transport")
	var transport *kt.ETLSTransport
	if transport, err = kt.NewETLSTransport(transportConfig); err != nil {
		return
	}

	// create twopc config
	log.Infof("create kayak twopc config")
	config = &kayak.TwoPCConfig{
		RuntimeConfig: kayak.RuntimeConfig{
			RootDir:        fmt.Sprintf(NodeDirPattern, nodeOffset),
			LocalID:        localNodeID,
			Runner:         runner,
			Transport:      transport,
			ProcessTimeout: time.Second * 5,
			Logger:         log.New(),
		},
		LogCodec: &LogCodec{},
		Storage:  worker,
	}

	// create runtime
	log.Infof("create kayak twopc runtime")
	runtime, err = kayak.NewRuntime(config, peers)
	if err != nil {
		return
	}

	// init runtime
	log.Infof("init kayak twopc runtime")
	err = runtime.Init()

	return
}

func initPeers(nodeOffset int, nodes []NodeInfo) (peers *kayak.Peers, err error) {
	if nodeOffset < 0 || nodeOffset >= len(nodes) {
		err = errors.New("invalid node offset")
		return
	}

	var publicKey *asymmetric.PublicKey
	var privateKey *asymmetric.PrivateKey

	// load public
	publicKey, err = kms.GetLocalPublicKey()
	if err != nil {
		return
	}

	// load private key
	privateKey, err = kms.GetLocalPrivateKey()
	if err != nil {
		return
	}

	// init peers struct
	peers = &kayak.Peers{
		Term:    uint64(1),
		Servers: make([]*kayak.Server, len(nodes)-1),
		PubKey:  publicKey,
	}

	for i := range nodes {
		var nodeID proto.NodeID
		var node = &nodes[i]

		// build node id
		if nodeID, err = initNode(node); err != nil {
			return
		}

		// build server struct
		s := &kayak.Server{
			Role:   node.Role,
			ID:     nodeID,
			PubKey: node.PublicKey,
		}

		// set leader
		if node.Role == kayak.Leader {
			peers.Leader = s
		} else if node.Role != kayak.Follower {
			// complete
			break
		}

		peers.Servers[i] = s

		// init current node
		if i == nodeOffset {
			// init current node by conf
			if err = initCurrentNode(node); err != nil {
				return
			}
		}
	}

	// sign peers
	err = peers.Sign(privateKey)

	return
}

func initCurrentNode(node *NodeInfo) (err error) {
	// set local node id info
	kms.SetLocalNodeIDNonce(node.Nonce.Hash.CloneBytes(), &node.Nonce.Nonce)

	return
}

func loadConf(filename string) (nodes []NodeInfo, err error) {
	nodes = make([]NodeInfo, 0)

	var fileContent []byte
	fileContent, err = ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = json.Unmarshal(fileContent, &nodes)
	if err != nil {
		return
	}

	for i := range nodes {
		nodes[i].PublicKey, err = asymmetric.ParsePubKey(nodes[i].PublicKeyBytes)
		if err != nil {
			return
		}
	}

	return
}

func generateConf(nodeCnt, minPort, maxPort int, filename string) (err error) {
	// server including client
	nodes := make([]NodeInfo, nodeCnt+1)

	var ports []int
	if ports, err = utils.GetRandomPorts(ListenHost, minPort, maxPort, nodeCnt); err != nil {
		return
	}

	// server conf
	for i := 0; i != nodeCnt; i++ {
		nodeRootDir := fmt.Sprintf(NodeDirPattern, i)
		nodes[i].Nonce, nodes[i].PublicKeyBytes, err = getSingleNodeConf(nodeRootDir)

		if err != nil {
			return
		}

		nodes[i].Port = ports[i]

		if i == 0 {
			nodes[i].Role = kayak.Leader
		} else {
			nodes[i].Role = kayak.Follower
		}
	}

	// client conf
	nodes[nodeCnt].Nonce, nodes[nodeCnt].PublicKeyBytes, err = getSingleNodeConf(fmt.Sprintf(NodeDirPattern, "client"))
	if err != nil {
		return
	}

	nodes[nodeCnt].Role = -1

	var confData []byte
	confData, err = json.MarshalIndent(nodes, "", strings.Repeat(" ", 4))

	err = ioutil.WriteFile(filename, confData, 0644)

	return
}

func getSingleNodeConf(nodeRootDir string) (nonce *cpuminer.NonceInfo, pubKeyBytes []byte, err error) {
	// remove node dir if exists
	os.RemoveAll(nodeRootDir)

	// create node root dir
	os.MkdirAll(nodeRootDir, 0755)

	// create private key
	privateKeyPath := filepath.Join(nodeRootDir, PrivateKeyFile)
	var privateKey *asymmetric.PrivateKey
	var publicKey *asymmetric.PublicKey

	// create new key
	privateKey, publicKey, err = asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	// save key to file
	err = kms.SavePrivateKey(privateKeyPath, privateKey, []byte(PrivateKeyMasterKey))

	if err != nil {
		return
	}

	// generate node nonce with public key
	n := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	nonce = &n
	pubKeyBytes = publicKey.Serialize()

	return
}

func createService() *kt.ETLSTransportService {
	return &kt.ETLSTransportService{}
}

func createServer(nodeOffset, port int, service *kt.ETLSTransportService) (server *rpc.Server, err error) {
	pubKeyStorePath := filepath.Join(fmt.Sprintf(NodeDirPattern, nodeOffset), PubKeyStoreFile)
	os.Remove(pubKeyStorePath)

	var dht *route.DHTService
	if dht, err = route.NewDHTService(pubKeyStorePath, true); err != nil {
		return
	}

	server, err = rpc.NewServerWithService(rpc.ServiceMap{
		"DHT":            dht,
		KayakServiceName: service,
	})
	if err != nil {
		return
	}

	listenAddr := fmt.Sprintf(ListenAddrPattern, port)
	privateKeyPath := filepath.Join(fmt.Sprintf(NodeDirPattern, nodeOffset), PrivateKeyFile)
	err = server.InitRPCServer(listenAddr, privateKeyPath, []byte(PrivateKeyMasterKey))

	return
}

func initNode(node *NodeInfo) (nodeID proto.NodeID, err error) {
	nodeID = proto.NodeID(node.Nonce.Hash.String())
	kms.SetPublicKey(nodeID, node.Nonce.Nonce, node.PublicKey)
	route.SetNodeAddr(&proto.RawNodeID{Hash: node.Nonce.Hash}, fmt.Sprintf(ListenAddrPattern, node.Port))

	return
}
