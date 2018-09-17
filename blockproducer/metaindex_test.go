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

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	pt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/coreos/bbolt"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMetaIndex(t *testing.T) {
	Convey("Given a new metaIndex object and persistence db instance", t, func() {
		var (
			ao      *accountObject
			co      *sqlchainObject
			loaded  bool
			addr1   = proto.AccountAddress{0x0, 0x0, 0x0, 0x1}
			addr2   = proto.AccountAddress{0x0, 0x0, 0x0, 0x2}
			dbid1   = proto.DatabaseID("db#1")
			dbid2   = proto.DatabaseID("db#2")
			mi      = newMetaIndex()
			fl      = path.Join(testDataDir, t.Name())
			db, err = bolt.Open(fl, 0600, nil)
		)
		So(err, ShouldBeNil)
		Reset(func() {
			// Clean database file after each pass
			err = db.Close()
			So(err, ShouldBeNil)
			err = os.Truncate(fl, 0)
			So(err, ShouldBeNil)
		})
		err = db.Update(func(tx *bolt.Tx) (err error) {
			var meta, txbk *bolt.Bucket
			if meta, err = tx.CreateBucket(metaBucket[:]); err != nil {
				return
			}
			if _, err = meta.CreateBucket(metaAccountIndexBucket); err != nil {
				return
			}
			if _, err = meta.CreateBucket(metaSQLChainIndexBucket); err != nil {
				return
			}
			if txbk, err = meta.CreateBucket(metaTransactionBucket); err != nil {
				return
			} else {
				for i := pi.TransactionType(0); i < pi.TransactionTypeNumber; i++ {
					if _, err = txbk.CreateBucket(i.Bytes()); err != nil {
						return
					}
				}
			}
			return
		})
		So(err, ShouldBeNil)
		Convey("The metaIndex should be empty", func() {
			err = db.Update(mi.IncreaseAccountStableBalance(addr1, 1))
			So(err, ShouldEqual, ErrAccountNotFound)
			err = db.Update(mi.DecreaseAccountStableBalance(addr1, 1))
			So(err, ShouldEqual, ErrAccountNotFound)
			err = db.Update(mi.IncreaseAccountCovenantBalance(addr1, 1))
			So(err, ShouldEqual, ErrAccountNotFound)
			err = db.Update(mi.DecreaseAccountCovenantBalance(addr1, 1))
			So(err, ShouldEqual, ErrAccountNotFound)
			err = db.Update(mi.CreateSQLChain(addr1, dbid1))
			So(err, ShouldEqual, ErrAccountNotFound)
		})
		Convey("When database objects are stored", func() {
			mi.storeSQLChainObject(&sqlchainObject{
				SQLChainProfile: pt.SQLChainProfile{
					ID: dbid1,
				},
			})
			mi.storeSQLChainObject(&sqlchainObject{
				SQLChainProfile: pt.SQLChainProfile{
					ID: dbid2,
				},
			})
			Convey("The database objects should be retrievable", func() {
				co, loaded = mi.databases[dbid1]
				So(loaded, ShouldBeTrue)
				So(co, ShouldNotBeNil)
				So(co.ID, ShouldEqual, dbid1)
				co, loaded = mi.databases[dbid2]
				So(loaded, ShouldBeTrue)
				So(co, ShouldNotBeNil)
				So(co.ID, ShouldEqual, dbid2)
			})
			Convey("When database objects are deleted", func() {
				mi.deleteSQLChainObject(dbid1)
				mi.deleteSQLChainObject(dbid2)
				Convey("The acount objects should not be retrievable anymore", func() {
					co, loaded = mi.databases[dbid1]
					So(loaded, ShouldBeFalse)
					So(co, ShouldBeNil)
					co, loaded = mi.databases[dbid2]
					So(loaded, ShouldBeFalse)
					So(co, ShouldBeNil)
				})
			})
		})
		Convey("When account objects are stored", func() {
			mi.storeAccountObject(&accountObject{
				Account: pt.Account{
					Address:             addr1,
					StableCoinBalance:   10,
					CovenantCoinBalance: 10,
				},
			})
			mi.storeAccountObject(&accountObject{
				Account: pt.Account{
					Address:             addr2,
					StableCoinBalance:   10,
					CovenantCoinBalance: 10,
				},
			})
			Convey("The account objects should be retrievable", func() {
				ao, loaded = mi.accounts[addr1]
				So(loaded, ShouldBeTrue)
				So(ao, ShouldNotBeNil)
				So(ao.Address, ShouldEqual, addr1)
				ao, loaded = mi.accounts[addr2]
				So(loaded, ShouldBeTrue)
				So(ao, ShouldNotBeNil)
				So(ao.Address, ShouldEqual, addr2)
			})
			Convey("When account objects are deleted", func() {
				mi.deleteAccountObject(addr1)
				mi.deleteAccountObject(addr2)
				Convey("The acount objects should not be retrievable anymore", func() {
					ao, loaded = mi.accounts[addr1]
					So(loaded, ShouldBeFalse)
					So(ao, ShouldBeNil)
					ao, loaded = mi.accounts[addr2]
					So(loaded, ShouldBeFalse)
					So(ao, ShouldBeNil)
				})
			})
			Convey("The account objects should keep track of the their own balances", func() {
				err = db.Update(mi.IncreaseAccountStableBalance(addr1, 1))
				So(err, ShouldBeNil)
				err = db.Update(mi.DecreaseAccountStableBalance(addr1, 1))
				So(err, ShouldBeNil)
				err = db.Update(mi.IncreaseAccountCovenantBalance(addr1, 1))
				So(err, ShouldBeNil)
				err = db.Update(mi.DecreaseAccountCovenantBalance(addr1, 1))
				So(err, ShouldBeNil)
			})
			Convey("The metaIndex should be ok to add SQLChain objects", func() {
				err = db.Update(mi.CreateSQLChain(addr1, dbid1))
				So(err, ShouldBeNil)
			})
		})
	})
}
