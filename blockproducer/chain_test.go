/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package blockproducer

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/coreos/bbolt"
	"gitlab.com/thunderdb/ThunderDB/blockproducer/types"
	"gitlab.com/thunderdb/ThunderDB/proto"

	"gitlab.com/thunderdb/ThunderDB/kayak"

	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/rpc"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	testPeersNumber                           = 5
	testPeriod                                = 1 * time.Second
	testTick                                  = 100 * time.Millisecond
	testQueryTTL             int32            = 10
	testDatabaseID           proto.DatabaseID = "tdb-test"
	testChainService                          = "sql-chain.thunderdb.rpc"
	testPeriodNumber         uint64           = 10
	testClientNumberPerChain                  = 10
)

func TestChain(t *testing.T) {
	Convey("test main chain", t, func() {
		fl, err := ioutil.TempFile("", "mainchain")
		So(err, ShouldBeNil)

		fl.Close()

		// create genesis block
		genesis, err := generateRandomBlock(genesisHash, true)
		So(err, ShouldBeNil)

		pub, err := kms.GetLocalPublicKey()
		So(err, ShouldBeNil)

		priv, err := kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)

		servers := [...]*kayak.Server{
			&kayak.Server{ID: "X1"},
			&kayak.Server{ID: "X2"},
			&kayak.Server{ID: "X3"},
			&kayak.Server{ID: "X4"},
			&kayak.Server{ID: "X5"},
		}

		peers := &kayak.Peers{
			Term:    0,
			Leader:  servers[0],
			Servers: servers[:0],
			PubKey:  pub,
		}
		err = peers.Sign(priv)
		So(err, ShouldBeNil)

		cfg := newConfig(genesis, fl.Name(), rpc.NewServer(), peers, servers[0].ID, testPeriod, testTick)
		chain, err := NewChain(cfg)
		So(err, ShouldBeNil)

		// Hack for signle instance test
		chain.rt.bpNum = 5

		// Run main cycle
		var now time.Time
		var d time.Duration
		var height uint64 = 1

		for {
			t.Logf("Chain state: head = %s, height = %d, turn = %d, nextturnstart = %s, ismyturn = %t",
				chain.st.Head, chain.st.Height, chain.rt.nextTurn,
				chain.rt.chainInitTime.Add(
					chain.rt.period*time.Duration(chain.rt.nextTurn)).Format(time.RFC3339Nano),
				chain.rt.isMyTurn())
			now, d = chain.rt.nextTick()

			t.Logf("Wake up at: now = %s, d = %.9f secs",
				now.Format(time.RFC3339Nano), d.Seconds())

			if d > 0 {
				time.Sleep(d)
			} else {
				if err := chain.produceBlock(now); err != nil {
					So(err, ShouldBeNil)
				}

				break
			}

			if !chain.rt.isMyTurn() {
				// chain will receive blocks and tx

				// receive block
				// generate valid txbillings
				tbs := make([]*types.TxBilling, 10)
				for i := range tbs {
					tb, err := generateRandomTxBillingWithSeqID(0)
					So(err, ShouldBeNil)
					tbs[i] = tb
				}

				// generate block
				block, err := generateRandomBlockWithTxBillings(chain.st.Head, tbs)
				So(err, ShouldBeNil)
				err = chain.pushBlock(block)
				So(err, ShouldBeNil)
				for _, val := range tbs {
					So(chain.ti.hasTxBilling(val.TxHash), ShouldBeTrue)
				}
				So(chain.bi.hasBlock(block.SignedHeader.BlockHash), ShouldBeTrue)
				So(chain.st.Height, ShouldEqual, height)

				specificHeightBlock1, err := chain.fetchBlockByHeight(height)
				So(err, ShouldBeNil)
				So(block.SignedHeader.BlockHash, ShouldResemble, specificHeightBlock1.SignedHeader.BlockHash)
				specificHeightBlock2, err := chain.fetchBlockByHeight(height + 1000)
				So(specificHeightBlock2, ShouldBeNil)
				So(err, ShouldNotBeNil)

				// receive txes
				receivedTbs := make([]*types.TxBilling, 9)
				for i := range receivedTbs {
					tb, err := generateRandomTxBillingWithSeqID(0)
					So(err, ShouldBeNil)
					receivedTbs[i] = tb
					chain.pushTxBilling(tb)
				}

				for _, val := range receivedTbs {
					So(chain.ti.hasTxBilling(val.TxHash), ShouldBeTrue)
				}

				So(height, ShouldEqual, chain.st.Height)
				height += 1

				t.Logf("Pushed new block: height = %d, %s <- %s",
					chain.st.Height,
					block.SignedHeader.ParentHash,
					block.SignedHeader.BlockHash)
			} else {
				// chain will produce block
				var b types.Block
				var enc []byte
				err := chain.db.View(func(tx *bolt.Tx) error {
					enc = tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Get(chain.st.node.indexKey())
					return nil
				})
				So(err, ShouldBeNil)
				err = b.Deserialize(enc)
				So(err, ShouldBeNil)

				So(height, ShouldEqual, chain.st.Height)
				height += 1

				t.Logf("Produced new block: height = %d, %s <- %s",
					chain.st.Height,
					b.SignedHeader.ParentHash,
					b.SignedHeader.BlockHash)
			}

			if chain.st.Height >= testPeriodNumber {
				break
			}
		}

		// load chain from db
		chain.db.Close()
		_, err = LoadChain(cfg)
		So(err, ShouldBeNil)
	})
}
