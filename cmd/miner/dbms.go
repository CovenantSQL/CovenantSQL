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
	"bytes"
	"errors"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/worker"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

var rootHash = hash.Hash{}

func startDBMS(server *rpc.Server) (dbms *worker.DBMS, err error) {
	if conf.GConf.Miner == nil {
		err = errors.New("invalid database config")
		return
	}

	cfg := &worker.DBMSConfig{
		RootDir:       conf.GConf.Miner.RootDir,
		Server:        server,
		MaxReqTimeGap: conf.GConf.Miner.MaxReqTimeGap,
	}

	if dbms, err = worker.NewDBMS(cfg); err != nil {
		return
	}

	if err = dbms.Init(); err != nil {
		return
	}

	// add test fixture database
	if conf.GConf.Miner.IsTestMode {
		// in test mode

		var pubKey *asymmetric.PublicKey
		var privKey *asymmetric.PrivateKey

		if pubKey, err = kms.GetLocalPublicKey(); err != nil {
			return
		}
		if privKey, err = kms.GetLocalPrivateKey(); err != nil {
			return
		}

		// add database to miner
		for _, testFixture := range conf.GConf.Miner.TestFixtures {
			// build test db instance configuration
			dbPeers := &kayak.Peers{
				Term: testFixture.Term,
				Leader: &kayak.Server{
					Role: proto.Leader,
					ID:   testFixture.Leader,
				},
				Servers: (func(servers []proto.NodeID) (ks []*kayak.Server) {
					ks = make([]*kayak.Server, len(servers))

					for i, s := range servers {
						ks[i] = &kayak.Server{
							Role: proto.Follower,
							ID:   s,
						}
						if s == testFixture.Leader {
							ks[i].Role = proto.Leader
						}
					}

					return
				})(testFixture.Servers),
				PubKey: pubKey,
			}

			if err = dbPeers.Sign(privKey); err != nil {
				return
			}

			// load genesis block
			var block *ct.Block
			if block, err = loadGenesisBlock(testFixture); err != nil {
				return
			}

			// add to dbms
			instance := &wt.ServiceInstance{
				DatabaseID:   testFixture.DatabaseID,
				Peers:        dbPeers,
				GenesisBlock: block,
			}
			if err = dbms.Create(instance, false); err != nil {
				return
			}
		}
	}

	return
}

func loadGenesisBlock(fixture *conf.MinerDatabaseFixture) (block *ct.Block, err error) {
	if fixture.GenesisBlockFile == "" {
		err = os.ErrNotExist
		return
	}

	var blockBytes []byte
	if blockBytes, err = ioutil.ReadFile(fixture.GenesisBlockFile); err == nil {
		return
	}

	if os.IsNotExist(err) && fixture.AutoGenerateGenesisBlock {
		// generate
		if block, err = createRandomBlock(rootHash, true); err != nil {
			return
		}

		// encode block
		var bytesBuffer *bytes.Buffer
		if bytesBuffer, err = utils.EncodeMsgPack(block); err != nil {
			return
		}

		blockBytes = bytesBuffer.Bytes()

		// write to file
		err = ioutil.WriteFile(fixture.GenesisBlockFile, blockBytes, 0644)
	} else {
		err = utils.DecodeMsgPack(blockBytes, &block)
	}

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
			Signee:    pub,
			Signature: nil,
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
		}, cpuminer.Uint256{}, 4)
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
