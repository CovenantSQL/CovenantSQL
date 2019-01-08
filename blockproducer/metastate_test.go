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
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
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
			dbID1    = proto.DatabaseID("db#1")
			dbID2    = proto.DatabaseID("db#2")
			dbID3    = proto.DatabaseID("db#3")
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
			bl, loaded = ms.loadAccountTokenBalance(addr1, types.Particle)
			So(loaded, ShouldBeFalse)
			bl, loaded = ms.loadAccountTokenBalance(addr1, types.Wave)
			So(loaded, ShouldBeFalse)
		})
		Convey("The database state should be empty", func() {
			co, loaded = ms.loadSQLChainObject(dbID1)
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
			err = ms.createSQLChain(addr1, dbID1)
			So(err, ShouldEqual, ErrAccountNotFound)
			err = ms.addSQLChainUser(dbID1, addr1, types.UserPermissionFromRole(types.Admin))
			So(err, ShouldEqual, ErrDatabaseNotFound)
			err = ms.deleteSQLChainUser(dbID1, addr1)
			So(err, ShouldEqual, ErrDatabaseNotFound)
			err = ms.alterSQLChainUser(dbID1, addr1, types.UserPermissionFromRole(types.Write))
			So(err, ShouldEqual, ErrDatabaseNotFound)
		})
		Convey("When new account and database objects are stored", func() {
			ao, loaded = ms.loadOrStoreAccountObject(addr1, &types.Account{Address: addr1})
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			ao, loaded = ms.loadOrStoreAccountObject(addr2, &types.Account{Address: addr2})
			So(ao, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			co, loaded = ms.loadOrStoreSQLChainObject(dbID1, &types.SQLChainProfile{
				ID: dbID1,
				Miners: []*types.MinerInfo{
					&types.MinerInfo{Address: addr1},
					&types.MinerInfo{Address: addr2},
				},
			})
			So(co, ShouldBeNil)
			So(loaded, ShouldBeFalse)
			co, loaded = ms.loadOrStoreSQLChainObject(dbID2, &types.SQLChainProfile{
				ID: dbID2,
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
				co, loaded = ms.loadSQLChainObject(dbID1)
				So(loaded, ShouldBeTrue)
				So(co, ShouldNotBeNil)
				So(co.ID, ShouldEqual, dbID1)
				co, loaded = ms.loadOrStoreSQLChainObject(dbID1, nil)
				So(loaded, ShouldBeTrue)
				So(co, ShouldNotBeNil)
				So(co.ID, ShouldEqual, dbID1)
				bl, loaded = ms.loadAccountTokenBalance(addr1, types.Particle)
				So(loaded, ShouldBeTrue)
				So(bl, ShouldEqual, 0)
				bl, loaded = ms.loadAccountTokenBalance(addr1, types.Wave)
				So(loaded, ShouldBeTrue)
				So(bl, ShouldEqual, 0)
			})
			Convey("When new SQLChain is created", func() {
				err = ms.createSQLChain(addr1, dbID3)
				So(err, ShouldBeNil)
				Convey("The metaState object should report database exists", func() {
					err = ms.createSQLChain(addr1, dbID3)
					So(err, ShouldEqual, ErrDatabaseExists)
				})
				Convey("When new SQLChain users are added", func() {
					err = ms.addSQLChainUser(dbID3, addr2, types.UserPermissionFromRole(types.Write))
					So(err, ShouldBeNil)
					err = ms.addSQLChainUser(dbID3, addr2, types.UserPermissionFromRole(types.Write))
					So(err, ShouldEqual, ErrDatabaseUserExists)
					Convey("The metaState object should be ok to delete user", func() {
						err = ms.deleteSQLChainUser(dbID3, addr2)
						So(err, ShouldBeNil)
						err = ms.deleteSQLChainUser(dbID3, addr2)
						So(err, ShouldBeNil)
					})
					Convey("The metaState object should be ok to alter user", func() {
						err = ms.alterSQLChainUser(dbID3, addr2, types.UserPermissionFromRole(types.Read))
						So(err, ShouldBeNil)
						err = ms.alterSQLChainUser(dbID3, addr2, types.UserPermissionFromRole(types.Write))
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
							err = ms.deleteSQLChainUser(dbID3, addr2)
							So(err, ShouldBeNil)
							err = ms.deleteSQLChainUser(dbID3, addr2)
							So(err, ShouldBeNil)
						})
						Convey("The metaState object should be ok to alter user", func() {
							err = ms.alterSQLChainUser(dbID3, addr2, types.UserPermissionFromRole(types.Read))
							So(err, ShouldBeNil)
							err = ms.alterSQLChainUser(dbID3, addr2, types.UserPermissionFromRole(types.Write))
							So(err, ShouldBeNil)
						})
					})
				})
				Convey("When metaState change is committed", func() {
					ms.commit()
					Convey("The metaState object should be ok to add users for database", func() {
						err = ms.addSQLChainUser(dbID3, addr2, types.UserPermissionFromRole(types.Write))
						So(err, ShouldBeNil)
						err = ms.addSQLChainUser(dbID3, addr2, types.UserPermissionFromRole(types.Write))
						So(err, ShouldEqual, ErrDatabaseUserExists)
					})
					Convey("The metaState object should report database exists", func() {
						err = ms.createSQLChain(addr1, dbID3)
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
					co, loaded = ms.loadSQLChainObject(dbID1)
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
					bl, loaded = ms.loadAccountTokenBalance(addr1, types.Particle)
					So(loaded, ShouldBeTrue)
					So(bl, ShouldEqual, incSta)
					bl, loaded = ms.loadAccountTokenBalance(addr1, types.Wave)
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
							bl, loaded = ms.loadAccountTokenBalance(addr1, types.Particle)
							So(loaded, ShouldBeTrue)
							So(bl, ShouldEqual, incSta)
							bl, loaded = ms.loadAccountTokenBalance(addr1, types.Wave)
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
							tran1 := &types.Transfer{
								TransferHeader: types.TransferHeader{
									Sender:    addr1,
									Receiver:  addr2,
									Amount:    incSta + 1,
									TokenType: types.Particle,
									Nonce:     1,
								},
							}
							err = tran1.Sign(privKey1)
							So(err, ShouldBeNil)
							err = ms.transferAccountToken(tran1)
							So(err, ShouldEqual, ErrInsufficientBalance)
							tran2 := &types.Transfer{
								TransferHeader: types.TransferHeader{
									Sender:    addr1,
									Receiver:  addr3,
									Amount:    1,
									TokenType: types.Particle,
									Nonce:     1,
								},
							}
							err = tran2.Sign(privKey1)
							So(err, ShouldBeNil)
							err = ms.transferAccountToken(tran2)
							So(err, ShouldBeNil)
							ms.commit()

							err = ms.increaseAccountStableBalance(addr2, math.MaxUint64)
							So(err, ShouldBeNil)

							ms.commit()

							tran3 := &types.Transfer{
								TransferHeader: types.TransferHeader{
									Sender:    addr2,
									Receiver:  addr1,
									Amount:    math.MaxUint64,
									TokenType: types.Particle,
									Nonce:     1,
								},
							}
							err = tran3.Sign(privKey2)
							So(err, ShouldBeNil)
							err = ms.transferAccountToken(tran3)
							So(err, ShouldEqual, ErrBalanceOverflow)
							tran4 := &types.Transfer{
								TransferHeader: types.TransferHeader{
									Sender:    addr2,
									Receiver:  addr3,
									Amount:    1,
									TokenType: types.Particle,
									Nonce:     1,
								},
							}
							err = tran4.Sign(privKey2)
							So(err, ShouldBeNil)
							err = ms.transferAccountToken(tran4)
							So(err, ShouldBeNil)
							ms.commit()
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
					_, loaded = ms.loadSQLChainObject(dbID1)
					So(loaded, ShouldBeTrue)
					_, loaded = ms.loadOrStoreSQLChainObject(dbID2, nil)
					So(loaded, ShouldBeTrue)
				})
				Convey("When some objects are deleted", func() {
					ms.deleteAccountObject(addr1)
					ms.deleteSQLChainObject(dbID1)
					Convey("The dirty map should return deleted states of these objects", func() {
						_, loaded = ms.loadAccountObject(addr1)
						So(loaded, ShouldBeFalse)
						_, loaded = ms.loadSQLChainObject(dbID1)
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
				bl, loaded = ms.loadAccountTokenBalance(addr1, types.Particle)
				So(loaded, ShouldBeTrue)
				So(bl, ShouldEqual, 84)
				bl, loaded = ms.loadAccountTokenBalance(addr2, types.Particle)
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
							TokenBalance: [types.SupportTokenNumber]uint64{1000000000, 1000000000},
						},
					),
					types.NewBaseAccount(
						&types.Account{
							Address:      addr2,
							TokenBalance: [types.SupportTokenNumber]uint64{10000000000, 100},
						},
					),
					types.NewBaseAccount(
						&types.Account{
							Address:      addr3,
							TokenBalance: [types.SupportTokenNumber]uint64{10000, 10000},
						},
					),
					types.NewBaseAccount(
						&types.Account{
							Address:      addr4,
							TokenBalance: [types.SupportTokenNumber]uint64{1000000000, 1000000000},
						},
					),
				}
			)

			err = txs[0].Sign(privKey1)
			So(err, ShouldBeNil)
			err = txs[1].Sign(privKey2)
			So(err, ShouldBeNil)
			err = txs[2].Sign(privKey3)
			So(err, ShouldBeNil)
			err = txs[3].Sign(privKey4)
			So(err, ShouldBeNil)
			for i := range txs {
				err = ms.apply(txs[i])
				So(err, ShouldBeNil)
				ms.commit()
			}

			Convey("When provider transaction is invalid", func() {
				invalidPs := types.ProvideService{
					ProvideServiceHeader: types.ProvideServiceHeader{
						TargetUser: []proto.AccountAddress{addr1},
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
							Node:         1,
						},
						GasPrice:       1,
						AdvancePayment: uint64(conf.GConf.QPS) * conf.GConf.BillingBlockCount * 1,
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
							Node:         1,
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
							Node:         1,
						},
						Nonce: 1,
					},
				}
				err = invalidCd4.Sign(privKey3)
				So(err, ShouldBeNil)
				invalidCd5 := types.CreateDatabase{
					CreateDatabaseHeader: types.CreateDatabaseHeader{
						Owner: addr3,
						ResourceMeta: types.ResourceMeta{
							TargetMiners: []proto.AccountAddress{addr2},
							Node:         2,
						},
						Nonce:          1,
						GasPrice:       1,
						AdvancePayment: uint64(conf.GConf.QPS) * conf.GConf.BillingBlockCount * 2,
					},
				}
				err = invalidCd5.Sign(privKey3)
				So(err, ShouldBeNil)
				invalidCd6 := types.CreateDatabase{
					CreateDatabaseHeader: types.CreateDatabaseHeader{
						Owner: addr3,
						ResourceMeta: types.ResourceMeta{
							TargetMiners: []proto.AccountAddress{addr2},
							Node:         0,
						},
						Nonce:          1,
						GasPrice:       1,
						AdvancePayment: uint64(conf.GConf.QPS) * conf.GConf.BillingBlockCount * 1,
					},
				}
				err = invalidCd6.Sign(privKey3)
				So(err, ShouldBeNil)
				invalidCd7 := types.CreateDatabase{
					CreateDatabaseHeader: types.CreateDatabaseHeader{
						Owner: addr3,
						ResourceMeta: types.ResourceMeta{
							TargetMiners:           []proto.AccountAddress{addr2},
							Node:                   2,
							Space:                  9,
							Memory:                 9,
							LoadAvgPerCPU:          0.1,
							UseEventualConsistency: false,
							ConsistencyLevel:       0,
						},
						Nonce:          1,
						GasPrice:       1,
						AdvancePayment: uint64(conf.GConf.QPS) * conf.GConf.BillingBlockCount * 2,
					},
				}
				err = invalidCd7.Sign(privKey3)
				So(err, ShouldBeNil)
				invalidCd8 := types.CreateDatabase{
					CreateDatabaseHeader: types.CreateDatabaseHeader{
						Owner: addr2,
						ResourceMeta: types.ResourceMeta{
							TargetMiners:           []proto.AccountAddress{addr2},
							Node:                   2,
							Space:                  9,
							Memory:                 9,
							LoadAvgPerCPU:          0.1,
							UseEventualConsistency: false,
							ConsistencyLevel:       0,
						},
						Nonce:          1,
						GasPrice:       1,
						AdvancePayment: uint64(conf.GConf.QPS) * uint64(conf.GConf.BillingBlockCount) * 2,
					},
				}
				err = invalidCd8.Sign(privKey2)
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
				err = ms.apply(&invalidCd5)
				So(errors.Cause(err), ShouldEqual, ErrNoEnoughMiner)
				err = ms.apply(&invalidCd6)
				So(errors.Cause(err), ShouldEqual, ErrInvalidMinerCount)
				ms.dirty.provider[proto.AccountAddress(hash.HashH([]byte("1")))] = &types.ProviderProfile{
					TargetUser: nil,
				}
				ms.readonly.provider[proto.AccountAddress(hash.HashH([]byte("2")))] = &types.ProviderProfile{
					TargetUser: nil,
				}
				ms.dirty.provider[proto.AccountAddress(hash.HashH([]byte("3")))] = &types.ProviderProfile{
					TargetUser: []proto.AccountAddress{addr3},
					GasPrice:   9999999999, // not pass
				}
				ms.dirty.provider[proto.AccountAddress(hash.HashH([]byte("4")))] = &types.ProviderProfile{
					TargetUser:    []proto.AccountAddress{addr3},
					GasPrice:      1,
					LoadAvgPerCPU: 1.0, // not pass
				}
				ms.dirty.provider[proto.AccountAddress(hash.HashH([]byte("5")))] = &types.ProviderProfile{
					TargetUser:    []proto.AccountAddress{addr3},
					GasPrice:      1,
					LoadAvgPerCPU: 0.001,
					Memory:        1, // not pass
				}
				ms.dirty.provider[proto.AccountAddress(hash.HashH([]byte("6")))] = &types.ProviderProfile{
					TargetUser:    []proto.AccountAddress{addr3},
					GasPrice:      1,
					LoadAvgPerCPU: 0.001,
					Memory:        100,
					Space:         1, // not pass
				}
				ms.dirty.provider[proto.AccountAddress(hash.HashH([]byte("7")))] = &types.ProviderProfile{
					TargetUser:    []proto.AccountAddress{addr3},
					GasPrice:      1,
					LoadAvgPerCPU: 0.001,
					Memory:        100,
					Space:         100,
					TokenType:     1, // not pass
				}
				ms.readonly.provider[proto.AccountAddress(hash.HashH([]byte("8")))] = &types.ProviderProfile{
					Provider:      proto.AccountAddress{},
					Space:         0,
					Memory:        0,
					LoadAvgPerCPU: 0,
					TargetUser:    []proto.AccountAddress{addr3},
					Deposit:       0,
					GasPrice:      0,
					TokenType:     0,
					NodeID:        "",
				}
				err = ms.apply(&invalidCd7)
				So(errors.Cause(err), ShouldEqual, ErrNoEnoughMiner)

				ms.readonly.provider[proto.AccountAddress(hash.HashH([]byte("9")))] = &types.ProviderProfile{
					TargetUser:    []proto.AccountAddress{addr2},
					GasPrice:      1,
					LoadAvgPerCPU: 0.001,
					Memory:        100,
					Space:         100,
					TokenType:     0,
					NodeID:        "0001111",
				}
				ms.dirty.provider[proto.AccountAddress(hash.HashH([]byte("9")))] = &types.ProviderProfile{
					TargetUser:    []proto.AccountAddress{addr2},
					GasPrice:      1,
					LoadAvgPerCPU: 0.001,
					Memory:        100,
					Space:         100,
					TokenType:     0,
					NodeID:        "0002111",
				}
				ms.dirty.provider[proto.AccountAddress(hash.HashH([]byte("10")))] = &types.ProviderProfile{
					TargetUser:    []proto.AccountAddress{addr2},
					GasPrice:      1,
					LoadAvgPerCPU: 0.001,
					Memory:        100,
					Space:         100,
					TokenType:     0,
					NodeID:        "0003111",
				}
				ms.dirty.provider[proto.AccountAddress(hash.HashH([]byte("11")))] = &types.ProviderProfile{
					TargetUser:    []proto.AccountAddress{addr2},
					GasPrice:      1,
					LoadAvgPerCPU: 0.001,
					Memory:        100,
					Space:         100,
					TokenType:     0,
					NodeID:        "0000003",
				}
				ms.dirty.provider[proto.AccountAddress(hash.HashH([]byte("12")))] = &types.ProviderProfile{
					TargetUser:    []proto.AccountAddress{addr2},
					GasPrice:      1,
					LoadAvgPerCPU: 0.001,
					Memory:        100,
					Space:         100,
					TokenType:     0,
					NodeID:        "0000001",
				}
				err = ms.apply(&invalidCd8)
				So(err, ShouldBeNil)
				dbID := proto.FromAccountAndNonce(addr2, uint32(invalidCd8.Nonce))

				mIDs := make([]string, 0)
				for _, m := range ms.dirty.databases[dbID].Miners {
					mIDs = append(mIDs, string(m.NodeID))
				}
				log.Debugf("mIDs: %v", mIDs)
				So(mIDs, ShouldContain, "0000003")
				So(mIDs, ShouldContain, "0000001")
			})
			Convey("When SQLChain create", func() {
				ps := types.ProvideService{
					ProvideServiceHeader: types.ProvideServiceHeader{
						TargetUser: []proto.AccountAddress{addr1},
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
							Node:         1,
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
							Node:         1,
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
				b1, loaded = ms.loadAccountTokenBalance(addr2, types.Particle)
				err = ms.apply(&ps)
				So(err, ShouldBeNil)
				ms.commit()
				b2, loaded = ms.loadAccountTokenBalance(addr2, types.Particle)
				So(loaded, ShouldBeTrue)
				So(b1-b2, ShouldEqual, conf.GConf.MinProviderDeposit)
				err = ms.apply(&cd2)
				So(errors.Cause(err), ShouldEqual, ErrMinerUserNotMatch)
				b1, loaded = ms.loadAccountTokenBalance(addr1, types.Particle)
				So(loaded, ShouldBeTrue)
				err = ms.apply(&cd1)
				So(err, ShouldBeNil)
				ms.commit()
				b2, loaded = ms.loadAccountTokenBalance(addr1, types.Particle)
				So(loaded, ShouldBeTrue)
				minAdvancePayment := uint64(cd2.GasPrice) * uint64(conf.GConf.QPS) *
					conf.GConf.BillingBlockCount * uint64(len(cd2.ResourceMeta.TargetMiners))
				So(b1-b2, ShouldEqual, cd1.AdvancePayment+minAdvancePayment)
				dbID := proto.FromAccountAndNonce(cd1.Owner, uint32(cd1.Nonce))
				co, loaded = ms.loadSQLChainObject(dbID)
				So(loaded, ShouldBeTrue)
				dbAccount, err := dbID.AccountAddress()
				So(err, ShouldBeNil)

				up := types.UpdatePermission{
					UpdatePermissionHeader: types.UpdatePermissionHeader{
						TargetSQLChain: addr1,
						TargetUser:     addr3,
						Permission:     types.UserPermissionFromRole(types.Read),
						Nonce:          cd1.Nonce + 1,
					},
				}
				err = up.Sign(privKey1)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(errors.Cause(err), ShouldEqual, ErrDatabaseNotFound)
				up.Permission = types.UserPermissionFromRole(types.Void)
				up.TargetSQLChain = dbAccount
				err = up.Sign(privKey1)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(errors.Cause(err), ShouldEqual, ErrInvalidPermission)
				// test permission update
				// addr1(admin) update addr3 as admin
				up.TargetUser = addr3
				up.Permission = types.UserPermissionFromRole(types.Admin)
				err = up.Sign(privKey1)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(err, ShouldBeNil)
				ms.commit()
				// addr3(admin) update addr4 as read
				up.TargetUser = addr4
				up.Nonce = cd2.Nonce
				up.Permission = types.UserPermissionFromRole(types.Read)
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
				up.Permission = types.UserPermissionFromRole(types.Read)
				up.Nonce = up.Nonce + 1
				err = up.Sign(privKey3)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(errors.Cause(err), ShouldEqual, ErrNoSuperUserLeft)
				// addr1(read) update addr3(admin) fail
				up.Nonce = cd1.Nonce + 2
				err = up.Sign(privKey1)
				So(err, ShouldBeNil)
				err = ms.apply(&up)
				So(errors.Cause(err), ShouldEqual, ErrAccountPermissionDeny)

				co, loaded = ms.loadSQLChainObject(dbID)
				for _, user := range co.Users {
					if user.Address == addr1 {
						So(user.Permission, ShouldNotBeNil)
						So(user.Permission.Role, ShouldEqual, types.Read)
						continue
					}
					if user.Address == addr3 {
						So(user.Permission, ShouldNotBeNil)
						So(user.Permission.Role, ShouldEqual, types.Admin)
						continue
					}
					if user.Address == addr4 {
						So(user.Permission, ShouldNotBeNil)
						So(user.Permission.Role, ShouldEqual, types.Read)
						continue
					}
				}
				Convey("transfer token", func() {
					addr1B1, ok := ms.loadAccountTokenBalance(addr1, types.Particle)
					So(ok, ShouldBeTrue)
					addr3B1, ok := ms.loadAccountTokenBalance(addr3, types.Particle)
					So(ok, ShouldBeTrue)
					trans1 := types.NewTransfer(&types.TransferHeader{
						Sender:    addr1,
						Receiver:  addr3,
						Amount:    20000000,
						TokenType: types.Particle,
					})
					nonce, err := ms.nextNonce(addr1)
					So(err, ShouldBeNil)
					trans1.Nonce = nonce
					err = trans1.Sign(privKey1)
					So(err, ShouldBeNil)
					err = ms.apply(trans1)
					So(err, ShouldBeNil)
					ms.commit()
					addr1B2, ok := ms.loadAccountTokenBalance(addr1, types.Particle)
					So(ok, ShouldBeTrue)
					addr3B2, ok := ms.loadAccountTokenBalance(addr3, types.Particle)
					So(ok, ShouldBeTrue)
					So(addr1B1-addr1B2, ShouldEqual, 20000000)
					So(addr3B2-addr3B1, ShouldEqual, 20000000)
					profile, ok := ms.loadSQLChainObject(dbID)
					So(ok, ShouldBeTrue)

					// transfer to sqlchain
					for _, user := range profile.Users {
						if user.Address == addr3 {
							So(user.Status, ShouldEqual, types.UnknownStatus)
							break
						}
					}
					trans2 := types.NewTransfer(&types.TransferHeader{
						Sender:    addr3,
						Receiver:  dbAccount,
						Amount:    8000000,
						TokenType: types.Particle,
					})
					nonce, err = ms.nextNonce(addr3)
					So(err, ShouldBeNil)
					So(dbID, ShouldEqual, dbAccount.DatabaseID())
					trans2.Nonce = nonce
					err = trans2.Sign(privKey3)
					So(err, ShouldBeNil)
					err = ms.apply(trans2)
					So(err, ShouldBeNil)
					// ms.commit()
					profile, ok = ms.loadSQLChainObject(dbID)
					So(ok, ShouldBeTrue)
					for _, user := range profile.Users {
						if user.Address == addr3 {
							So(user.Status, ShouldEqual, types.Normal)
							break
						}
					}

					// make addr3 arrears
					ub := types.NewUpdateBilling(&types.UpdateBillingHeader{
						Receiver: dbAccount,
						Users: []*types.UserCost{
							&types.UserCost{
								User: addr3,
								Cost: 4500000,
								Miners: []*types.MinerIncome{
									&types.MinerIncome{
										Miner:  addr2,
										Income: 4500000,
									},
								},
							},
						},
					})
					nonce, err = ms.nextNonce(addr2)
					So(err, ShouldBeNil)
					ub.Nonce = nonce
					err = ub.Sign(privKey2)
					So(err, ShouldBeNil)
					err = ms.apply(ub)
					So(err, ShouldBeNil)
					ms.commit()
					profile, ok = ms.loadSQLChainObject(dbID)
					So(ok, ShouldBeTrue)
					for _, user := range profile.Users {
						if user.Address == addr3 {
							So(user.Status, ShouldEqual, types.Arrears)
							break
						}
					}

					// transfer failed
					trans3 := types.NewTransfer(&types.TransferHeader{
						Sender:    addr3,
						Receiver:  dbAccount,
						Amount:    40000,
						TokenType: types.Particle,
					})
					nonce, err = ms.nextNonce(addr3)
					So(err, ShouldBeNil)
					trans3.Nonce = nonce
					err = trans3.Sign(privKey3)
					So(err, ShouldBeNil)
					err = ms.apply(trans3)
					So(err, ShouldEqual, ErrInsufficientTransfer)
					profile, ok = ms.loadSQLChainObject(dbID)
					So(ok, ShouldBeTrue)
					for _, user := range profile.Users {
						if user.Address == addr3 {
							So(user.Status, ShouldEqual, types.Arrears)
							break
						}
					}

					// transfer enough token
					trans4 := types.NewTransfer(&types.TransferHeader{
						Sender:    addr3,
						Receiver:  dbAccount,
						Amount:    4000000,
						TokenType: types.Particle,
					})
					nonce, err = ms.nextNonce(addr3)
					So(err, ShouldBeNil)
					trans4.Nonce = nonce
					err = trans4.Sign(privKey3)
					So(err, ShouldBeNil)
					err = ms.apply(trans4)
					ms.commit()
					profile, ok = ms.loadSQLChainObject(dbID)
					So(ok, ShouldBeTrue)
					for _, user := range profile.Users {
						if user.Address == addr3 {
							So(user.Status, ShouldEqual, types.Normal)
							break
						}
					}

				})
				Convey("update key", func() {
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

					co, loaded = ms.loadSQLChainObject(dbID)
					for _, miner := range co.Miners {
						if miner.Address == addr1 {
							So(miner.EncryptionKey, ShouldEqual, encryptKey)
						}
					}
				})
				Convey("update billing", func() {
					ub1 := &types.UpdateBilling{
						UpdateBillingHeader: types.UpdateBillingHeader{
							Receiver: addr1,
							Nonce:    up.Nonce,
						},
					}
					err = ub1.Sign(privKey1)
					So(err, ShouldBeNil)
					err = ms.apply(ub1)
					So(errors.Cause(err), ShouldEqual, ErrDatabaseNotFound)
					trans1 := types.NewTransfer(&types.TransferHeader{
						Sender:    addr1,
						Receiver:  dbAccount,
						Amount:    8000000,
						TokenType: types.Particle,
					})
					nonce, err := ms.nextNonce(addr1)
					So(err, ShouldBeNil)
					trans1.Nonce = nonce
					err = trans1.Sign(privKey1)
					So(err, ShouldBeNil)
					err = ms.apply(trans1)
					So(err, ShouldBeNil)
					ms.commit()
					trans2 := types.NewTransfer(&types.TransferHeader{
						Sender:    addr3,
						Receiver:  dbAccount,
						Amount:    800,
						TokenType: types.Particle,
					})
					nonce, err = ms.nextNonce(addr3)
					So(err, ShouldBeNil)
					trans2.Nonce = nonce
					err = trans2.Sign(privKey3)
					So(err, ShouldBeNil)
					err = ms.apply(trans2)
					So(err, ShouldBeNil)
					ms.commit()
					trans3 := types.NewTransfer(&types.TransferHeader{
						Sender:    addr4,
						Receiver:  dbAccount,
						Amount:    8000000,
						TokenType: types.Particle,
					})
					nonce, err = ms.nextNonce(addr4)
					So(err, ShouldBeNil)
					trans3.Nonce = nonce
					err = trans3.Sign(privKey4)
					So(err, ShouldBeNil)
					err = ms.apply(trans3)
					So(err, ShouldBeNil)
					ms.commit()

					users := [3]*types.UserCost{
						&types.UserCost{
							User: addr1,
							Cost: 100,
							Miners: []*types.MinerIncome{
								&types.MinerIncome{
									Miner:  addr2,
									Income: 100,
								},
							},
						},
						&types.UserCost{
							User: addr3,
							Cost: 10,
							Miners: []*types.MinerIncome{
								&types.MinerIncome{
									Miner:  addr2,
									Income: 10,
								},
							},
						},
						&types.UserCost{
							User: addr4,
							Cost: 15,
							Miners: []*types.MinerIncome{
								&types.MinerIncome{
									Miner:  addr2,
									Income: 15,
								},
							},
						},
					}
					ub2 := &types.UpdateBilling{
						UpdateBillingHeader: types.UpdateBillingHeader{
							Receiver: dbAccount,
							Users:    users[:],
							Nonce:    2,
						},
					}
					err = ub2.Sign(privKey2)
					So(err, ShouldBeNil)
					err = ms.apply(ub2)
					ms.commit()
					sqlchain, loaded := ms.loadSQLChainObject(dbID)
					So(loaded, ShouldBeTrue)
					So(len(sqlchain.Miners), ShouldEqual, 1)
					So(sqlchain.Miners[0].PendingIncome, ShouldEqual, 115)
					users = [3]*types.UserCost{
						&types.UserCost{
							User: addr1,
							Cost: 100,
							Miners: []*types.MinerIncome{
								&types.MinerIncome{
									Miner:  addr2,
									Income: 100,
								},
							},
						},
						&types.UserCost{
							User: addr2,
							Cost: 10,
							Miners: []*types.MinerIncome{
								&types.MinerIncome{
									Miner:  addr3,
									Income: 10,
								},
							},
						},
						&types.UserCost{
							User: addr4,
							Cost: 15,
							Miners: []*types.MinerIncome{
								&types.MinerIncome{
									Miner:  addr2,
									Income: 15,
								},
							},
						},
					}
					ub3 := &types.UpdateBilling{
						UpdateBillingHeader: types.UpdateBillingHeader{
							Receiver: dbAccount,
							Users:    users[:],
							Nonce:    3,
						},
					}
					err = ub3.Sign(privKey2)
					So(err, ShouldBeNil)
					err = ms.apply(ub3)
					So(err, ShouldBeNil)
					sqlchain, loaded = ms.loadSQLChainObject(dbID)
					So(loaded, ShouldBeTrue)
					So(len(sqlchain.Miners), ShouldEqual, 1)
					So(sqlchain.Miners[0].PendingIncome, ShouldEqual, 115)
					So(sqlchain.Miners[0].ReceivedIncome, ShouldEqual, 115)
				})
			})
		})
	})
}
