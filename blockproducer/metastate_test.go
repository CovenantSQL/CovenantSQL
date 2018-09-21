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
	"os"
	"path"
	"testing"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	pt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/coreos/bbolt"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMetaState(t *testing.T) {
	Convey("Given a new metaState object and a persistence db instance", t, func() {
		var (
			ao      *accountObject
			co      *sqlchainObject
			bl      uint64
			loaded  bool
			addr1   = proto.AccountAddress{0x0, 0x0, 0x0, 0x1}
			addr2   = proto.AccountAddress{0x0, 0x0, 0x0, 0x2}
			addr3   = proto.AccountAddress{0x0, 0x0, 0x0, 0x3}
			dbid1   = proto.DatabaseID("db#1")
			dbid2   = proto.DatabaseID("db#2")
			dbid3   = proto.DatabaseID("db#3")
			ms      = newMetaState()
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
			}
			for i := pi.TransactionType(0); i < pi.TransactionTypeNumber; i++ {
				if _, err = txbk.CreateBucket(i.Bytes()); err != nil {
					return
				}
			}
			return
		})
		So(err, ShouldBeNil)
		Convey("The account state should be empty", func() {
			ao, loaded = ms.loadAccountObject(addr1)
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			bl, loaded = ms.loadAccountStableBalance(addr1)
			So(loaded, ShouldBeFalse)
			bl, loaded = ms.loadAccountCovenantBalance(addr1)
			So(loaded, ShouldBeFalse)
		})
		Convey("The database state should be empty", func() {
			co, loaded = ms.loadSQLChainObject(dbid1)
			So(co, ShouldBeNil)
			So(loaded, ShouldBeFalse)
		})
		Convey("The nonce state should be empty", func() {
			_, err = ms.nextNonce(addr1)
			So(err, ShouldEqual, ErrAccountNotFound)
			err = ms.increaseNonce(addr1)
			So(err, ShouldEqual, ErrAccountNotFound)
		})
		Convey("The metaState should failed to operate SQLChain for unknown user", func() {
			err = ms.createSQLChain(addr1, dbid1)
			So(err, ShouldEqual, ErrAccountNotFound)
			err = ms.addSQLChainUser(dbid1, addr1, pt.Admin)
			So(err, ShouldEqual, ErrDatabaseNotFound)
			err = ms.deleteSQLChainUser(dbid1, addr1)
			So(err, ShouldEqual, ErrDatabaseNotFound)
			err = ms.alterSQLChainUser(dbid1, addr1, pt.ReadWrite)
			So(err, ShouldEqual, ErrDatabaseNotFound)
		})
		Convey("When new account and database objects are stored", func() {
			ao, loaded = ms.loadOrStoreAccountObject(addr1, &accountObject{
				Account: pt.Account{
					Address: addr1,
				},
			})
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			ao, loaded = ms.loadOrStoreAccountObject(addr2, &accountObject{
				Account: pt.Account{
					Address: addr2,
				},
			})
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			co, loaded = ms.loadOrStoreSQLChainObject(dbid1, &sqlchainObject{
				SQLChainProfile: pt.SQLChainProfile{
					ID: dbid1,
				},
			})
			So(co, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			co, loaded = ms.loadOrStoreSQLChainObject(dbid2, &sqlchainObject{
				SQLChainProfile: pt.SQLChainProfile{
					ID: dbid2,
				},
			})
			So(co, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			Convey("The state should include the account and database objects", func() {
				ao, loaded = ms.loadAccountObject(addr1)
				So(loaded, ShouldBeTrue)
				So(ao, ShouldNotBeNil)
				So(ao.Address, ShouldEqual, addr1)
				ao, loaded = ms.loadOrStoreAccountObject(addr1, nil)
				So(loaded, ShouldBeTrue)
				So(ao, ShouldNotBeNil)
				So(ao.Address, ShouldEqual, addr1)
				co, loaded = ms.loadSQLChainObject(dbid1)
				So(loaded, ShouldBeTrue)
				So(co, ShouldNotBeNil)
				So(co.ID, ShouldEqual, dbid1)
				co, loaded = ms.loadOrStoreSQLChainObject(dbid1, nil)
				So(loaded, ShouldBeTrue)
				So(co, ShouldNotBeNil)
				So(co.ID, ShouldEqual, dbid1)
				bl, loaded = ms.loadAccountStableBalance(addr1)
				So(loaded, ShouldBeTrue)
				So(bl, ShouldEqual, 0)
				bl, loaded = ms.loadAccountCovenantBalance(addr1)
				So(loaded, ShouldBeTrue)
				So(bl, ShouldEqual, 0)
			})
			Convey("When new SQLChain is created", func() {
				err = ms.createSQLChain(addr1, dbid3)
				So(err, ShouldBeNil)
				Convey("The metaState object should report database exists", func() {
					err = ms.createSQLChain(addr1, dbid3)
					So(err, ShouldEqual, ErrDatabaseExists)
				})
				Convey("When new SQLChain users are added", func() {
					err = ms.addSQLChainUser(dbid3, addr2, pt.ReadWrite)
					So(err, ShouldBeNil)
					err = ms.addSQLChainUser(dbid3, addr2, pt.ReadWrite)
					So(err, ShouldEqual, ErrDatabaseUserExists)
					Convey("The metaState object should be ok to delete user", func() {
						err = ms.deleteSQLChainUser(dbid3, addr2)
						So(err, ShouldBeNil)
						err = ms.deleteSQLChainUser(dbid3, addr2)
						So(err, ShouldBeNil)
					})
					Convey("The metaState object should be ok to alter user", func() {
						err = ms.alterSQLChainUser(dbid3, addr2, pt.Read)
						So(err, ShouldBeNil)
						err = ms.alterSQLChainUser(dbid3, addr2, pt.ReadWrite)
						So(err, ShouldBeNil)
					})
					Convey("When metaState change is committed", func() {
						err = db.Update(ms.commitProcedure())
						So(err, ShouldBeNil)
						Convey("The metaState object should be ok to delete user", func() {
							err = ms.deleteSQLChainUser(dbid3, addr2)
							So(err, ShouldBeNil)
							err = ms.deleteSQLChainUser(dbid3, addr2)
							So(err, ShouldBeNil)
						})
						Convey("The metaState object should be ok to alter user", func() {
							err = ms.alterSQLChainUser(dbid3, addr2, pt.Read)
							So(err, ShouldBeNil)
							err = ms.alterSQLChainUser(dbid3, addr2, pt.ReadWrite)
							So(err, ShouldBeNil)
						})
					})
				})
				Convey("When metaState change is committed", func() {
					err = db.Update(ms.commitProcedure())
					So(err, ShouldBeNil)
					Convey("The metaState object should be ok to add users for database", func() {
						err = ms.addSQLChainUser(dbid3, addr2, pt.ReadWrite)
						So(err, ShouldBeNil)
						err = ms.addSQLChainUser(dbid3, addr2, pt.ReadWrite)
						So(err, ShouldEqual, ErrDatabaseUserExists)
					})
					Convey("The metaState object should report database exists", func() {
						err = ms.createSQLChain(addr1, dbid3)
						So(err, ShouldEqual, ErrDatabaseExists)
					})
				})
			})
			Convey("When all the above modification are reset", func() {
				ms.clean()
				Convey("The account state should be empty", func() {
					ao, loaded = ms.loadAccountObject(addr1)
					So(ao, ShouldBeNil)
					So(loaded, ShouldBeFalse)
				})
				Convey("The database state should be empty", func() {
					co, loaded = ms.loadSQLChainObject(dbid1)
					So(co, ShouldBeNil)
					So(loaded, ShouldBeFalse)
				})
			})
			Convey("When the account balance is increased", func() {
				var (
					incSta uint64 = 100
					decSta uint64 = 10
					incCov uint64 = 1000
					decCov uint64 = 100
				)
				err = ms.increaseAccountStableBalance(addr1, incSta)
				So(err, ShouldBeNil)
				err = ms.increaseAccountCovenantBalance(addr1, incCov)
				So(err, ShouldBeNil)
				Convey("The state should report error when the account balance is increased"+
					" by an impossible amount",
					func() {
						err = ms.increaseAccountStableBalance(addr1, math.MaxUint64)
						So(err, ShouldEqual, ErrBalanceOverflow)
						err = ms.increaseAccountCovenantBalance(addr1, math.MaxUint64)
						So(err, ShouldEqual, ErrBalanceOverflow)
					},
				)
				Convey("The state should report error when the account balance is decreased"+
					" by an impossible amount",
					func() {
						err = ms.decreaseAccountStableBalance(addr1, incSta+1)
						So(err, ShouldEqual, ErrInsufficientBalance)
						err = ms.decreaseAccountCovenantBalance(addr1, incCov+1)
						So(err, ShouldEqual, ErrInsufficientBalance)
					},
				)
				Convey("The account balance should be kept correctly in account object", func() {
					ao, loaded = ms.loadAccountObject(addr1)
					So(loaded, ShouldBeTrue)
					So(ao, ShouldNotBeNil)
					So(ao.Address, ShouldEqual, addr1)
					So(ao.StableCoinBalance, ShouldEqual, incSta)
					So(ao.CovenantCoinBalance, ShouldEqual, incCov)
					bl, loaded = ms.loadAccountStableBalance(addr1)
					So(loaded, ShouldBeTrue)
					So(bl, ShouldEqual, incSta)
					bl, loaded = ms.loadAccountCovenantBalance(addr1)
					So(loaded, ShouldBeTrue)
					So(bl, ShouldEqual, incCov)
				})
				Convey("When the account balance is decreased", func() {
					err = ms.decreaseAccountStableBalance(addr1, decSta)
					So(err, ShouldBeNil)
					err = ms.decreaseAccountCovenantBalance(addr1, decCov)
					So(err, ShouldBeNil)
					Convey(
						"The account balance should still be kept correctly in account object",
						func() {
							ao, loaded = ms.loadAccountObject(addr1)
							So(loaded, ShouldBeTrue)
							So(ao, ShouldNotBeNil)
							So(ao.Address, ShouldEqual, addr1)
							So(ao.StableCoinBalance, ShouldEqual, incSta-decSta)
							So(ao.CovenantCoinBalance, ShouldEqual, incCov-decCov)
						},
					)
				})
				Convey("When metaState changes are committed", func() {
					err = db.Update(ms.commitProcedure())
					So(err, ShouldBeNil)
					Convey(
						"The account balance should be kept correctly in account object",
						func() {
							bl, loaded = ms.loadAccountStableBalance(addr1)
							So(loaded, ShouldBeTrue)
							So(bl, ShouldEqual, incSta)
							bl, loaded = ms.loadAccountCovenantBalance(addr1)
							So(loaded, ShouldBeTrue)
							So(bl, ShouldEqual, incCov)
						},
					)
					Convey(
						"The metaState should copy object when stable balance increased",
						func() {
							err = ms.increaseAccountStableBalance(addr3, 1)
							So(err, ShouldEqual, ErrAccountNotFound)
							err = ms.increaseAccountStableBalance(addr1, 1)
							So(err, ShouldBeNil)
						},
					)
					Convey(
						"The metaState should copy object when stable balance decreased",
						func() {
							err = ms.decreaseAccountStableBalance(addr3, 1)
							So(err, ShouldEqual, ErrAccountNotFound)
							err = ms.decreaseAccountStableBalance(addr1, 1)
							So(err, ShouldBeNil)
						},
					)
					Convey(
						"The metaState should copy object when covenant balance increased",
						func() {
							err = ms.increaseAccountCovenantBalance(addr3, 1)
							So(err, ShouldEqual, ErrAccountNotFound)
							err = ms.increaseAccountCovenantBalance(addr1, 1)
							So(err, ShouldBeNil)
						},
					)
					Convey(
						"The metaState should copy object when covenant balance decreased",
						func() {
							err = ms.decreaseAccountCovenantBalance(addr3, 1)
							So(err, ShouldEqual, ErrAccountNotFound)
							err = ms.decreaseAccountCovenantBalance(addr1, 1)
							So(err, ShouldBeNil)
						},
					)
					Convey(
						"The metaState should copy object when stable balance transferred",
						func() {
							err = ms.transferAccountStableBalance(addr1, addr3, incSta+1)
							So(err, ShouldEqual, ErrInsufficientBalance)
							err = ms.transferAccountStableBalance(addr1, addr3, 1)
							So(err, ShouldBeNil)
							err = ms.increaseAccountStableBalance(addr2, math.MaxUint64)
							So(err, ShouldBeNil)
							err = db.Update(ms.commitProcedure())
							So(err, ShouldBeNil)
							err = ms.transferAccountStableBalance(addr2, addr1, math.MaxUint64)
							So(err, ShouldEqual, ErrBalanceOverflow)
							err = ms.transferAccountStableBalance(addr2, addr3, 1)
							So(err, ShouldBeNil)
						},
					)
					Convey(
						"The metaState should copy object when nonce increased",
						func() {
							err = ms.increaseNonce(addr1)
							So(err, ShouldBeNil)
						},
					)
				})
			})
			Convey("When a new account key slot is overwritten", func() {
				err = db.Update(func(tx *bolt.Tx) (err error) {
					var bucket = tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
					if err = bucket.Delete(addr1[:]); err != nil {
						return
					}
					if _, err = bucket.CreateBucket(addr1[:]); err != nil {
						return
					}
					return
				})
				So(err, ShouldBeNil)
				Convey("The reloadProcedure should report error", func() {
					err = db.Update(ms.commitProcedure())
					So(err, ShouldNotBeNil)
				})
			})
			Convey("When a new database key slot is overwritten", func() {
				err = db.Update(func(tx *bolt.Tx) (err error) {
					var bucket = tx.Bucket(metaBucket[:]).Bucket(metaSQLChainIndexBucket)
					if err = bucket.Delete([]byte(dbid1)); err != nil {
						return
					}
					if _, err = bucket.CreateBucket([]byte(dbid1)); err != nil {
						return
					}
					return
				})
				So(err, ShouldBeNil)
				Convey("The reloadProcedure should report error", func() {
					err = db.Update(ms.commitProcedure())
					So(err, ShouldNotBeNil)
				})
			})
			Convey("When metaState changes are committed", func() {
				err = db.Update(ms.commitProcedure())
				So(err, ShouldBeNil)
				Convey("The cached object should be retrievable from readonly map", func() {
					var loaded bool
					_, loaded = ms.loadAccountObject(addr1)
					So(loaded, ShouldBeTrue)
					_, loaded = ms.loadOrStoreAccountObject(addr1, nil)
					So(loaded, ShouldBeTrue)
					_, loaded = ms.loadSQLChainObject(dbid1)
					So(loaded, ShouldBeTrue)
					_, loaded = ms.loadOrStoreSQLChainObject(dbid2, nil)
					So(loaded, ShouldBeTrue)
				})
				Convey("The metaState should be reproducible from the persistence db", func() {
					var (
						oa1, oa2, ra1, ra2 *accountObject
						oc1, oc2, rc1, rc2 *sqlchainObject
						loaded             bool
						rms                = newMetaState()
						err                = db.View(rms.reloadProcedure())
					)
					So(err, ShouldBeNil)
					oa1, loaded = ms.loadAccountObject(addr1)
					So(loaded, ShouldBeTrue)
					So(oa1, ShouldNotBeNil)
					oa2, loaded = ms.loadAccountObject(addr2)
					So(loaded, ShouldBeTrue)
					So(oa2, ShouldNotBeNil)
					ra1, loaded = rms.loadAccountObject(addr1)
					So(loaded, ShouldBeTrue)
					So(ra1, ShouldNotBeNil)
					ra2, loaded = rms.loadAccountObject(addr2)
					So(loaded, ShouldBeTrue)
					So(ra2, ShouldNotBeNil)
					So(&oa1.Account, ShouldResemble, &ra1.Account)
					So(&oa2.Account, ShouldResemble, &ra2.Account)
					oc1, loaded = ms.loadSQLChainObject(dbid1)
					So(loaded, ShouldBeTrue)
					So(oc1, ShouldNotBeNil)
					oc2, loaded = ms.loadSQLChainObject(dbid2)
					So(loaded, ShouldBeTrue)
					So(oc2, ShouldNotBeNil)
					rc1, loaded = rms.loadSQLChainObject(dbid1)
					So(loaded, ShouldBeTrue)
					So(rc1, ShouldNotBeNil)
					rc2, loaded = rms.loadSQLChainObject(dbid2)
					So(loaded, ShouldBeTrue)
					So(rc2, ShouldNotBeNil)
					So(&oc1.SQLChainProfile, ShouldResemble, &rc1.SQLChainProfile)
					So(&oc2.SQLChainProfile, ShouldResemble, &rc2.SQLChainProfile)
				})
				Convey("When the some accountObject is corrupted", func() {
					err = db.Update(func(tx *bolt.Tx) (err error) {
						return tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket).Put(
							addr1[:], []byte{0x1, 0x2, 0x3, 0x4a},
						)
					})
					So(err, ShouldBeNil)
					Convey("The reloadProcedure should report error", func() {
						var (
							rms = newMetaState()
							err = db.View(rms.reloadProcedure())
						)
						So(err, ShouldNotBeNil)
					})
				})
				Convey("When the some sqlchainObject is corrupted", func() {
					err = db.Update(func(tx *bolt.Tx) (err error) {
						return tx.Bucket(metaBucket[:]).Bucket(metaSQLChainIndexBucket).Put(
							[]byte(dbid1), []byte{0x1, 0x2, 0x3, 0x4a},
						)
					})
					So(err, ShouldBeNil)
					Convey("The reloadProcedure should report error", func() {
						var (
							rms = newMetaState()
							err = db.View(rms.reloadProcedure())
						)
						So(err, ShouldNotBeNil)
					})
				})
				Convey("When some objects are deleted", func() {
					ms.deleteAccountObject(addr1)
					ms.deleteSQLChainObject(dbid1)
					Convey("The dirty map should return deleted states of these objects", func() {
						_, loaded = ms.loadAccountObject(addr1)
						So(loaded, ShouldBeFalse)
						_, loaded = ms.loadSQLChainObject(dbid1)
						So(loaded, ShouldBeFalse)
					})
					Convey("When the deleted account key slot is overwritten", func() {
						err = db.Update(func(tx *bolt.Tx) (err error) {
							var bucket = tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
							if err = bucket.Delete(addr1[:]); err != nil {
								return
							}
							if _, err = bucket.CreateBucket(addr1[:]); err != nil {
								return
							}
							return
						})
						So(err, ShouldBeNil)
						Convey("The commitProcedure should report error", func() {
							var err = db.Update(ms.commitProcedure())
							So(err, ShouldNotBeNil)
						})
					})
					Convey("When the deleted database key slot is overwritten", func() {
						err = db.Update(func(tx *bolt.Tx) (err error) {
							var bucket = tx.Bucket(metaBucket[:]).Bucket(metaSQLChainIndexBucket)
							if err = bucket.Delete([]byte(dbid1)); err != nil {
								return
							}
							if _, err = bucket.CreateBucket([]byte(dbid1)); err != nil {
								return
							}
							return
						})
						So(err, ShouldBeNil)
						Convey("The commitProcedure should report error", func() {
							var err = db.Update(ms.commitProcedure())
							So(err, ShouldNotBeNil)
						})
					})
					Convey("When metaState changes are committed again", func() {
						err = db.Update(ms.commitProcedure())
						So(err, ShouldBeNil)
						Convey(
							"The metaState should also be reproducible from the persistence db",
							func() {
								var (
									rms                = newMetaState()
									err                = db.View(rms.reloadProcedure())
									oa1, oa2, ra1, ra2 *accountObject
									oc1, oc2, rc1, rc2 *sqlchainObject
									loaded             bool
								)
								So(err, ShouldBeNil)
								oa1, loaded = ms.loadAccountObject(addr1)
								So(loaded, ShouldBeFalse)
								So(oa1, ShouldBeNil)
								oa2, loaded = ms.loadAccountObject(addr2)
								So(loaded, ShouldBeTrue)
								So(oa2, ShouldNotBeNil)
								ra1, loaded = rms.loadAccountObject(addr1)
								So(loaded, ShouldBeFalse)
								So(ra1, ShouldBeNil)
								ra2, loaded = rms.loadAccountObject(addr2)
								So(loaded, ShouldBeTrue)
								So(ra2, ShouldNotBeNil)
								So(&oa2.Account, ShouldResemble, &ra2.Account)
								oc1, loaded = ms.loadSQLChainObject(dbid1)
								So(loaded, ShouldBeFalse)
								So(oc1, ShouldBeNil)
								oc2, loaded = ms.loadSQLChainObject(dbid2)
								So(loaded, ShouldBeTrue)
								So(oc2, ShouldNotBeNil)
								rc1, loaded = rms.loadSQLChainObject(dbid1)
								So(loaded, ShouldBeFalse)
								So(rc1, ShouldBeNil)
								rc2, loaded = rms.loadSQLChainObject(dbid2)
								So(loaded, ShouldBeTrue)
								So(rc2, ShouldNotBeNil)
								So(&oc2.SQLChainProfile, ShouldResemble, &rc2.SQLChainProfile)
							},
						)
					})
				})
			})
			Convey("When transacions are added", func() {
				var (
					n  pi.AccountNonce
					t1 = &pt.Transfer{
						TransferHeader: pt.TransferHeader{
							Sender:   addr1,
							Receiver: addr2,
							Nonce:    0,
							Amount:   0,
						},
					}
					t2 = &pt.TxBilling{
						TxContent: pt.TxContent{
							SequenceID: 1,
							Receivers:  []*proto.AccountAddress{&addr2},
							Fees:       []uint64{1},
							Rewards:    []uint64{1},
						},
						AccountAddress: &addr1,
					}
				)
				err = t1.Sign(testPrivKey)
				So(err, ShouldBeNil)
				err = t2.Sign(testPrivKey)
				So(err, ShouldBeNil)
				err = db.Update(ms.applyTransactionProcedure(t1))
				So(err, ShouldBeNil)
				err = db.Update(ms.applyTransactionProcedure(t2))
				So(err, ShouldBeNil)
				Convey("The metaState should report error if tx fails verification", func() {
					err = db.Update(ms.applyTransactionProcedure(t1))
					So(err, ShouldEqual, ErrInvalidAccountNonce)
					t1.Nonce, err = ms.nextNonce(addr1)
					So(err, ShouldBeNil)
					So(t1.Nonce, ShouldEqual, ms.dirty.accounts[addr1].NextNonce)
					err = db.Update(ms.applyTransactionProcedure(t1))
					So(err, ShouldNotBeNil)
					err = t1.Sign(testPrivKey)
					So(err, ShouldBeNil)
					err = db.Update(ms.applyTransactionProcedure(t1))
					So(err, ShouldBeNil)
				})
				Convey("The metaState should automatically increase nonce", func() {
					n, err = ms.nextNonce(addr1)
					So(err, ShouldBeNil)
					So(n, ShouldEqual, 2)
				})
				Convey("The metaState should report error on unknown transaction type", func() {
					err = ms.applyTransaction(nil)
					So(err, ShouldEqual, ErrUnknownTransactionType)
				})
			})
		})
	})
}
