/*
 *  Copyright 2018 The CovenantSQL Authors.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package worker

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewBusService(t *testing.T) {
	Convey("Create a BusService with mock bp", t, func() {
		var (
			err     error
			cleanup func()
		)
		cleanup, _, err = initNode()
		So(err, ShouldBeNil)

		var (
			privKey           *asymmetric.PrivateKey
			pubKey            *asymmetric.PublicKey
			addr              proto.AccountAddress
			testCheckInterval = 1 * time.Second
		)
		privKey, err = kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)
		pubKey = privKey.PubKey()
		addr, err = crypto.PubKeyHash(pubKey)
		So(err, ShouldBeNil)
		ctx, _ := context.WithCancel(context.Background())
		bs := NewBusService(ctx, addr, testCheckInterval)
		topic := fmt.Sprintf("/%s/", testOddBlocks.Transactions[0].GetTransactionType().String())
		var count uint32 = 0
		err = bs.Subscribe(topic, func(tx interfaces.Transaction, c uint32) {
			atomic.AddUint32(&count, 1)
		})
		So(err, ShouldBeNil)
		bs.extractTxs(&testEventBlocks, 1)
		So(count, ShouldEqual, len(testEventBlocks.Transactions))

		bs.Start()

		time.Sleep(2 * time.Second)

		c := atomic.LoadUint32(&bs.blockCount)
		if c%2 == 0 {
			p, ok := bs.RequestSQLProfile(&testEventID)
			So(ok, ShouldBeTrue)
			exist := false
			for _, profile := range testEventProfiles {
				if profile.ID == p.ID {
					So(p, ShouldResemble, profile)
					exist = true
				}
			}
			So(exist, ShouldBeTrue)
			dbMap := bs.GetCurrentDBMapping()
			for _, profile := range testEventProfiles {
				p, ok := dbMap[profile.ID]
				So(ok, ShouldBeTrue)
				So(profile, ShouldResemble, p)
			}
			p, ok = bs.RequestSQLProfile(&testOddID)
			So(ok, ShouldBeFalse)
			So(p, ShouldBeNil)
		} else {
			p, ok := bs.RequestSQLProfile(&testOddID)
			So(ok, ShouldBeTrue)
			exist := false
			for _, profile := range testOddProfiles {
				if profile.ID == p.ID {
					So(p, ShouldResemble, profile)
					exist = true
				}
			}
			So(exist, ShouldBeTrue)
			dbMap := bs.GetCurrentDBMapping()
			for _, profile := range testOddProfiles {
				p, ok := dbMap[profile.ID]
				So(ok, ShouldBeTrue)
				So(profile, ShouldResemble, p)
			}
			p, ok = bs.RequestSQLProfile(&testEventID)
			So(ok, ShouldBeFalse)
			So(p, ShouldBeNil)
		}

		b, err := bs.fetchBlockByCount(1)
		So(err, ShouldBeNil)
		So(len(b.Transactions), ShouldEqual, len(testOddBlocks.Transactions))
		b, err = bs.fetchBlockByCount(0)
		So(err, ShouldBeNil)
		So(len(b.Transactions), ShouldEqual, len(testEventBlocks.Transactions))
		b, err = bs.fetchBlockByCount(10000)
		So(err.Error(), ShouldEqual, ErrNotExists.Error())
		So(b, ShouldBeNil)

		bs.Stop()

		cleanup()
	})
}
