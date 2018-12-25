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
	"sync"
	"testing"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMetaState(t *testing.T) {
	Convey("Given a new metaState object and a persistence db instance", t, func() {
		var (
			ao       *types.Account
			co       *types.SQLChainProfile
			po       *types.ProviderProfile
			bl       uint64
			loaded   bool
			err      error
			privKey1 *asymmetric.PrivateKey
			privKey2 *asymmetric.PrivateKey
			privKey3 *asymmetric.PrivateKey
			privKey4 *asymmetric.PrivateKey
			addr1    proto.AccountAddress
			addr2    proto.AccountAddress
			addr3    proto.AccountAddress
			addr4    proto.AccountAddress
			dbid1    = proto.DatabaseID("db#1")
			dbid2    = proto.DatabaseID("db#2")
			dbid3    = proto.DatabaseID("db#3")
			ms       = newMetaState()
		)
		So(err, ShouldBeNil)

		// Create key pairs and addresses for test
		privKey1, _, err = asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		privKey2, _, err = asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		privKey3, _, err = asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		privKey4, _, err = asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		addr1, err = crypto.PubKeyHash(privKey1.PubKey())
		So(err, ShouldBeNil)
		addr2, err = crypto.PubKeyHash(privKey2.PubKey())
		So(err, ShouldBeNil)
		addr3, err = crypto.PubKeyHash(privKey3.PubKey())
		So(err, ShouldBeNil)
		addr4, err = crypto.PubKeyHash(privKey4.PubKey())
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
		Convey("The provider state should be empty", func() {
			po, loaded = ms.loadProviderObject(addr1)
			So(po, ShouldBeNil)
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
			err = ms.addSQLChainUser(dbid1, addr1, types.Admin)
			So(err, ShouldEqual, ErrDatabaseNotFound)
			err = ms.deleteSQLChainUser(dbid1, addr1)
			So(err, ShouldEqual, ErrDatabaseNotFound)
			err = ms.alterSQLChainUser(dbid1, addr1, types.Write)
			So(err, ShouldEqual, ErrDatabaseNotFound)
		})
		Convey("When new account and database objects are stored", func() {
			ao, loaded = ms.loadOrStoreAccountObject(addr1, &types.Account{Address: addr1})
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			ao, loaded = ms.loadOrStoreAccountObject(addr2, &types.Account{Address: addr2})
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			co, loaded = ms.loadOrStoreSQLChainObject(dbid1, &types.SQLChainProfile{
				ID: dbid1,
				Miners: []*types.MinerInfo{
					&types.MinerInfo{Address: addr1},
					&types.MinerInfo{Address: addr2},
				},
			})
			So(co, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			co, loaded = ms.loadOrStoreSQLChainObject(dbid2, &types.SQLChainProfile{
				ID: dbid2,
				Miners: []*types.MinerInfo{
					&types.MinerInfo{Address: addr2},
					&types.MinerInfo{Address: addr3},
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
					err = ms.addSQLChainUser(dbid3, addr2, types.Write)
					So(err, ShouldBeNil)
					err = ms.addSQLChainUser(dbid3, addr2, types.Write)
					So(err, ShouldEqual, ErrDatabaseUserExists)
					Convey("The metaState object should be ok to delete user", func() {
						err = ms.deleteSQLChainUser(dbid3, addr2)
						So(err, ShouldBeNil)
						err = ms.deleteSQLChainUser(dbid3, addr2)
						So(err, ShouldBeNil)
					})
					Convey("The metaState object should be ok to alter user", func() {
						err = ms.alterSQLChainUser(dbid3, addr2, types.Read)
						So(err, ShouldBeNil)
						err = ms.alterSQLChainUser(dbid3, addr2, types.Write)
						So(err, ShouldBeNil)
					})
					Convey("When metaState change is committed", func() {
						ms.commit()
						Convey("The metaState object should return correct db list", func() {
							var dbs []*types.SQLChainProfile
							dbs = ms.loadROSQLChains(addr1)
							So(len(dbs), ShouldEqual, 1)
							dbs = ms.loadROSQLChains(addr2)
							So(len(dbs), ShouldEqual, 2)
							dbs = ms.loadROSQLChains(addr4)
							So(dbs, ShouldBeEmpty)
						})
						Convey("The metaState object should be ok to delete user", func() {
							err = ms.deleteSQLChainUser(dbid3, addr2)
							So(err, ShouldBeNil)
							err = ms.deleteSQLChainUser(dbid3, addr2)
							So(err, ShouldBeNil)
						})
						Convey("The metaState object should be ok to alter user", func() {
							err = ms.alterSQLChainUser(dbid3, addr2, types.Read)
							So(err, ShouldBeNil)
							err = ms.alterSQLChainUser(dbid3, addr2, types.Write)
							So(err, ShouldBeNil)
						})
					})
				})
				Convey("When metaState change is committed", func() {
					ms.commit()
					Convey("The metaState object should be ok to add users for database", func() {
						err = ms.addSQLChainUser(dbid3, addr2, types.Write)
						So(err, ShouldBeNil)
						err = ms.addSQLChainUser(dbid3, addr2, types.Write)
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
					So(ao.TokenBalance[types.Particle], ShouldEqual, incSta)
					So(ao.TokenBalance[types.Wave], ShouldEqual, incCov)
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
							So(ao.TokenBalance[types.Particle], ShouldEqual, incSta-decSta)
							So(ao.TokenBalance[types.Wave], ShouldEqual, incCov-decCov)
						},
					)
				})
				Convey("When metaState changes are committed", func() {
					ms.commit()
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
							So(errors.Cause(err), ShouldEqual, ErrAccountNotFound)
							err = ms.increaseAccountStableBalance(addr1, 1)
							So(err, ShouldBeNil)
						},
					)
					Convey(
						"The metaState should copy object when stable balance decreased",
						func() {
							err = ms.decreaseAccountStableBalance(addr3, 1)
							So(errors.Cause(err), ShouldEqual, ErrAccountNotFound)
							err = ms.decreaseAccountStableBalance(addr1, 1)
							So(err, ShouldBeNil)
						},
					)
					Convey(
						"The metaState should copy object when covenant balance increased",
						func() {
							err = ms.increaseAccountCovenantBalance(addr3, 1)
							So(errors.Cause(err), ShouldEqual, ErrAccountNotFound)
							err = ms.increaseAccountCovenantBalance(addr1, 1)
							So(err, ShouldBeNil)
						},
					)
					Convey(
						"The metaState should copy object when covenant balance decreased",
						func() {
							err = ms.decreaseAccountCovenantBalance(addr3, 1)
							So(errors.Cause(err), ShouldEqual, ErrAccountNotFound)
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

							ms.commit()
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
			Convey("When metaState changes are committed", func() {
				ms.commit()
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
				Convey("When some objects are deleted", func() {
					ms.deleteAccountObject(addr1)
					ms.deleteSQLChainObject(dbid1)
					Convey("The dirty map should return deleted states of these objects", func() {
						_, loaded = ms.loadAccountObject(addr1)
						So(loaded, ShouldBeFalse)
						_, loaded = ms.loadSQLChainObject(dbid1)
						So(loaded, ShouldBeFalse)
					})
				})
			})
			Convey("When transactions are added", func() {
				var (
					n  pi.AccountNonce
					t0 = types.NewBaseAccount(&types.Account{
						Address: addr1,
					})
					t1 = types.NewTransfer(
						&types.TransferHeader{
							Sender:   addr1,
							Receiver: addr2,
							Nonce:    1,
							Amount:   0,
						},
					)
					t2 = types.NewBilling(
						&types.BillingHeader{
							Nonce:     2,
							Producer:  addr1,
							Receivers: []*proto.AccountAddress{&addr2},
							Fees:      []uint64{1},
							Rewards:   []uint64{1},
						},
					)
				)
				err = t1.Sign(privKey1)
				So(err, ShouldBeNil)
				err = t2.Sign(privKey1)
				So(err, ShouldBeNil)
				err = ms.apply(t0)
				So(err, ShouldBeNil)
				ms.commit()
				err = ms.apply(t1)
				So(err, ShouldBeNil)
				ms.commit()
				err = ms.apply(t2)
				So(err, ShouldBeNil)

				Convey("The metaState should report error if tx fails verification", func() {
					t1.Nonce = pi.AccountNonce(10)
					err = t1.Sign(privKey1)
					So(err, ShouldBeNil)
					err = ms.apply(t1)
					So(err, ShouldEqual, ErrInvalidAccountNonce)
					t1.Nonce, err = ms.nextNonce(addr1)
					So(err, ShouldBeNil)
					So(t1.Nonce, ShouldEqual, ms.dirty.accounts[addr1].NextNonce)
					ms.commit()
				})
				Convey("The metaState should automatically increase nonce", func() {
					n, err = ms.nextNonce(addr1)
					So(err, ShouldBeNil)
					So(n, ShouldEqual, 3)
				})
				Convey("The metaState should report error on unknown transaction type", func() {
					err = ms.applyTransaction(nil)
					So(err, ShouldEqual, ErrUnknownTransactionType)
				})
			})
		})
		Convey("When base account txs are added", func() {
			var (
				txs = []pi.Transaction{
					types.NewBaseAccount(
						&types.Account{
							Address:      addr1,
							TokenBalance: [types.SupportTokenNumber]uint64{100, 100},
						},
					),
					types.NewBaseAccount(
						&types.Account{
							Address:      addr2,
							TokenBalance: [types.SupportTokenNumber]uint64{100, 100},
						},
					),
					types.NewTransfer(
						&types.TransferHeader{
							Sender:   addr1,
							Receiver: addr2,
							Nonce:    1,
							Amount:   10,
						},
					),
					types.NewBilling(
						&types.BillingHeader{
							Nonce:     2,
							Producer:  addr1,
							Receivers: []*proto.AccountAddress{&addr2},
							Fees:      []uint64{1},
							Rewards:   []uint64{1},
						},
					),
					types.NewBilling(
						&types.BillingHeader{
							Nonce:     1,
							Producer:  addr2,
							Receivers: []*proto.AccountAddress{&addr1},
							Fees:      []uint64{1},
							Rewards:   []uint64{1},
						},
					),
					types.NewTransfer(
						&types.TransferHeader{
							Sender:   addr2,
							Receiver: addr1,
							Nonce:    2,
							Amount:   1,
						},
					),
					types.NewTransfer(
						&types.TransferHeader{
							Sender:   addr1,
							Receiver: addr2,
							Nonce:    3,
							Amount:   10,
						},
					),
					types.NewTransfer(
						&types.TransferHeader{
							Sender:   addr2,
							Receiver: addr1,
							Nonce:    3,
							Amount:   1,
						},
					),
					types.NewTransfer(
						&types.TransferHeader{
							Sender:   addr2,
							Receiver: addr1,
							Nonce:    4,
							Amount:   1,
						},
					),
				}
			)
			txs[0].Sign(privKey1)
			txs[1].Sign(privKey2)
			txs[2].Sign(privKey1)
			txs[3].Sign(privKey1)
			txs[4].Sign(privKey2)
			txs[5].Sign(privKey2)
			txs[6].Sign(privKey1)
			txs[7].Sign(privKey2)
			txs[8].Sign(privKey2)
			for _, tx := range txs {
				err = ms.apply(tx)
				So(err, ShouldBeNil)
			}
			ms.commit()
			Convey("The state should match the update result", func() {
				bl, loaded = ms.loadAccountStableBalance(addr1)
				So(loaded, ShouldBeTrue)
				So(bl, ShouldEqual, 84)
				bl, loaded = ms.loadAccountStableBalance(addr2)
				So(loaded, ShouldBeTrue)
				So(bl, ShouldEqual, 118)
			})
		})
		Convey("When SQLChain are created", func() {
			conf.GConf, err = conf.LoadConfig("../test/node_standalone/config.yaml")
			So(err, ShouldBeNil)

			privKeyFile := "../test/node_standalone/private.key"
			pubKeyFile := "../test/node_standalone/public.keystore"
			os.Remove(pubKeyFile)
			defer os.Remove(pubKeyFile)
			route.Once = sync.Once{}
			route.InitKMS(pubKeyFile)
			err = kms.InitLocalKeyPair(privKeyFile, []byte(""))
			So(err, ShouldBeNil)

			ao, loaded = ms.loadOrStoreAccountObject(addr1, &types.Account{Address: addr1})
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			ao, loaded = ms.loadOrStoreAccountObject(addr2, &types.Account{Address: addr2})
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			ao, loaded = ms.loadOrStoreAccountObject(addr3, &types.Account{Address: addr3})
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			ao, loaded = ms.loadOrStoreAccountObject(addr4, &types.Account{Address: addr4})
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)

			// increase account balance
			var (
				txs = []pi.Transaction{
					types.NewBaseAccount(
						&types.Account{
							Address:      addr1,
							TokenBalance: [types.SupportTokenNumber]uint64{10000000, 100},
						},
					),
					types.NewBaseAccount(
						&types.Account{
							Address:      addr2,
							TokenBalance: [types.SupportTokenNumber]uint64{10000000, 100},
						},
					),
					types.NewBaseAccount(
						&types.Account{
							Address:      addr3,
							TokenBalance: [types.SupportTokenNumber]uint64{100000, 100},
						},
					),
				}
			)

			err = txs[0].Sign(privKey1)
			So(err, ShouldBeNil)
			err = txs[1].Sign(privKey2)
			So(err, ShouldBeNil)
			err = txs[2].Sign(privKey3)
			for i := range txs {
				err = ms.apply(txs[i])
				So(err, ShouldBeNil)
				ms.commit()
			}

			Convey("When provider transaction is invalid", func() {
				invalidPs := types.ProvideService{
					ProvideServiceHeader: types.ProvideServiceHeader{
						TargetUser: addr1,
						Nonce:      1,
					},
				}
				err = invalidPs.Sign(privKey3)
				So(err, ShouldBeNil)
				invalidCd1 := types.CreateDatabase{
					CreateDatabaseHeader: types.CreateDatabaseHeader{
						Owner:     addr2,
						GasPrice:  1,
						TokenType: types.Particle,
						Nonce:     1,
					},
				}
				err = invalidCd1.Sign(privKey1)
				So(err, ShouldBeNil)
				invalidCd2 := types.CreateDatabase{
					CreateDatabaseHeader: types.CreateDatabaseHeader{
						Owner: addr1,
						ResourceMeta: types.ResourceMeta{
							TargetMiners: []proto.AccountAddress{addr2},
						},
						GasPrice:       1,
						AdvancePayment: uint64(conf.GConf.QPS) * uint64(conf.GConf.UpdatePeriod) * 1,
						TokenType:      types.Particle,
						Nonce:          1,
					},
				}
				err = invalidCd2.Sign(privKey1)
				So(err, ShouldBeNil)
				invalidCd3 := types.CreateDatabase{
					CreateDatabaseHeader: types.CreateDatabaseHeader{
						Owner: addr3,
						ResourceMeta: types.ResourceMeta{
							TargetMiners: []proto.AccountAddress{addr2},
						},
						GasPrice:  1,
						TokenType: types.Particle,
						Nonce:     1,
					},
				}
				err = invalidCd3.Sign(privKey3)
				So(err, ShouldBeNil)
				invalidCd4 := types.CreateDatabase{
					CreateDatabaseHeader: types.CreateDatabaseHeader{
						Owner: addr3,
						ResourceMeta: types.ResourceMeta{
							TargetMiners: []proto.AccountAddress{addr2},
						},
						Nonce: 1,
					},
				}
				err = invalidCd4.Sign(privKey3)
				So(err, ShouldBeNil)

				err = ms.apply(&invalidPs)
				So(errors.Cause(err), ShouldEqual, ErrInsufficientBalance)
				err = ms.apply(&invalidCd1)
				So(errors.Cause(err), ShouldEqual, ErrInvalidSender)
				err = ms.apply(&invalidCd2)
				So(errors.Cause(err), ShouldEqual, ErrNoSuchMiner)
				err = ms.apply(&invalidCd3)
				So(errors.Cause(err), ShouldEqual, ErrInsufficientAdvancePayment)
				err = ms.apply(&invalidCd4)
				So(errors.Cause(err), ShouldEqual, ErrInvalidGasPrice)
			})
			Convey("When SQLChain create", func() {
				ps := types.ProvideService{
					ProvideServiceHeader: types.ProvideServiceHeader{
						TargetUser: addr1,
						GasPrice:   1,
						TokenType:  types.Particle,
						Nonce:      1,
					},
				}
				err = ps.Sign(privKey2)
				So(err, ShouldBeNil)
				cd1 := types.CreateDatabase{
					CreateDatabaseHeader: types.CreateDatabaseHeader{
						Owner: addr1,
						ResourceMeta: types.ResourceMeta{
							TargetMiners: []proto.AccountAddress{addr2},
						},
						GasPrice:       1,
						AdvancePayment: 3600000,
						TokenType:      types.Particle,
						Nonce:          1,
					},
				}
				err = cd1.Sign(privKey1)
				So(err, ShouldBeNil)
				cd2 := types.CreateDatabase{
					CreateDatabaseHeader: types.CreateDatabaseHeader{
						Owner: addr3,
						ResourceMeta: types.ResourceMeta{
							TargetMiners: []proto.AccountAddress{addr2},
						},
						GasPrice:       1,
						AdvancePayment: 3600000,
						TokenType:      types.Particle,
						Nonce:          1,
					},
				}
				err = cd2.Sign(privKey3)
				So(err, ShouldBeNil)

				var b1, b2 uint64
				b1, loaded = ms.loadAccountStableBalance(addr2)
				err = ms.apply(&ps)
				So(err, ShouldBeNil)
				ms.commit()
				b2, loaded = ms.loadAccountStableBalance(addr2)
				So(loaded, ShouldBeTrue)
				So(b1-b2, ShouldEqual, conf.GConf.MinProviderDeposit)
				err = ms.apply(&cd2)
				So(errors.Cause(err), ShouldEqual, ErrMinerUserNotMatch)
				b1, loaded = ms.loadAccountStableBalance(addr1)
				So(loaded, ShouldBeTrue)
				err = ms.apply(&cd1)
				So(err, ShouldBeNil)
				ms.commit()
				b2, loaded = ms.loadAccountStableBalance(addr1)
				So(loaded, ShouldBeTrue)
				minAdvancePayment := uint64(cd2.GasPrice) * uint64(conf.GConf.QPS) *
					uint64(conf.GConf.UpdatePeriod) * uint64(len(cd2.ResourceMeta.TargetMiners))
				So(b1-b2, ShouldEqual, cd1.AdvancePayment+minAdvancePayment)
				dbID := proto.FromAccountAndNonce(cd1.Owner, uint32(cd1.Nonce))
				co, loaded = ms.loadSQLChainObject(*dbID)
				So(loaded, ShouldBeTrue)
				dbAccount, err := dbID.AccountAddress()
				So(err, ShouldBeNil)

				up := types.UpdatePermission{
					UpdatePermissionHeader: types.UpdatePermissionHeader{
						TargetSQLChain: addr1,
						TargetUser:     addr3,
						Permission:     types.Read,
						Nonce:          cd1.Nonce + 1,
					},
				}
				err = up.Sign(privKey1)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(errors.Cause(err), ShouldEqual, ErrDatabaseNotFound)
				up.Permission = 4
				up.TargetSQLChain = dbAccount
				err = up.Sign(privKey1)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(errors.Cause(err), ShouldEqual, ErrInvalidPermission)
				// test permission update
				// addr1(admin) update addr3 as admin
				up.TargetUser = addr3
				up.Permission = types.Admin
				err = up.Sign(privKey1)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(err, ShouldBeNil)
				ms.commit()
				// addr3(admin) update addr4 as read
				up.TargetUser = addr4
				up.Nonce = cd2.Nonce
				up.Permission = types.Read
				err = up.Sign(privKey3)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(err, ShouldBeNil)
				ms.commit()
				// addr3(admin) update addr1(admin) as read
				up.TargetUser = addr1
				up.Nonce = up.Nonce + 1
				err = up.Sign(privKey3)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(err, ShouldBeNil)
				ms.commit()
				// addr3(admin) update addr3(admin) as read fail
				up.TargetUser = addr3
				up.Permission = types.Read
				up.Nonce = up.Nonce + 1
				err = up.Sign(privKey3)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(errors.Cause(err), ShouldEqual, ErrInvalidSender)
				// addr1(read) update addr3(admin) fail
				up.Nonce = cd1.Nonce + 2
				err = up.Sign(privKey1)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(errors.Cause(err), ShouldEqual, ErrAccountPermissionDeny)

				co, loaded = ms.loadSQLChainObject(*dbID)
				for _, user := range co.Users {
					if user.Address == addr1 {
						So(user.Permission, ShouldEqual, types.Read)
						continue
					}
					if user.Address == addr3 {
						So(user.Permission, ShouldEqual, types.Admin)
						continue
					}
					if user.Address == addr4 {
						So(user.Permission, ShouldEqual, types.Read)
						continue
					}
				}
				Convey("Update key", func() {
					invalidIk1 := &types.IssueKeys{}
					err = invalidIk1.Sign(privKey1)
					So(err, ShouldBeNil)
					err = ms.apply(invalidIk1)
					So(err, ShouldEqual, ErrInvalidAccountNonce)
					invalidIk2 := &types.IssueKeys{
						IssueKeysHeader: types.IssueKeysHeader{
							TargetSQLChain: addr1,
							Nonce:          3,
						},
					}
					err = invalidIk2.Sign(privKey3)
					So(err, ShouldBeNil)
					err = ms.apply(invalidIk2)
					So(err, ShouldEqual, ErrDatabaseNotFound)
					invalidIk3 := &types.IssueKeys{
						IssueKeysHeader: types.IssueKeysHeader{
							TargetSQLChain: dbAccount,
							Nonce:          3,
						},
					}
					err = invalidIk3.Sign(privKey1)
					So(err, ShouldBeNil)
					err = ms.apply(invalidIk3)
					So(err, ShouldEqual, ErrAccountPermissionDeny)
					ik1 := &types.IssueKeys{
						IssueKeysHeader: types.IssueKeysHeader{
							TargetSQLChain: dbAccount,
							Nonce:          3,
						},
					}
					err = ik1.Sign(privKey3)
					So(err, ShouldBeNil)
					err = ms.apply(ik1)
					So(err, ShouldBeNil)
					ms.commit()
					encryptKey := "12345"
					ik2 := &types.IssueKeys{
						IssueKeysHeader: types.IssueKeysHeader{
							MinerKeys: []types.MinerKey{
								{
									Miner:         addr1,
									EncryptionKey: encryptKey,
								},
							},
							TargetSQLChain: dbAccount,
							Nonce:          4,
						},
					}
					err = ik2.Sign(privKey3)
					So(err, ShouldBeNil)
					err = ms.apply(ik2)
					So(err, ShouldBeNil)
					ms.commit()

					co, loaded = ms.loadSQLChainObject(*dbID)
					for _, miner := range co.Miners {
						if miner.Address == addr1 {
							So(miner.EncryptionKey, ShouldEqual, encryptKey)
						}
					}
				})
			})
		})
	})
}
