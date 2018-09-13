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
	"math"
	"path"
	"testing"

	pt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/coreos/bbolt"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMetaState(t *testing.T) {
	Convey("Given a new metaState object", t, func() {
		var (
			ms   = newMetaState()
			addr = proto.AccountAddress(generateRandomHash())
			dbid = proto.DatabaseID("db#1")
		)
		Convey("The account state should be empty", func() {
			var o, loaded = ms.loadAccountObject(addr)
			So(o, ShouldBeNil)
			So(loaded, ShouldBeFalse)
		})
		Convey("The database state should be empty", func() {
			var o, loaded = ms.loadSQLChainObject(dbid)
			So(o, ShouldBeNil)
			So(loaded, ShouldBeFalse)
		})
		Convey("When a new account object is stored", func() {
			var o, loaded = ms.loadOrStoreAccountObject(addr, &accountObject{
				Account: pt.Account{
					Address: addr,
				},
			})
			So(o, ShouldBeNil)
			So(loaded, ShouldEqual, false)
			Convey("The state should include the account", func() {
				o, loaded = ms.loadAccountObject(addr)
				So(o, ShouldNotBeNil)
				So(o.Address, ShouldEqual, addr)
				So(loaded, ShouldBeTrue)
				o, loaded = ms.loadOrStoreAccountObject(addr, nil)
				So(o, ShouldNotBeNil)
				So(o.Address, ShouldEqual, addr)
				So(loaded, ShouldBeTrue)
			})
			Convey("When the account balance is increased", func() {
				var (
					err    error
					incSta uint64 = 100
					decSta uint64 = 10
					incCov uint64 = 1000
					decCov uint64 = 100
				)
				err = ms.increaseAccountStableBalance(addr, incSta)
				So(err, ShouldBeNil)
				err = ms.increaseAccountcovenantBalance(addr, incCov)
				So(err, ShouldBeNil)
				Convey("When the account balance is increased by an impossible amount", func() {
					err = ms.increaseAccountStableBalance(addr, math.MaxUint64)
					So(err, ShouldEqual, ErrBalanceOverflow)
					err = ms.increaseAccountcovenantBalance(addr, math.MaxUint64)
					So(err, ShouldEqual, ErrBalanceOverflow)
				})
				Convey("When the account balance is decreased by an impossible amount", func() {
					err = ms.decreaseAccountStableBalance(addr, incSta+1)
					So(err, ShouldEqual, ErrInsufficientBalance)
					err = ms.decreaseAccountCovenantBalance(addr, incCov+1)
					So(err, ShouldEqual, ErrInsufficientBalance)
				})
				Convey("The account balance should be kept correctly in account object", func() {
					o, loaded = ms.loadAccountObject(addr)
					So(loaded, ShouldBeTrue)
					So(o, ShouldNotBeNil)
					So(o.Address, ShouldEqual, addr)
					So(o.StableCoinBalance, ShouldEqual, incSta)
					So(o.CovenantCoinBalance, ShouldEqual, incCov)
				})
				Convey("When the account balance is decreased", func() {
					err = ms.decreaseAccountStableBalance(addr, decSta)
					So(err, ShouldBeNil)
					err = ms.decreaseAccountCovenantBalance(addr, decCov)
					So(err, ShouldBeNil)
					Convey(
						"The account balance should still be kept correctly in account object",
						func() {
							o, loaded = ms.loadAccountObject(addr)
							So(loaded, ShouldBeTrue)
							So(o, ShouldNotBeNil)
							So(o.Address, ShouldEqual, addr)
							So(o.StableCoinBalance, ShouldEqual, incSta-decSta)
							So(o.CovenantCoinBalance, ShouldEqual, incCov-decCov)
						},
					)
				})
			})
			Convey("When the account object is deleted", func() {
				ms.deleteAccountObject(addr)
				Convey("The state should not include the account state anymore", func() {
					var o, loaded = ms.loadAccountObject(addr)
					So(o, ShouldBeNil)
					So(loaded, ShouldBeFalse)
				})
			})
		})
		Convey("When a new database object is stored", func() {
			var o, loaded = ms.loadOrStoreSQLChainObject(dbid, &sqlchainObject{
				SQLChainProfile: pt.SQLChainProfile{
					ID: dbid,
				},
			})
			So(o, ShouldBeNil)
			So(loaded, ShouldEqual, false)
			Convey("The state should include the sqlchain profile", func() {
				o, loaded = ms.loadSQLChainObject(dbid)
				So(o, ShouldNotBeNil)
				So(o.ID, ShouldEqual, dbid)
				So(loaded, ShouldBeTrue)
				o, loaded = ms.loadOrStoreSQLChainObject(dbid, nil)
				So(o, ShouldNotBeNil)
				So(o.ID, ShouldEqual, dbid)
				So(loaded, ShouldBeTrue)
			})
			Convey("When the database object is deleted", func() {
				ms.deleteSQLChainObject(dbid)
				Convey("The state should not include the database state anymore", func() {
					var o, loaded = ms.loadSQLChainObject(dbid)
					So(o, ShouldBeNil)
					So(loaded, ShouldBeFalse)
				})
			})
		})
		Convey("When all the above modification are reset", func() {
			ms.clean()
			Convey("The account state should be empty", func() {
				var o, loaded = ms.loadAccountObject(addr)
				So(o, ShouldBeNil)
				So(loaded, ShouldBeFalse)
			})
			Convey("The database state should be empty", func() {
				var o, loaded = ms.loadSQLChainObject(dbid)
				So(o, ShouldBeNil)
				So(loaded, ShouldBeFalse)
			})
		})
	})
	Convey("Given a new metaState object and a persistence db instance", t, func() {
		var (
			ms      = newMetaState()
			addr1   = proto.AccountAddress(generateRandomHash())
			addr2   = proto.AccountAddress(generateRandomHash())
			fl      = path.Join(testDataDir, t.Name())
			db, err = bolt.Open(fl, 0600, nil)
		)
		So(err, ShouldBeNil)
		err = db.Update(func(tx *bolt.Tx) (err error) {
			var meta *bolt.Bucket
			if meta, err = tx.CreateBucket(metaBucket[:]); err != nil {
				return
			}
			if _, err = meta.CreateBucket(metaAccountIndexBucket); err != nil {
				return
			}
			if _, err = meta.CreateBucket(metaSQLChainIndexBucket); err != nil {
				return
			}
			return
		})
		So(err, ShouldBeNil)
		Convey("When new account objects are stored", func() {
			var (
				o      *accountObject
				loaded bool
			)
			o, loaded = ms.loadOrStoreAccountObject(addr1, &accountObject{
				Account: pt.Account{
					Address: addr1,
				},
			})
			So(o, ShouldBeNil)
			So(loaded, ShouldEqual, false)
			o, loaded = ms.loadOrStoreAccountObject(addr2, &accountObject{
				Account: pt.Account{
					Address: addr2,
				},
			})
			So(o, ShouldBeNil)
			So(loaded, ShouldEqual, false)
			Convey("When metaState change is committed", func() {
				var err = db.Update(ms.commitProcedure())
				So(err, ShouldBeNil)
			})
		})

	})
}
