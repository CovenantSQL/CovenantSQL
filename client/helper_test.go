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

package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/worker"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

const (
	// PubKeyStorePath defines public cache store.
	PubKeyStorePath = "./public.keystore"
)

var (
	rootHash = hash.Hash{}
)

// fake BPDB service
type stubBPDBService struct{}

func (s *stubBPDBService) CreateDatabase(req *bp.CreateDatabaseRequest, resp *bp.CreateDatabaseResponse) (err error) {
	if resp.Header.InstanceMeta, err = s.getInstanceMeta(proto.DatabaseID("db")); err != nil {
		return
	}
	if resp.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	err = resp.Sign(privateKey)

	return
}

func (s *stubBPDBService) DropDatabase(req *bp.DropDatabaseRequest, resp *bp.DropDatabaseRequest) (err error) {
	return
}

func (s *stubBPDBService) GetDatabase(req *bp.GetDatabaseRequest, resp *bp.GetDatabaseResponse) (err error) {
	if resp.Header.InstanceMeta, err = s.getInstanceMeta(req.Header.DatabaseID); err != nil {
		return
	}
	if resp.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	err = resp.Sign(privateKey)

	return
}

func (s *stubBPDBService) GetNodeDatabases(req *wt.InitService, resp *wt.InitServiceResponse) (err error) {
	resp.Header.Instances = make([]wt.ServiceInstance, 0)
	if resp.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if resp.Sign(privateKey); err != nil {
		return
	}
	return
}

func (s *stubBPDBService) getInstanceMeta(dbID proto.DatabaseID) (instance wt.ServiceInstance, err error) {
	var privKey *asymmetric.PrivateKey
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	instance.DatabaseID = proto.DatabaseID(dbID)
	instance.Peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Term:    1,
			Leader:  nodeID,
			Servers: []proto.NodeID{nodeID},
		},
	}
	if err = instance.Peers.Sign(privKey); err != nil {
		return
	}
	instance.GenesisBlock, err = createRandomBlock(rootHash, true)

	return
}

func (s *stubBPDBService) QueryAccountStableBalance(req *bp.QueryAccountStableBalanceReq,
	resp *bp.QueryAccountStableBalanceResp) (err error) {
	return
}

func (s *stubBPDBService) QueryAccountCovenantBalance(req *bp.QueryAccountCovenantBalanceReq,
	resp *bp.QueryAccountCovenantBalanceResp) (err error) {
	return
}

func startTestService() (stopTestService func(), tempDir string, err error) {
	var server *rpc.Server
	var cleanup func()
	if cleanup, tempDir, server, err = initNode(); err != nil {
		return
	}

	var rootDir string
	if rootDir, err = ioutil.TempDir("", "dbms_test_"); err != nil {
		return
	}

	cfg := &worker.DBMSConfig{
		RootDir:       rootDir,
		Server:        server,
		MaxReqTimeGap: worker.DefaultMaxReqTimeGap,
	}

	var dbms *worker.DBMS
	if dbms, err = worker.NewDBMS(cfg); err != nil {
		return
	}

	stopTestService = func() {
		if dbms != nil {
			dbms.Shutdown()
		}

		cleanup()

		// cleanup session pool
		rpc.GetSessionPoolInstance().Close()
	}

	// init
	if err = dbms.Init(); err != nil {
		return
	}

	// add database
	var req *wt.UpdateService
	var res wt.UpdateServiceResponse
	var peers *proto.Peers
	var block *ct.Block

	dbID := proto.DatabaseID("db")

	// create sqlchain block
	block, err = createRandomBlock(rootHash, true)

	// get database peers
	if peers, err = genPeers(1); err != nil {
		return
	}

	// build create database request
	req = new(wt.UpdateService)
	req.Header.Op = wt.CreateDB
	req.Header.Instance = wt.ServiceInstance{
		DatabaseID:   dbID,
		Peers:        peers,
		GenesisBlock: block,
	}
	if req.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if err = req.Sign(privateKey); err != nil {
		return
	}

	// send create database request
	if err = testRequest(route.DBSDeploy, req, &res); err != nil {
		return
	}

	return
}

func initNode() (cleanupFunc func(), tempDir string, server *rpc.Server, err error) {
	if tempDir, err = ioutil.TempDir("", "db_test_"); err != nil {
		return
	}
	log.WithField("d", tempDir).Debug("created temp dir")

	// init conf
	_, testFile, _, _ := runtime.Caller(0)
	pubKeyStoreFile := filepath.Join(tempDir, PubKeyStorePath+"_dht")
	os.Remove(pubKeyStoreFile)
	clientPubKeyStoreFile := filepath.Join(tempDir, PubKeyStorePath+"_c")
	os.Remove(clientPubKeyStoreFile)
	dupConfFile := filepath.Join(tempDir, "config.yaml")
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/config.yaml")
	if err = dupConf(confFile, dupConfFile); err != nil {
		return
	}
	privateKeyPath := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/private.key")
	conf.GConf, _ = conf.LoadConfig(dupConfFile)
	log.Debugf("GConf: %#v", conf.GConf)
	_, err = utils.CopyFile(privateKeyPath, conf.GConf.PrivateKeyFile)
	if err != nil {
		log.WithFields(log.Fields{
			"from": privateKeyPath,
			"to":   conf.GConf.PrivateKeyFile,
		}).WithError(err).Fatal("copy private key failed")
		return
	}
	// reset the once
	route.Once = sync.Once{}
	route.InitKMS(clientPubKeyStoreFile)

	var dht *route.DHTService

	// init dht
	dht, err = route.NewDHTService(pubKeyStoreFile, new(consistent.KMSStorage), true)
	if err != nil {
		return
	}

	// init rpc
	if server, err = rpc.NewServerWithService(rpc.ServiceMap{route.DHTRPCName: dht}); err != nil {
		return
	}

	// register bpdb service
	if err = server.RegisterService(route.BPDBRPCName, &stubBPDBService{}); err != nil {
		return
	}

	// register fake chain service
	if err = server.RegisterService(route.BlockProducerRPCName, &stubBPDBService{}); err != nil {
		return
	}

	// init private key
	masterKey := []byte("")
	if err = server.InitRPCServer(conf.GConf.ListenAddr, privateKeyPath, masterKey); err != nil {
		return
	}

	// start server
	go server.Serve()

	cleanupFunc = func() {
		os.RemoveAll(tempDir)
		server.Listener.Close()
		server.Stop()
	}

	// fake database init already processed
	atomic.StoreUint32(&driverInitialized, 1)

	return
}

// copied from sqlchain.xxx_test.
func createRandomBlock(parent hash.Hash, isGenesis bool) (b *ct.Block, err error) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &ct.Block{
		SignedHeader: ct.SignedHeader{
			Header: ct.Header{
				Version:     0x01000000,
				Producer:    proto.NodeID(h.String()),
				GenesisHash: rootHash,
				ParentHash:  parent,
				Timestamp:   time.Now().UTC(),
			},
		},
		Queries: make([]*hash.Hash, rand.Intn(10)+10),
	}

	for i := range b.Queries {
		b.Queries[i] = new(hash.Hash)
		rand.Read(b.Queries[i][:])
	}

	if isGenesis {
		// Compute nonce with public key
		nonceCh := make(chan cpuminer.NonceInfo)
		quitCh := make(chan struct{})
		miner := cpuminer.NewCPUMiner(quitCh)
		go miner.ComputeBlockNonce(cpuminer.MiningBlock{
			Data:      pub.Serialize(),
			NonceChan: nonceCh,
			Stop:      nil,
		}, cpuminer.Uint256{A: 0, B: 0, C: 0, D: 0}, 4)
		nonce := <-nonceCh
		close(quitCh)
		close(nonceCh)
		// Add public key to KMS
		id := cpuminer.HashBlock(pub.Serialize(), nonce.Nonce)
		b.SignedHeader.Header.Producer = proto.NodeID(id.String())
		err = kms.SetPublicKey(proto.NodeID(id.String()), nonce.Nonce, pub)

		if err != nil {
			return nil, err
		}
	}

	err = b.PackAndSignBlock(priv)
	return
}

func testRequest(method route.RemoteFunc, req interface{}, response interface{}) (err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	return rpc.NewCaller().CallNode(nodeID, method.String(), req, response)
}

func getKeys() (privKey *asymmetric.PrivateKey, pubKey *asymmetric.PublicKey, err error) {
	// get public key
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	// get private key
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	return
}

func genPeers(term uint64) (peers *proto.Peers, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get private/public key
	var privateKey *asymmetric.PrivateKey

	if privateKey, _, err = getKeys(); err != nil {
		return
	}

	// generate peers and sign
	peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Term:    term,
			Leader:  nodeID,
			Servers: []proto.NodeID{nodeID},
		},
	}
	err = peers.Sign(privateKey)
	return
}

// duplicate conf file using random new listen addr to avoid failure on concurrent test cases
func dupConf(confFile string, newConfFile string) (err error) {
	// replace port in confFile
	var fileBytes []byte
	if fileBytes, err = ioutil.ReadFile(confFile); err != nil {
		return
	}

	var ports []int
	if ports, err = utils.GetRandomPorts("127.0.0.1", 4000, 5000, 1); err != nil {
		return
	}

	newConfBytes := bytes.Replace(fileBytes, []byte(":2230"), []byte(fmt.Sprintf(":%v", ports[0])), -1)

	return ioutil.WriteFile(newConfFile, newConfBytes, 0644)
}
