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
	"os"
	"path"
	"testing"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testPeersNumber                 = 1
	testPeriod                      = 1 * time.Second
	testTick                        = 100 * time.Millisecond
	testPeriodNumber         uint32 = 10
	testClientNumberPerChain        = 10
)

func newTransfer(
	nonce pi.AccountNonce, signer *asymmetric.PrivateKey,
	sender, receiver proto.AccountAddress, amount uint64,
) (
	t *types.Transfer, err error,
) {
	t = types.NewTransfer(&types.TransferHeader{
		Sender:   sender,
		Receiver: receiver,
		Nonce:    nonce,
		Amount:   amount,
	})
	err = t.Sign(signer)
	return
}

func TestChain(t *testing.T) {
	Convey("Given a new block producer chain", t, func() {
		var (
			rawids = [...]proto.RawNodeID{
				{Hash: hash.Hash{0x0, 0x0, 0x0, 0x1}},
				{Hash: hash.Hash{0x0, 0x0, 0x0, 0x2}},
				{Hash: hash.Hash{0x0, 0x0, 0x0, 0x3}},
				{Hash: hash.Hash{0x0, 0x0, 0x0, 0x4}},
				{Hash: hash.Hash{0x0, 0x0, 0x0, 0x5}},
			}

			err     error
			config  *Config
			genesis *types.BPBlock
			begin   time.Time
			leader  proto.NodeID
			servers []proto.NodeID
			chain   *Chain

			priv1, priv2 *asymmetric.PrivateKey
			addr1, addr2 proto.AccountAddress
		)

		priv1, err = kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)
		priv2, _, err = asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		addr1, err = crypto.PubKeyHash(priv1.PubKey())
		So(err, ShouldBeNil)
		addr2, err = crypto.PubKeyHash(priv2.PubKey())

		genesis = &types.BPBlock{
			SignedHeader: types.BPSignedHeader{
				BPHeader: types.BPHeader{
					Timestamp: time.Now().UTC(),
				},
			},
			Transactions: []pi.Transaction{
				types.NewBaseAccount(&types.Account{
					Address:      addr1,
					TokenBalance: [5]uint64{1000, 1000, 1000, 1000, 1000},
				}),
			},
		}
		err = genesis.PackAndSignBlock(testingPrivateKey)
		So(err, ShouldBeNil)
		begin = genesis.Timestamp()

		for _, v := range rawids {
			servers = append(servers, v.ToNodeID())
		}
		leader = servers[0]

		config = &Config{
			Genesis:  genesis,
			DataFile: path.Join(testingDataDir, t.Name()),
			Server:   nil,
			Peers: &proto.Peers{
				PeersHeader: proto.PeersHeader{
					Leader:  leader,
					Servers: servers,
				},
			},
			NodeID: leader,
			Period: time.Duration(1 * time.Second),
			Tick:   time.Duration(100 * time.Millisecond),
		}

		chain, err = NewChain(config)
		So(err, ShouldBeNil)
		So(chain, ShouldNotBeNil)

		// Close chain on reset
		Reset(func() {
			if chain != nil {
				err = chain.Stop()
				So(err, ShouldBeNil)
			}
			err = os.Remove(config.DataFile)
			So(err, ShouldBeNil)
		})

		Convey("When transfer transactions are added", func() {
			var (
				nonce      pi.AccountNonce
				t1, t2, t3 pi.Transaction
				f0, f1     *branch
				bl         *types.BPBlock
			)
			nonce, err = chain.rt.nextNonce(addr1)
			So(err, ShouldBeNil)
			So(nonce, ShouldEqual, 1)
			t1, err = newTransfer(nonce, priv1, addr1, addr2, 1)
			So(err, ShouldBeNil)
			t2, err = newTransfer(nonce+1, priv1, addr1, addr2, 1)
			So(err, ShouldBeNil)
			t3, err = newTransfer(nonce+2, priv1, addr1, addr2, 1)
			So(err, ShouldBeNil)

			// Fork from #0
			f0 = chain.rt.headBranch.makeCopy()

			err = chain.rt.addTx(chain.st, t1)
			So(err, ShouldBeNil)
			Convey("The chain should report error on duplicated transaction", func() {
				err = chain.rt.addTx(chain.st, t1)
				So(err, ShouldEqual, ErrExistedTx)
			})
			err = chain.produceBlock(begin.Add(chain.rt.period))
			So(err, ShouldBeNil)

			// Create a sibling block from fork#0 and apply
			_, bl, err = f0.produceBlock(2, begin.Add(2*chain.rt.period), addr2, priv2)
			So(err, ShouldBeNil)
			So(bl, ShouldNotBeNil)
			err = chain.pushBlock(bl)
			So(err, ShouldBeNil)

			// Fork from #1
			f1 = chain.rt.headBranch.makeCopy()

			err = chain.rt.addTx(chain.st, t2)
			So(err, ShouldBeNil)
			err = chain.produceBlock(begin.Add(2 * chain.rt.period))
			So(err, ShouldBeNil)

			err = chain.rt.addTx(chain.st, t3)
			So(err, ShouldBeNil)
			err = chain.produceBlock(begin.Add(3 * chain.rt.period))
			So(err, ShouldBeNil)
			// Create a sibling block from fork#1 and apply
			f1, bl, err = f1.produceBlock(3, begin.Add(3*chain.rt.period), addr2, priv2)
			So(err, ShouldBeNil)
			So(bl, ShouldNotBeNil)
			err = chain.pushBlock(bl)
			So(err, ShouldBeNil)

			// This should trigger a branch pruning on fork #0
			for i := uint32(4); i <= 6; i++ {
				err = chain.produceBlock(begin.Add(time.Duration(i) * chain.rt.period))
				So(err, ShouldBeNil)
				// Create a sibling block from fork#1 and apply
				f1, bl, err = f1.produceBlock(
					i, begin.Add(time.Duration(i)*chain.rt.period), addr2, priv2)
				So(err, ShouldBeNil)
				So(bl, ShouldNotBeNil)
				err = chain.pushBlock(bl)
				So(err, ShouldBeNil)
			}

			// Add 2 more blocks to fork #1, this should trigger a branch switch to fork #1
			f1, bl, err = f1.produceBlock(7, begin.Add(8*chain.rt.period), addr2, priv2)
			So(err, ShouldBeNil)
			So(bl, ShouldNotBeNil)
			err = chain.pushBlock(bl)
			So(err, ShouldBeNil)
			f1, bl, err = f1.produceBlock(8, begin.Add(9*chain.rt.period), addr2, priv2)
			So(err, ShouldBeNil)
			So(bl, ShouldNotBeNil)
			err = chain.pushBlock(bl)
			So(err, ShouldBeNil)

			Convey("The chain should have same state after reloading", func() {
				err = chain.Stop()
				So(err, ShouldBeNil)
				chain, err = NewChain(config)
				So(err, ShouldBeNil)
				So(chain, ShouldNotBeNil)
				chain.rt.log()
			})
		})
	})
}
