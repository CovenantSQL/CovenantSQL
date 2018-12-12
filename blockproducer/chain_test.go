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
		Convey("When transfer transactions are added", func() {
			var (
				nonce pi.AccountNonce
				t1    pi.Transaction
			)
			nonce, err = chain.rt.nextNonce(addr1)
			So(err, ShouldBeNil)
			t1, err = newTransfer(nonce, priv1, addr1, addr2, 1)
			So(err, ShouldBeNil)
			err = chain.rt.addTx(chain.st, t1)
			So(err, ShouldBeNil)
		})
	})
}
