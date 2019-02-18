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

package main

import (
	"bytes"
	"io/ioutil"
	"os"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/worker"
	"github.com/pkg/errors"
)

var rootHash = hash.Hash{}

func startDBMS(server *rpc.Server, onCreateDB func()) (dbms *worker.DBMS, err error) {
	if conf.GConf.Miner == nil {
		err = errors.New("invalid database config")
		return
	}

	cfg := &worker.DBMSConfig{
		RootDir:          conf.GConf.Miner.RootDir,
		Server:           server,
		MaxReqTimeGap:    conf.GConf.Miner.MaxReqTimeGap,
		OnCreateDatabase: onCreateDB,
	}

	if dbms, err = worker.NewDBMS(cfg); err != nil {
		err = errors.Wrap(err, "create new DBMS failed")
		return
	}

	if err = dbms.Init(); err != nil {
		err = errors.Wrap(err, "init DBMS failed")
		return
	}

	// add test fixture database
	if conf.GConf.Miner.IsTestMode {
		// in test mode
		var privKey *asymmetric.PrivateKey

		if privKey, err = kms.GetLocalPrivateKey(); err != nil {
			err = errors.Wrap(err, "get local private key failed")
			return
		}

		// add database to miner
		for _, testFixture := range conf.GConf.Miner.TestFixtures {
			// build test db instance configuration
			dbPeers := &proto.Peers{
				PeersHeader: proto.PeersHeader{
					Term:    testFixture.Term,
					Leader:  testFixture.Leader,
					Servers: testFixture.Servers,
				},
			}

			if err = dbPeers.Sign(privKey); err != nil {
				err = errors.Wrap(err, "sign peers failed")
				return
			}

			// load genesis block
			var block *types.Block
			if block, err = loadGenesisBlock(testFixture); err != nil {
				err = errors.Wrap(err, "load genesis block failed")
				return
			}

			// add to dbms
			instance := &types.ServiceInstance{
				DatabaseID:   testFixture.DatabaseID,
				Peers:        dbPeers,
				GenesisBlock: block,
			}
			if err = dbms.Create(instance, false); err != nil {
				err = errors.Wrap(err, "add new DBMS failed")
				return
			}
		}
	}

	return
}

func loadGenesisBlock(fixture *conf.MinerDatabaseFixture) (block *types.Block, err error) {
	if fixture.GenesisBlockFile == "" {
		err = os.ErrNotExist
		return
	}

	var blockBytes []byte
	if blockBytes, err = ioutil.ReadFile(fixture.GenesisBlockFile); err == nil {
		err = errors.Wrap(err, "read block failed")
		return
	}

	if os.IsNotExist(err) && fixture.AutoGenerateGenesisBlock {
		// generate
		if block, err = types.CreateRandomBlock(rootHash, true); err != nil {
			err = errors.Wrap(err, "create random block failed")
			return
		}

		// encode block
		var bytesBuffer *bytes.Buffer
		if bytesBuffer, err = utils.EncodeMsgPack(block); err != nil {
			err = errors.Wrap(err, "encode block failed")
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
