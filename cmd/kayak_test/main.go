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
	"gitlab.com/thunderdb/ThunderDB/rpc"
	kt "gitlab.com/thunderdb/ThunderDB/kayak/transport"
	"gitlab.com/thunderdb/ThunderDB/route"
	"fmt"
	"os"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"time"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"encoding/json"
	"gitlab.com/thunderdb/ThunderDB/utils"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"strings"
	"io/ioutil"
	"flag"
	log "github.com/sirupsen/logrus"
	"errors"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	"gitlab.com/thunderdb/ThunderDB/twopc"
	"context"
	"gitlab.com/thunderdb/ThunderDB/crypto/etls"
)

const (
	ListenHost               = "127.0.0.1"
	ListenAddrPattern        = ListenHost + ":%v"
	NodeDirPattern           = "./node_%v"
	PubKeyStorePattern       = "./public.%v.keystore"
	PrivateKeyPattern        = "./private.%v.keypair"
	PrivateKeyMasterKey      = "abc"
	KayakServiceName         = "kayak"
	KayakInstanceTransportID = "kayak_test"
)

var (
	genConf    bool
	confPath   string
	minPort    int
	maxPort    int
	nodeCnt    int
	nodeOffset int
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

func (m *LogCodec) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func init() {
	flag.BoolVar(&genConf, "generate", false, "generate conf mode")
	flag.StringVar(&confPath, "conf", "peers.conf", "peers conf path")
	flag.IntVar(&minPort, "minPort", 10000, "minimum port number to allocate (used in conf generation)")
	flag.IntVar(&maxPort, "maxPort", 20000, "maximum port number to allocate (used in conf generation)")
	flag.IntVar(&nodeCnt, "nodeCnt", 3, "node count for conf generation")
	flag.IntVar(&nodeOffset, "nodeOffset", 0, "node offset in peers conf")
}

func main() {
	flag.Parse()

	if genConf {
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
	if stor, err = initStorage(); err != nil {
		return
	}

	// init kayak
	log.Infof("init kayak runtime")
	var kayakRuntime *kayak.Runtime
	if _, kayakRuntime, err = initKayakTwoPC(nodeOffset, &nodes[nodeOffset], peers, stor, service);
		err != nil {
		return
	}

	// start server
	go server.Serve()

	// TODO, make following statement as rpc endpoint

	// call kayak apply
	// get codec
	codec := &LogCodec{}

	var testData []byte
	testData, err = codec.Encode("test data")
	if err != nil {
		return
	}
	log.Infof("apply data to kayak")
	err = kayakRuntime.Apply(testData)

	return
}

func initStorage() (stor *storage.Storage, err error) {
	return storage.New(":memory:")
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

			if d, ok := ctx.Deadline(); ok {
				conn.SetDeadline(d)
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
			ProcessTimeout: time.Second,
			Logger:         log.New(),
		},
		LogCodec:        &LogCodec{},
		Storage:         worker,
		PrepareTimeout:  time.Millisecond * 500,
		CommitTimeout:   time.Millisecond * 500,
		RollbackTimeout: time.Millisecond * 500,
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
		Servers: make([]*kayak.Server, len(nodes)),
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

		peers.Servers[i] = s

		// set leader
		if node.Role == kayak.Leader {
			peers.Leader = s
		}

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
	nodes := make([]NodeInfo, nodeCnt)

	var ports []int
	if ports, err = utils.GetRandomPorts(ListenHost, minPort, maxPort, nodeCnt); err != nil {
		return
	}

	for i := 0; i != nodeCnt; i++ {
		nodes[i].Nonce, nodes[i].PublicKeyBytes, err = getSingleNodeNonce(i)

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

	var confData []byte
	confData, err = json.MarshalIndent(nodes, "", strings.Repeat(" ", 4))

	err = ioutil.WriteFile(filename, confData, 0644)

	return
}

func getSingleNodeNonce(nodeOffset int) (nonce *cpuminer.NonceInfo, pubKeyBytes []byte, err error) {
	privateKeyPath := fmt.Sprintf(PrivateKeyPattern, nodeOffset)
	var privateKey *asymmetric.PrivateKey
	var publicKey *asymmetric.PublicKey

	privateKey, err = kms.LoadPrivateKey(privateKeyPath, []byte(PrivateKeyMasterKey))
	if err == nil {
		publicKey = privateKey.PubKey()
	} else if err == kms.ErrNotKeyFile {
		return
	} else {
		// create new key
		if _, ok := err.(*os.PathError); ok || err == os.ErrNotExist {
			privateKey, publicKey, err = asymmetric.GenSecp256k1KeyPair()

			if err != nil {
				return
			}

			err = kms.SavePrivateKey(privateKeyPath, privateKey, []byte(PrivateKeyMasterKey))
		}
	}

	n := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	nonce = &n
	pubKeyBytes = publicKey.Serialize()

	return
}

func createService() *kt.ETLSTransportService {
	return &kt.ETLSTransportService{}
}

func createServer(nodeOffset, port int, service *kt.ETLSTransportService) (server *rpc.Server, err error) {
	pubKeyStorePath := fmt.Sprintf(PubKeyStorePattern, port)
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
	privateKayPath := fmt.Sprintf(PrivateKeyPattern, nodeOffset)
	err = server.InitRPCServer(listenAddr, privateKayPath, []byte(PrivateKeyMasterKey))

	return
}

func initNode(node *NodeInfo) (nodeID proto.NodeID, err error) {
	nodeID = proto.NodeID(node.Nonce.Hash.String())
	kms.SetPublicKey(nodeID, node.Nonce.Nonce, node.PublicKey)
	route.SetNodeAddr(&proto.RawNodeID{Hash: node.Nonce.Hash}, fmt.Sprintf(ListenAddrPattern, node.Port))

	return
}
