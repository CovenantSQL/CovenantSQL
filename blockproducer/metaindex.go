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
	"bytes"
	"sync"

	pt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/coreos/bbolt"
)

// safeAdd provides a safe add method with upper overflow check for uint64.
func safeAdd(x, y *uint64) (err error) {
	if *x+*y < *x {
		return ErrBalanceOverflow
	}
	*x += *y
	return
}

// safeAdd provides a safe sub method with lower overflow check for uint64.
func safeSub(x, y *uint64) (err error) {
	if *x < *y {
		return ErrInsufficientBalance
	}
	*x -= *y
	return
}

type accountObject struct {
	sync.RWMutex
	pt.Account
}

type sqlchainObject struct {
	sync.RWMutex
	pt.SQLChainProfile
}

type metaIndex struct {
	sync.RWMutex
	accounts  map[proto.AccountAddress]*accountObject
	databases map[proto.DatabaseID]*sqlchainObject
}

func newMetaIndex() *metaIndex {
	return &metaIndex{
		accounts:  make(map[proto.AccountAddress]*accountObject),
		databases: make(map[proto.DatabaseID]*sqlchainObject),
	}
}

func (i *metaIndex) storeAccountObject(o *accountObject) {
	i.Lock()
	defer i.Unlock()
	i.accounts[o.Address] = o
}

func (i *metaIndex) deleteAccountObject(k proto.AccountAddress) {
	i.Lock()
	defer i.Unlock()
	delete(i.accounts, k)
}

func (i *metaIndex) storeSQLChainObject(o *sqlchainObject) {
	i.Lock()
	defer i.Unlock()
	i.databases[o.ID] = o
}

func (i *metaIndex) deleteSQLChainObject(k proto.DatabaseID) {
	i.Lock()
	defer i.Unlock()
	delete(i.databases, k)
}

// IncreaseAccountStableBalance increases account stable coin balance and write persistence within
// a boltdb transaction.
func (i *metaIndex) IncreaseAccountStableBalance(
	addr proto.AccountAddress, amount uint64) (_ func(*bolt.Tx) error,
) {
	return func(tx *bolt.Tx) (err error) {
		var (
			ao  *accountObject
			ok  bool
			bk  = tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
			enc *bytes.Buffer
		)

		i.Lock()
		defer i.Unlock()
		if ao, ok = i.accounts[addr]; !ok {
			err = ErrAccountNotFound
			return
		}
		if err = safeAdd(&ao.StableCoinBalance, &amount); err != nil {
			return
		}
		ao.NextNonce++

		if enc, err = utils.EncodeMsgPack(&ao.Account); err != nil {
			return
		}
		if err = bk.Put(ao.Address[:], enc.Bytes()); err != nil {
			return
		}

		return
	}
}

// DecreaseAccountStableBalance decreases account stable coin balance and write persistence within
// a boltdb transaction.
func (i *metaIndex) DecreaseAccountStableBalance(
	addr proto.AccountAddress, amount uint64) (_ func(*bolt.Tx) error,
) {
	return func(tx *bolt.Tx) (err error) {
		var (
			ao  *accountObject
			ok  bool
			bk  = tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
			enc *bytes.Buffer
		)

		i.Lock()
		defer i.Unlock()
		if ao, ok = i.accounts[addr]; !ok {
			err = ErrAccountNotFound
			return
		}
		if err = safeSub(&ao.StableCoinBalance, &amount); err != nil {
			return
		}
		ao.NextNonce++

		if enc, err = utils.EncodeMsgPack(&ao.Account); err != nil {
			return
		}
		if err = bk.Put(ao.Address[:], enc.Bytes()); err != nil {
			return
		}

		return
	}
}

// IncreaseAccountCovenantBalance increases account covenant coin balance and write persistence
// within a boltdb transaction.
func (i *metaIndex) IncreaseAccountCovenantBalance(
	addr proto.AccountAddress, amount uint64) (_ func(*bolt.Tx) error,
) {
	return func(tx *bolt.Tx) (err error) {
		var (
			ao  *accountObject
			ok  bool
			bk  = tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
			enc *bytes.Buffer
		)

		i.Lock()
		defer i.Unlock()
		if ao, ok = i.accounts[addr]; !ok {
			err = ErrAccountNotFound
			return
		}
		if err = safeAdd(&ao.CovenantCoinBalance, &amount); err != nil {
			return
		}
		ao.NextNonce++

		if enc, err = utils.EncodeMsgPack(&ao.Account); err != nil {
			return
		}
		if err = bk.Put(ao.Address[:], enc.Bytes()); err != nil {
			return
		}

		return
	}
}

// DecreaseAccountCovenantBalance decreases account covenant coin balance and write persistence
// within a boltdb transaction.
func (i *metaIndex) DecreaseAccountCovenantBalance(
	addr proto.AccountAddress, amount uint64) (_ func(*bolt.Tx) error,
) {
	return func(tx *bolt.Tx) (err error) {
		var (
			ao  *accountObject
			ok  bool
			bk  = tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
			enc *bytes.Buffer
		)

		i.Lock()
		defer i.Unlock()
		if ao, ok = i.accounts[addr]; !ok {
			err = ErrAccountNotFound
			return
		}
		if err = safeSub(&ao.CovenantCoinBalance, &amount); err != nil {
			return
		}
		ao.NextNonce++

		if enc, err = utils.EncodeMsgPack(&ao.Account); err != nil {
			return
		}
		if err = bk.Put(ao.Address[:], enc.Bytes()); err != nil {
			return
		}

		return
	}
}

func (i *metaIndex) CreateSQLChain(
	addr proto.AccountAddress, id proto.DatabaseID) (_ func(*bolt.Tx) error,
) {
	return func(tx *bolt.Tx) (err error) {
		var (
			ao  *accountObject
			co  *sqlchainObject
			ok  bool
			bk  = tx.Bucket(metaBucket[:]).Bucket(metaSQLChainIndexBucket)
			enc *bytes.Buffer
		)

		i.Lock()
		defer i.Unlock()
		// Make sure that the target account exists
		if ao, ok = i.accounts[addr]; !ok {
			err = ErrAccountNotFound
			return
		}
		// Create new sqlchainProfile
		co = &sqlchainObject{
			SQLChainProfile: pt.SQLChainProfile{
				ID:    id,
				Owner: addr,
			},
		}
		i.databases[id] = co
		ao.NextNonce++

		if enc, err = utils.EncodeMsgPack(co); err != nil {
			return
		}
		if err = bk.Put([]byte(id), enc.Bytes()); err != nil {
			return
		}

		return
	}
}
