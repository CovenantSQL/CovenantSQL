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
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/consistent"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	ka "gitlab.com/thunderdb/ThunderDB/kayak/api"
	kt "gitlab.com/thunderdb/ThunderDB/kayak/transport"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	"gitlab.com/thunderdb/ThunderDB/twopc"
	"gitlab.com/thunderdb/ThunderDB/utils"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

const (
	listenHost          = "127.0.0.1"
	listenAddrPattern   = listenHost + ":%v"
	nodeDirPattern      = "./node_%v"
	pubKeyStoreFile     = "public.keystore"
	privateKeyFile      = "private.key"
	privateKeyMasterKey = "abc"
	kayakServiceName    = "kayak"
	dbServiceName       = "Database"
	dbFileName          = "storage.db"
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
	payloadCodec    logCodec
)

// NodeInfo for conf generation and load purpose.
type NodeInfo struct {
	Nonce          *cpuminer.NonceInfo
	PublicKeyBytes []byte
	PublicKey      *asymmetric.PublicKey
	Port           int
	Role           conf.ServerRole
}

type logCodec struct{}

func (m *logCodec) encode(execLog *storage.ExecLog) ([]byte, error) {
	return json.Marshal(execLog)
}

func (m *logCodec) decode(data []byte, execLog *storage.ExecLog) (err error) {
	return json.Unmarshal(data, execLog)
}

type storageWrap struct {
	*storage.Storage
}

// Prepare implements twopc.Worker.Prepare.
func (s *storageWrap) Prepare(ctx context.Context, wb twopc.WriteBatch) (err error) {
	var execLog *storage.ExecLog
	if execLog, err = s.decodeLog(wb); err != nil {
		return
	}

	return s.Storage.Prepare(ctx, execLog)
}

// Commit implements twopc.Worker.Commit.
func (s *storageWrap) Commit(ctx context.Context, wb twopc.WriteBatch) (err error) {
	var execLog *storage.ExecLog
	if execLog, err = s.decodeLog(wb); err != nil {
		return
	}

	return s.Storage.Commit(ctx, execLog)
}

// Rollback implements twopc.Worker.Rollback.
func (s *storageWrap) Rollback(ctx context.Context, wb twopc.WriteBatch) (err error) {
	var execLog *storage.ExecLog
	if execLog, err = s.decodeLog(wb); err != nil {
		return
	}

	return s.Storage.Rollback(ctx, execLog)
}

func (s *storageWrap) decodeLog(wb twopc.WriteBatch) (log *storage.ExecLog, err error) {
	var bytesPayload []byte
	var execLog storage.ExecLog
	var ok bool

	if bytesPayload, ok = wb.([]byte); !ok {
		err = kayak.ErrInvalidLog
		return
	}
	if err = payloadCodec.decode(bytesPayload, &execLog); err != nil {
		return
	}

	log = &execLog
	return
}

type stubServer struct {
	runtime *kayak.Runtime
	storage *storageWrap
}

type responseRows []map[string]interface{}

// Write defines Database.Write rpc.
func (s *stubServer) Write(sql string, _ *responseRows) (err error) {
	var writeData []byte

	l := &storage.ExecLog{
		Queries: []storage.Query{
			{
				Pattern: sql,
			},
		},
	}

	if writeData, err = payloadCodec.encode(l); err != nil {
		return err
	}

	err = s.runtime.Apply(writeData)

	return
}

// Read defines Database.Read rpc.
func (s *stubServer) Read(sql string, rows *responseRows) (err error) {
	var columns []string
	var result [][]interface{}
	columns, _, result, err = s.storage.Query(context.Background(), []storage.Query{
		{
			Pattern: sql,
		},
	})

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
	flag.StringVar(&confPath, "conf", "peers.json", "peers conf path")
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
		if nodes[i].Role == conf.Leader {
			leader = &nodes[i]
		} else if nodes[i].Role != conf.Follower {
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
	var conn net.Conn
	if conn, err = rpc.DialToNode(leaderNodeID, nil); err != nil {
		return
	}

	var client *rpc.Client
	if client, err = rpc.InitClientConn(conn); err != nil {
		return
	}

	reqType = strings.Title(strings.ToLower(reqType))

	var rows responseRows
	if err = client.Call(fmt.Sprintf("%v.%v", dbServiceName, reqType), sql, &rows); err != nil {
		return
	}
	var res []byte
	if res, err = json.MarshalIndent(rows, "", strings.Repeat(" ", 4)); err != nil {
		return
	}

	fmt.Println(string(res))

	return
}

func initClientKey(client *NodeInfo, leader *NodeInfo) (err error) {
	clientRootDir := fmt.Sprintf(nodeDirPattern, "client")
	os.MkdirAll(clientRootDir, 0755)

	// init local key store
	pubKeyStorePath := filepath.Join(clientRootDir, pubKeyStoreFile)
	if _, err = consistent.InitConsistent(pubKeyStorePath, new(consistent.KMSStorage), true); err != nil {
		return
	}

	// init client private key
	//route.initResolver()
	privateKeyStorePath := filepath.Join(clientRootDir, privateKeyFile)
	if err = kms.InitLocalKeyPair(privateKeyStorePath, []byte(privateKeyMasterKey)); err != nil {
		return
	}

	kms.SetLocalNodeIDNonce(client.Nonce.Hash.CloneBytes(), &client.Nonce.Nonce)

	// set leader key
	leaderNodeID := proto.NodeID(leader.Nonce.Hash.String())
	kms.SetPublicKey(leaderNodeID, leader.Nonce.Nonce, leader.PublicKey)

	// set route to leader
	route.SetNodeAddrCache(&proto.RawNodeID{Hash: leader.Nonce.Hash}, fmt.Sprintf(listenAddrPattern, leader.Port))

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
	var server *rpc.Server

	// create server
	log.Infof("create server")
	if service, server, err = createServer(nodeOffset, nodes[nodeOffset].Port); err != nil {
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
	var st *storageWrap
	if st, err = initStorage(nodeOffset); err != nil {
		return
	}

	// init kayak
	log.Infof("init kayak runtime")
	var kayakRuntime *kayak.Runtime
	if _, kayakRuntime, err = initKayakTwoPC(nodeOffset, &nodes[nodeOffset], peers, st, service); err != nil {
		return
	}

	// register service rpc
	server.RegisterService(dbServiceName, &stubServer{
		runtime: kayakRuntime,
		storage: st,
	})

	// start server
	server.Serve()

	return
}

func initStorage(nodeOffset int) (stor *storageWrap, err error) {
	dbFile := filepath.Join(fmt.Sprintf(nodeDirPattern, nodeOffset), dbFileName)
	var st *storage.Storage
	if st, err = storage.New(dbFile); err != nil {
		return
	}

	stor = &storageWrap{
		Storage: st,
	}
	return
}

func initKayakTwoPC(nodeOffset int, node *NodeInfo, peers *kayak.Peers,
	worker twopc.Worker, service *kt.ETLSTransportService) (config kayak.Config, runtime *kayak.Runtime, err error) {
	// create kayak config
	log.Infof("create twopc config")
	rootDir := fmt.Sprintf(nodeDirPattern, nodeOffset)
	config = ka.NewTwoPCConfig(rootDir, service, worker)

	// create kayak runtime
	log.Infof("create kayak runtime")
	runtime, err = ka.NewTwoPCKayak(peers, config)
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
		if node.Role == conf.Leader {
			peers.Leader = s
		} else if node.Role != conf.Follower {
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
	if ports, err = utils.GetRandomPorts(listenHost, minPort, maxPort, nodeCnt); err != nil {
		return
	}

	// server conf
	for i := 0; i != nodeCnt; i++ {
		nodeRootDir := fmt.Sprintf(nodeDirPattern, i)
		nodes[i].Nonce, nodes[i].PublicKeyBytes, err = getSingleNodeConf(nodeRootDir)

		if err != nil {
			return
		}

		nodes[i].Port = ports[i]

		if i == 0 {
			nodes[i].Role = conf.Leader
		} else {
			nodes[i].Role = conf.Follower
		}
	}

	// client conf
	nodes[nodeCnt].Nonce, nodes[nodeCnt].PublicKeyBytes, err = getSingleNodeConf(fmt.Sprintf(nodeDirPattern, "client"))
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
	privateKeyPath := filepath.Join(nodeRootDir, privateKeyFile)
	var privateKey *asymmetric.PrivateKey
	var publicKey *asymmetric.PublicKey

	// create new key
	privateKey, publicKey, err = asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	// save key to file
	err = kms.SavePrivateKey(privateKeyPath, privateKey, []byte(privateKeyMasterKey))

	if err != nil {
		return
	}

	// generate node nonce with public key
	n := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	nonce = &n
	pubKeyBytes = publicKey.Serialize()

	return
}

func createServer(nodeOffset, port int) (service *kt.ETLSTransportService, server *rpc.Server, err error) {
	pubKeyStorePath := filepath.Join(fmt.Sprintf(nodeDirPattern, nodeOffset), pubKeyStoreFile)
	os.Remove(pubKeyStorePath)

	var dht *route.DHTService
	if dht, err = route.NewDHTService(pubKeyStorePath, new(consistent.KMSStorage), true); err != nil {
		return
	}

	server, err = rpc.NewServerWithService(rpc.ServiceMap{
		"DHT": dht,
	})
	if err != nil {
		return
	}

	listenAddr := fmt.Sprintf(listenAddrPattern, port)
	privateKeyPath := filepath.Join(fmt.Sprintf(nodeDirPattern, nodeOffset), privateKeyFile)
	err = server.InitRPCServer(listenAddr, privateKeyPath, []byte(privateKeyMasterKey))
	service = ka.NewMuxService(kayakServiceName, server)

	return
}

func initNode(node *NodeInfo) (nodeID proto.NodeID, err error) {
	nodeID = proto.NodeID(node.Nonce.Hash.String())
	kms.SetPublicKey(nodeID, node.Nonce.Nonce, node.PublicKey)
	route.SetNodeAddrCache(&proto.RawNodeID{Hash: node.Nonce.Hash}, fmt.Sprintf(listenAddrPattern, node.Port))

	return
}
