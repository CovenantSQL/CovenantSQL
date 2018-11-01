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
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	pt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testPeersNumber                 = 1
	testPeriod                      = 1 * time.Second
	testTick                        = 100 * time.Millisecond
	testPeriodNumber         uint32 = 10
	testClientNumberPerChain        = 10
)

func TestChain(t *testing.T) {
	Convey("test main chain", t, func() {
		log.SetLevel(log.InfoLevel)
		confDir := "../test/mainchain/node_standalone/config.yaml"
		privDir := "../test/mainchain/node_standalone/private.key"
		cleanup, _, _, rpcServer, err := initNode(
			confDir,
			privDir,
		)
		defer cleanup()
		So(err, ShouldBeNil)

		fl, err := ioutil.TempFile("", "mainchain")
		So(err, ShouldBeNil)

		fl.Close()
		os.Remove(fl.Name())

		// create genesis block
		genesis, err := generateRandomBlock(genesisHash, true)
		So(err, ShouldBeNil)

		priv, err := kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)
		_, peers, err := createTestPeersWithPrivKeys(priv, testPeersNumber)

		cfg := NewConfig(genesis, fl.Name(), rpcServer, peers, peers.Servers[0], testPeriod, testTick)
		chain, err := NewChain(cfg)
		So(err, ShouldBeNil)
		ao, ok := chain.ms.readonly.accounts[testAddress1]
		So(ok, ShouldBeTrue)
		So(ao, ShouldNotBeNil)
		So(chain.ms.pool.entries[testAddress1].transactions, ShouldBeEmpty)
		So(chain.ms.pool.entries[testAddress1].baseNonce, ShouldEqual, 1)
		var (
			bl     uint64
			loaded bool
		)
		bl, loaded = chain.ms.loadAccountStableBalance(testAddress1)
		So(loaded, ShouldBeTrue)
		So(bl, ShouldEqual, testInitBalance)
		bl, loaded = chain.ms.loadAccountStableBalance(testAddress2)
		So(loaded, ShouldBeTrue)
		So(bl, ShouldEqual, testInitBalance)
		bl, loaded = chain.ms.loadAccountCovenantBalance(testAddress1)
		So(loaded, ShouldBeTrue)
		So(bl, ShouldEqual, testInitBalance)
		bl, loaded = chain.ms.loadAccountCovenantBalance(testAddress2)
		So(loaded, ShouldBeTrue)
		So(bl, ShouldEqual, testInitBalance)

		// Hack for single instance test
		chain.rt.bpNum = 5

		for {
			time.Sleep(testPeriod)
			t.Logf("Chain state: head = %s, height = %d, turn = %d, nextturnstart = %s, ismyturn = %t",
				chain.rt.getHead().getHeader(), chain.rt.getHead().getHeight(), chain.rt.nextTurn,
				chain.rt.chainInitTime.Add(
					chain.rt.period*time.Duration(chain.rt.nextTurn)).Format(time.RFC3339Nano),
				chain.rt.isMyTurn())

			// chain will receive blocks and tx
			// receive block
			// generate valid txbillings
			tbs := make([]pi.Transaction, 0, 20)

			// pull previous processed transactions
			tbs = append(tbs, chain.ms.pullTxs()...)

			for i := 0; i != 10; i++ {
				tb, err := generateRandomAccountBilling()
				So(err, ShouldBeNil)
				tbs = append(tbs, tb)
			}

			// generate block
			block, err := generateRandomBlockWithTransactions(*chain.rt.getHead().getHeader(), tbs)
			So(err, ShouldBeNil)
			err = chain.pushBlock(block)
			So(err, ShouldBeNil)
			nextNonce, err := chain.ms.nextNonce(testAddress1)
			So(err, ShouldBeNil)
			for _, val := range tbs {
				// should be packed
				So(nextNonce >= val.GetAccountNonce(), ShouldBeTrue)
			}
			So(chain.bi.hasBlock(block.SignedHeader.BlockHash), ShouldBeTrue)
			// So(chain.rt.getHead().Height, ShouldEqual, height)

			height := chain.rt.getHead().getHeight()
			specificHeightBlock1, _, err := chain.fetchBlockByHeight(height)
			So(err, ShouldBeNil)
			So(block.SignedHeader.BlockHash, ShouldResemble, specificHeightBlock1.SignedHeader.BlockHash)
			specificHeightBlock2, _, err := chain.fetchBlockByHeight(height + 1000)
			So(specificHeightBlock2, ShouldBeNil)
			So(err, ShouldNotBeNil)

			// receive txs
			receivedTbs := make([]*pt.Billing, 9)
			for i := range receivedTbs {
				tb, err := generateRandomAccountBilling()
				So(err, ShouldBeNil)
				receivedTbs[i] = tb
				err = chain.processTx(tb)
				So(err, ShouldBeNil)
			}

			nextNonce, err = chain.ms.nextNonce(testAddress1)

			for _, val := range receivedTbs {
				// should be packed or unpacked
				So(chain.ms.pool.hasTx(val), ShouldBeTrue)
			}

			// So(height, ShouldEqual, chain.rt.getHead().Height)
			height++

			t.Logf("Pushed new block: height = %d, %s <- %s",
				chain.rt.getHead().getHeight(),
				block.ParentHash(),
				block.BlockHash())

			if chain.rt.getHead().getHeight() >= testPeriodNumber {
				break
			}
		}

		// load chain from db
		chain.db.Close()
		_, err = LoadChain(cfg)
		So(err, ShouldBeNil)
	})
}

func TestMultiNode(t *testing.T) {
	Convey("test multi-nodes", t, func(c C) {
		// create genesis block
		genesis, err := generateRandomBlock(genesisHash, true)
		So(err, ShouldBeNil)
		So(genesis.Transactions, ShouldNotBeEmpty)

		// Create sql-chain instances
		chains := make([]*Chain, testPeersNumber)
		configs := []string{
			"../test/mainchain/node_multi_0/config.yaml",
			// "../test/mainchain/node_multi_1/config.yaml",
			// "../test/mainchain/node_multi_2/config.yaml",
		}
		privateKeys := []string{
			"../test/mainchain/node_multi_0/private.key",
			// "../test/mainchain/node_multi_1/private.key",
			// "../test/mainchain/node_multi_2/private.key",
		}

		var nis []cpuminer.NonceInfo
		var peers *proto.Peers
		peerInited := false
		for i := range chains {
			// create tmp file
			fl, err := ioutil.TempFile("", "mainchain")
			So(err, ShouldBeNil)
			fl.Close()
			os.Remove(fl.Name())

			// init config
			cleanup, dht, _, server, err := initNode(configs[i], privateKeys[i])
			So(err, ShouldBeNil)
			defer cleanup()

			// Create peer list
			if !peerInited {
				nis, peers, err = createTestPeers(testPeersNumber)
				So(err, ShouldBeNil)

				for i, p := range peers.Servers {
					t.Logf("Peer #%d: %s", i, p)
				}

				peerInited = true
			}

			cfg := NewConfig(genesis, fl.Name(), server, peers, peers.Servers[i], testPeriod, testTick)

			// init chain
			chains[i], err = NewChain(cfg)
			So(err, ShouldBeNil)

			// Register address
			pub, err := kms.GetLocalPublicKey()
			So(err, ShouldBeNil)
			node := proto.Node{
				ID: peers.Servers[i],
				Role: func(peers *proto.Peers, i int) proto.ServerRole {
					if peers.Leader.IsEqual(&peers.Servers[i]) {
						return proto.Leader
					} else {
						return proto.Follower
					}
				}(peers, i),
				Addr:      server.Listener.Addr().String(),
				PublicKey: pub,
				Nonce:     nis[i].Nonce,
			}
			req := proto.PingReq{
				Node:     node,
				Envelope: proto.Envelope{},
			}
			var resp proto.PingResp
			dht.Ping(&req, &resp)
			log.WithField("resp", resp).Debug("got ping response")

			err = chains[i].Start()
			So(err, ShouldBeNil)
			defer func(c *Chain) {
				c.Stop()
			}(chains[i])
		}

		for i := range chains {
			wg := &sync.WaitGroup{}
			sC := make(chan struct{})

			for j := 0; j < testClientNumberPerChain; j++ {
				wg.Add(1)
				go func(val int) {
					defer wg.Done()
				foreverLoop:
					for {
						select {
						case <-sC:
							break foreverLoop
						default:
							// test AdviseBillingRequest RPC
							br, err := generateRandomBillingRequest()
							c.So(err, ShouldBeNil)

							bReq := &ct.AdviseBillingReq{
								Envelope: proto.Envelope{
									// TODO(lambda): Add fields.
								},
								Req: br,
							}
							bResp := &ct.AdviseBillingResp{}
							log.WithFields(log.Fields{
								"node":        val,
								"requestHash": br.RequestHash,
							}).Debug("advising billing request")
							err = chains[i].cl.CallNode(chains[i].rt.nodeID, route.MCCAdviseBillingRequest.String(), bReq, bResp)
							if err != nil {
								log.WithFields(log.Fields{
									"peer":         chains[i].rt.getPeerInfoString(),
									"curr_turn":    chains[i].rt.getNextTurn(),
									"now_time":     time.Now().UTC().Format(time.RFC3339Nano),
									"request_hash": br.RequestHash,
								}).WithError(err).Error("Failed to advise new billing request")
							}
							// TODO(leventeliu): this test needs to be improved using some preset
							// accounts. Or this request will return an "ErrAccountNotFound" error.
							c.So(err, ShouldNotBeNil)
							c.So(err.Error(), ShouldEqual, ErrAccountNotFound.Error())
							//log.Debugf("response %d hash is %s", val, bResp.Resp.RequestHash)

						}
					}
				}(j)
			}
			defer func() {
				close(sC)
				wg.Wait()
			}()
		}
		time.Sleep(time.Duration(testPeriodNumber) * testPeriod)
	})

	return
}
