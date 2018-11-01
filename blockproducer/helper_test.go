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

package blockproducer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/metric"
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
	PubKeyStorePath = "./pubkey.store"
)

var (
	rootHash = hash.Hash{}
)

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

// fake a persistence driver.
type stubDBMetaPersistence struct{}

func (p *stubDBMetaPersistence) GetDatabase(dbID proto.DatabaseID) (instance wt.ServiceInstance, err error) {
	// for test purpose, name with db prefix consider it exists
	if !strings.HasPrefix(string(dbID), "db") {
		err = ErrNoSuchDatabase
		return
	}

	return p.getInstanceMeta(dbID)
}

func (p *stubDBMetaPersistence) SetDatabase(meta wt.ServiceInstance) (err error) {
	return
}

func (p *stubDBMetaPersistence) DeleteDatabase(dbID proto.DatabaseID) (err error) {
	return
}

func (p *stubDBMetaPersistence) GetAllDatabases() (instances []wt.ServiceInstance, err error) {
	instances = make([]wt.ServiceInstance, 1)
	instances[0], err = p.getInstanceMeta("db")
	return
}

func (p *stubDBMetaPersistence) getInstanceMeta(dbID proto.DatabaseID) (instance wt.ServiceInstance, err error) {
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

func initNode(confRP, privateKeyRP string) (cleanupFunc func(), dht *route.DHTService, metricService *metric.CollectServer, server *rpc.Server, err error) {
	var d string
	if d, err = ioutil.TempDir("", "db_test_"); err != nil {
		return
	}
	log.WithField("d", d).Debug("created temp dir")

	// init conf
	_, testFile, _, _ := runtime.Caller(0)
	pubKeyStoreFile := filepath.Join(d, PubKeyStorePath)
	os.Remove(pubKeyStoreFile)
	clientPubKeyStoreFile := filepath.Join(d, PubKeyStorePath+"_c")
	os.Remove(clientPubKeyStoreFile)
	dupConfFile := filepath.Join(d, "config.yaml")
	confFile := filepath.Join(filepath.Dir(testFile), confRP)
	if err = dupConf(confFile, dupConfFile); err != nil {
		return
	}
	privateKeyPath := filepath.Join(filepath.Dir(testFile), privateKeyRP)

	conf.GConf, _ = conf.LoadConfig(dupConfFile)
	log.Debugf("GConf: %#v", conf.GConf)
	// reset the once
	route.Once = sync.Once{}
	route.InitKMS(clientPubKeyStoreFile)

	// init dht
	dht, err = route.NewDHTService(pubKeyStoreFile, new(consistent.KMSStorage), true)
	if err != nil {
		return
	}

	// init rpc
	if server, err = rpc.NewServerWithService(rpc.ServiceMap{route.DHTRPCName: dht}); err != nil {
		return
	}

	// register metric service
	metricService = metric.NewCollectServer()
	if err = server.RegisterService(metric.MetricServiceName, metricService); err != nil {
		log.WithError(err).Error("init metric service failed")
		return
	}

	// register database service
	_, err = worker.NewDBMS(&worker.DBMSConfig{
		RootDir:       d,
		Server:        server,
		MaxReqTimeGap: worker.DefaultMaxReqTimeGap,
	})

	// init private key
	masterKey := []byte("")
	if err = server.InitRPCServer(conf.GConf.ListenAddr, privateKeyPath, masterKey); err != nil {
		return
	}

	// start server
	go server.Serve()

	cleanupFunc = func() {
		os.RemoveAll(d)
		server.Listener.Close()
		server.Stop()
		rpc.GetSessionPoolInstance().Close()
	}

	return
}

// duplicate conf file using random new listen addr to avoid failure on concurrent test cases.
func dupConf(confFile string, newConfFile string) (err error) {
	// replace port in confFile
	var fileBytes []byte
	if fileBytes, err = ioutil.ReadFile(confFile); err != nil {
		return
	}

	var ports []int
	if ports, err = utils.GetRandomPorts("127.0.0.1", 3000, 4000, 1); err != nil {
		return
	}

	newConfBytes := bytes.Replace(fileBytes, []byte(":2230"), []byte(fmt.Sprintf(":%v", ports[0])), -1)

	return ioutil.WriteFile(newConfFile, newConfBytes, 0644)
}
