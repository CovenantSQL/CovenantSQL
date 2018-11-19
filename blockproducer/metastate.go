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

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	pt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/coreos/bbolt"
	"github.com/ulule/deepcopier"
)

// TODO(leventeliu): lock optimization.

type metaState struct {
	sync.RWMutex
	dirty, readonly *metaIndex
	pool            *txPool
}

func newMetaState() *metaState {
	return &metaState{
		dirty:    newMetaIndex(),
		readonly: newMetaIndex(),
		pool:     newTxPool(),
	}
}

func (s *metaState) loadAccountObject(k proto.AccountAddress) (o *accountObject, loaded bool) {
	s.RLock()
	defer s.RUnlock()
	if o, loaded = s.dirty.accounts[k]; loaded {
		if o == nil {
			loaded = false
		}
		return
	}
	if o, loaded = s.readonly.accounts[k]; loaded {
		return
	}
	return
}

func (s *metaState) loadOrStoreAccountObject(
	k proto.AccountAddress, v *accountObject) (o *accountObject, loaded bool,
) {
	s.Lock()
	defer s.Unlock()
	if o, loaded = s.dirty.accounts[k]; loaded && o != nil {
		return
	}
	if o, loaded = s.readonly.accounts[k]; loaded {
		return
	}
	s.dirty.accounts[k] = v
	return
}

func (s *metaState) loadAccountStableBalance(addr proto.AccountAddress) (b uint64, loaded bool) {
	var o *accountObject
	defer func() {
		log.WithFields(log.Fields{
			"account": addr.String(),
			"balance": b,
			"loaded":  loaded,
		}).Debug("queried stable account")
	}()
	
	s.Lock()
	defer s.Unlock()

	if o, loaded = s.dirty.accounts[addr]; loaded && o != nil {
		b = o.StableCoinBalance
		return
	}
	if o, loaded = s.readonly.accounts[addr]; loaded {
		b = o.StableCoinBalance
		return
	}
	return
}

func (s *metaState) loadAccountCovenantBalance(addr proto.AccountAddress) (b uint64, loaded bool) {
	var o *accountObject
	defer func() {
		log.WithFields(log.Fields{
			"account": addr.String(),
			"balance": b,
			"loaded":  loaded,
		}).Debug("queried covenant account")
	}()

	s.Lock()
	defer s.Unlock()

	if o, loaded = s.dirty.accounts[addr]; loaded && o != nil {
		b = o.CovenantCoinBalance
		return
	}
	if o, loaded = s.readonly.accounts[addr]; loaded {
		b = o.CovenantCoinBalance
		return
	}
	return
}

func (s *metaState) storeBaseAccount(k proto.AccountAddress, v *accountObject) (err error) {
	log.WithFields(log.Fields{
		"addr":    k.String(),
		"account": v,
	}).Debug("store account")
	// Since a transfer tx may create an empty receiver account, this method should try to cover
	// the side effect.
	if ao, ok := s.loadOrStoreAccountObject(k, v); ok {
		ao.Lock()
		defer ao.Unlock()
		if ao.Account.NextNonce != 0 {
			err = ErrAccountExists
			return
		}
		var (
			cb = ao.CovenantCoinBalance
			sb = ao.StableCoinBalance
		)
		if err = safeAdd(&cb, &v.Account.CovenantCoinBalance); err != nil {
			return
		}
		if err = safeAdd(&sb, &v.Account.StableCoinBalance); err != nil {
			return
		}
		ao.CovenantCoinBalance = cb
		ao.StableCoinBalance = sb
	}
	return
}

func (s *metaState) loadSQLChainObject(k proto.DatabaseID) (o *sqlchainObject, loaded bool) {
	s.RLock()
	defer s.RUnlock()
	if o, loaded = s.dirty.databases[k]; loaded {
		if o == nil {
			loaded = false
		}
		return
	}
	if o, loaded = s.readonly.databases[k]; loaded {
		return
	}
	return
}

func (s *metaState) loadOrStoreSQLChainObject(
	k proto.DatabaseID, v *sqlchainObject) (o *sqlchainObject, loaded bool,
) {
	s.Lock()
	defer s.Unlock()
	if o, loaded = s.dirty.databases[k]; loaded && o != nil {
		return
	}
	if o, loaded = s.readonly.databases[k]; loaded {
		return
	}
	s.dirty.databases[k] = v
	return
}

func (s *metaState) deleteAccountObject(k proto.AccountAddress) {
	s.Lock()
	defer s.Unlock()
	// Use a nil pointer to mark a deletion, which will be later used by commit procedure.
	s.dirty.accounts[k] = nil
}

func (s *metaState) deleteSQLChainObject(k proto.DatabaseID) {
	s.Lock()
	defer s.Unlock()
	// Use a nil pointer to mark a deletion, which will be later used by commit procedure.
	s.dirty.databases[k] = nil
}

func (s *metaState) commitProcedure() (_ func(*bolt.Tx) error) {
	return func(tx *bolt.Tx) (err error) {
		var (
			enc *bytes.Buffer
			ab  = tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
			cb  = tx.Bucket(metaBucket[:]).Bucket(metaSQLChainIndexBucket)
		)
		s.Lock()
		defer s.Unlock()
		for k, v := range s.dirty.accounts {
			if v != nil {
				// New/update object
				s.readonly.accounts[k] = v
				if enc, err = utils.EncodeMsgPack(v.Account); err != nil {
					return
				}
				if err = ab.Put(k[:], enc.Bytes()); err != nil {
					return
				}
			} else {
				// Delete object
				delete(s.readonly.accounts, k)
				if err = ab.Delete(k[:]); err != nil {
					return
				}
			}
		}
		for k, v := range s.dirty.databases {
			if v != nil {
				// New/update object
				s.readonly.databases[k] = v
				if enc, err = utils.EncodeMsgPack(v.SQLChainProfile); err != nil {
					return
				}
				if err = cb.Put([]byte(k), enc.Bytes()); err != nil {
					return
				}
			} else {
				// Delete object
				delete(s.readonly.databases, k)
				if err = cb.Delete([]byte(k)); err != nil {
					return
				}
			}
		}
		// Clean dirty map and tx pool
		s.dirty = newMetaIndex()
		s.pool = newTxPool()
		return
	}
}

// partialCommitProcedure compares txs with pooled items, replays and commits the state due to txs
// if txs matches part of or all the pooled items. Not committed txs will be left in the pool.
func (s *metaState) partialCommitProcedure(txs []pi.Transaction) (_ func(*bolt.Tx) error) {
	return func(tx *bolt.Tx) (err error) {
		var (
			enc *bytes.Buffer
			ab  = tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
			cb  = tx.Bucket(metaBucket[:]).Bucket(metaSQLChainIndexBucket)
		)
		s.Lock()
		defer s.Unlock()

		// Make a half-deep copy of pool (txs are not copied, readonly) and deep copy of readonly
		// state
		var (
			cp = s.pool.halfDeepCopy()
			cm = &metaState{
				dirty:    newMetaIndex(),
				readonly: s.readonly.deepCopy(),
			}
		)
		// Compare and replay commits, stop whenever a tx has mismatched
		for _, v := range txs {
			if !cp.cmpAndMoveNextTx(v) {
				err = ErrTransactionMismatch
				return
			}
			if err = cm.applyTransaction(v); err != nil {
				return
			}
		}

		for k, v := range cm.dirty.accounts {
			if v != nil {
				// New/update object
				cm.readonly.accounts[k] = v
				if enc, err = utils.EncodeMsgPack(v.Account); err != nil {
					return
				}
				if err = ab.Put(k[:], enc.Bytes()); err != nil {
					return
				}
			} else {
				// Delete object
				delete(cm.readonly.accounts, k)
				if err = ab.Delete(k[:]); err != nil {
					return
				}
			}
		}
		for k, v := range cm.dirty.databases {
			if v != nil {
				// New/update object
				cm.readonly.databases[k] = v
				if enc, err = utils.EncodeMsgPack(v.SQLChainProfile); err != nil {
					return
				}
				if err = cb.Put([]byte(k), enc.Bytes()); err != nil {
					return
				}
			} else {
				// Delete object
				delete(cm.readonly.databases, k)
				if err = cb.Delete([]byte(k)); err != nil {
					return
				}
			}
		}

		// Rebuild dirty map
		cm.dirty = newMetaIndex()
		for _, v := range cp.entries {
			for _, tx := range v.transactions {
				if err = cm.applyTransaction(tx); err != nil {
					return
				}
			}
		}

		// Clean dirty map and tx pool
		s.pool = cp
		s.readonly = cm.readonly
		s.dirty = cm.dirty
		return
	}
}

func (s *metaState) reloadProcedure() (_ func(*bolt.Tx) error) {
	return func(tx *bolt.Tx) (err error) {
		s.Lock()
		defer s.Unlock()
		// Clean state
		s.dirty = newMetaIndex()
		s.readonly = newMetaIndex()
		// Reload state
		var (
			ab = tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
			cb = tx.Bucket(metaBucket[:]).Bucket(metaSQLChainIndexBucket)
		)
		if err = ab.ForEach(func(k, v []byte) (err error) {
			ao := &accountObject{}
			if err = utils.DecodeMsgPack(v, &ao.Account); err != nil {
				return
			}
			s.readonly.accounts[ao.Account.Address] = ao
			return
		}); err != nil {
			return
		}
		if err = cb.ForEach(func(k, v []byte) (err error) {
			co := &sqlchainObject{}
			if err = utils.DecodeMsgPack(v, &co.SQLChainProfile); err != nil {
				return
			}
			s.readonly.databases[co.SQLChainProfile.ID] = co
			return
		}); err != nil {
			return
		}
		return
	}
}

func (s *metaState) clean() {
	s.Lock()
	defer s.Unlock()
	s.dirty = newMetaIndex()
}

func (s *metaState) increaseAccountStableBalance(k proto.AccountAddress, amount uint64) error {
	s.Lock()
	defer s.Unlock()
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			return ErrAccountNotFound
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeAdd(&dst.Account.StableCoinBalance, &amount)
}

func (s *metaState) decreaseAccountStableBalance(k proto.AccountAddress, amount uint64) error {
	s.Lock()
	defer s.Unlock()
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			return ErrAccountNotFound
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeSub(&dst.Account.StableCoinBalance, &amount)
}

func (s *metaState) transferAccountStableBalance(
	sender, receiver proto.AccountAddress, amount uint64) (err error,
) {
	if sender == receiver || amount == 0 {
		return
	}

	// Create empty receiver account if not found
	s.loadOrStoreAccountObject(receiver, &accountObject{Account: pt.Account{Address: receiver}})

	s.Lock()
	defer s.Unlock()
	var (
		so, ro     *accountObject
		sd, rd, ok bool
	)

	// Load sender and receiver objects
	if so, sd = s.dirty.accounts[sender]; !sd {
		if so, ok = s.readonly.accounts[sender]; !ok {
			err = ErrAccountNotFound
			return
		}
	}
	if ro, rd = s.dirty.accounts[receiver]; !rd {
		if ro, ok = s.readonly.accounts[receiver]; !ok {
			err = ErrAccountNotFound
			return
		}
	}

	// Try transfer
	var (
		sb = so.StableCoinBalance
		rb = ro.StableCoinBalance
	)
	if err = safeSub(&sb, &amount); err != nil {
		return
	}
	if err = safeAdd(&rb, &amount); err != nil {
		return
	}

	// Proceed transfer
	if !sd {
		var cpy = &accountObject{}
		deepcopier.Copy(&so.Account).To(&cpy.Account)
		so = cpy
		s.dirty.accounts[sender] = cpy
	}
	if !rd {
		var cpy = &accountObject{}
		deepcopier.Copy(&ro.Account).To(&cpy.Account)
		ro = cpy
		s.dirty.accounts[receiver] = cpy
	}
	so.StableCoinBalance = sb
	ro.StableCoinBalance = rb
	return
}

func (s *metaState) increaseAccountCovenantBalance(k proto.AccountAddress, amount uint64) error {
	s.Lock()
	defer s.Unlock()
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			return ErrAccountNotFound
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeAdd(&dst.Account.CovenantCoinBalance, &amount)
}

func (s *metaState) decreaseAccountCovenantBalance(k proto.AccountAddress, amount uint64) error {
	s.Lock()
	defer s.Unlock()
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			return ErrAccountNotFound
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeSub(&dst.Account.CovenantCoinBalance, &amount)
}

func (s *metaState) createSQLChain(addr proto.AccountAddress, id proto.DatabaseID) error {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.dirty.accounts[addr]; !ok {
		if _, ok := s.readonly.accounts[addr]; !ok {
			return ErrAccountNotFound
		}
	}
	if _, ok := s.dirty.databases[id]; ok {
		return ErrDatabaseExists
	} else if _, ok := s.readonly.databases[id]; ok {
		return ErrDatabaseExists
	}
	s.dirty.databases[id] = &sqlchainObject{
		SQLChainProfile: pt.SQLChainProfile{
			ID:     id,
			Owner:  addr,
			Miners: make([]proto.AccountAddress, 0),
			Users: []*pt.SQLChainUser{
				{
					Address:    addr,
					Permission: pt.Admin,
				},
			},
		},
	}
	return nil
}

func (s *metaState) addSQLChainUser(
	k proto.DatabaseID, addr proto.AccountAddress, perm pt.UserPermission) (_ error,
) {
	s.Lock()
	defer s.Unlock()
	var (
		src, dst *sqlchainObject
		ok       bool
	)
	if dst, ok = s.dirty.databases[k]; !ok {
		if src, ok = s.readonly.databases[k]; !ok {
			return ErrDatabaseNotFound
		}
		dst = &sqlchainObject{}
		deepcopier.Copy(&src.SQLChainProfile).To(&dst.SQLChainProfile)
		s.dirty.databases[k] = dst
	}
	for _, v := range dst.Users {
		if v.Address == addr {
			return ErrDatabaseUserExists
		}
	}
	dst.SQLChainProfile.Users = append(dst.SQLChainProfile.Users, &pt.SQLChainUser{
		Address:    addr,
		Permission: perm,
	})
	return
}

func (s *metaState) deleteSQLChainUser(k proto.DatabaseID, addr proto.AccountAddress) error {
	s.Lock()
	defer s.Unlock()
	var (
		src, dst *sqlchainObject
		ok       bool
	)
	if dst, ok = s.dirty.databases[k]; !ok {
		if src, ok = s.readonly.databases[k]; !ok {
			return ErrDatabaseNotFound
		}
		dst = &sqlchainObject{}
		deepcopier.Copy(&src.SQLChainProfile).To(&dst.SQLChainProfile)
		s.dirty.databases[k] = dst
	}
	for i, v := range dst.Users {
		if v.Address == addr {
			last := len(dst.Users) - 1
			dst.Users[i] = dst.Users[last]
			dst.Users[last] = nil
			dst.Users = dst.Users[:last]
		}
	}
	return nil
}

func (s *metaState) alterSQLChainUser(
	k proto.DatabaseID, addr proto.AccountAddress, perm pt.UserPermission) (_ error,
) {
	s.Lock()
	defer s.Unlock()
	var (
		src, dst *sqlchainObject
		ok       bool
	)
	if dst, ok = s.dirty.databases[k]; !ok {
		if src, ok = s.readonly.databases[k]; !ok {
			return ErrDatabaseNotFound
		}
		dst = &sqlchainObject{}
		deepcopier.Copy(&src.SQLChainProfile).To(&dst.SQLChainProfile)
		s.dirty.databases[k] = dst
	}
	for _, v := range dst.Users {
		if v.Address == addr {
			v.Permission = perm
		}
	}
	return
}

func (s *metaState) nextNonce(addr proto.AccountAddress) (nonce pi.AccountNonce, err error) {
	s.Lock()
	defer s.Unlock()
	if e, ok := s.pool.getTxEntries(addr); ok {
		nonce = e.nextNonce()
		return
	}
	var (
		o      *accountObject
		loaded bool
	)
	if o, loaded = s.dirty.accounts[addr]; !loaded {
		if o, loaded = s.readonly.accounts[addr]; !loaded {
			err = ErrAccountNotFound
			return
		}
	}
	nonce = o.Account.NextNonce
	return
}

func (s *metaState) increaseNonce(addr proto.AccountAddress) (err error) {
	s.Lock()
	defer s.Unlock()
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[addr]; !ok {
		if src, ok = s.readonly.accounts[addr]; !ok {
			return ErrAccountNotFound
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[addr] = dst
	}
	dst.NextNonce++
	return
}

func (s *metaState) applyBilling(tx *pt.Billing) (err error) {
	for i, v := range tx.Receivers {
		// Create empty receiver account if not found
		s.loadOrStoreAccountObject(*v, &accountObject{Account: pt.Account{Address: *v}})

		if err = s.increaseAccountCovenantBalance(*v, tx.Fees[i]); err != nil {
			return
		}
		if err = s.increaseAccountStableBalance(*v, tx.Rewards[i]); err != nil {
			return
		}
	}
	return
}

func (s *metaState) applyTransaction(tx pi.Transaction) (err error) {
	switch t := tx.(type) {
	case *pt.Transfer:
		err = s.transferAccountStableBalance(t.Sender, t.Receiver, t.Amount)
	case *pt.Billing:
		err = s.applyBilling(t)
	case *pt.BaseAccount:
		err = s.storeBaseAccount(t.Address, &accountObject{Account: t.Account})
	case *pi.TransactionWrapper:
		// call again using unwrapped transaction
		err = s.applyTransaction(t.Unwrap())
	default:
		err = ErrUnknownTransactionType
	}
	return
}

// applyTransaction tries to apply t to the metaState and push t to the memory pool if and
// only if it can be applied correctly.
func (s *metaState) applyTransactionProcedure(t pi.Transaction) (_ func(*bolt.Tx) error) {
	var (
		err     error
		errPass = func(*bolt.Tx) error {
			return err
		}
	)

	log.WithField("tx", t).Debug("try applying transaction")

	// Static checks, which have no relation with metaState
	if err = t.Verify(); err != nil {
		return errPass
	}

	var (
		enc   *bytes.Buffer
		hash  = t.GetHash()
		addr  = t.GetAccountAddress()
		nonce = t.GetAccountNonce()
		ttype = t.GetTransactionType()
	)
	if enc, err = utils.EncodeMsgPack(t); err != nil {
		log.WithField("tx", t).WithError(err).Debug("encode failed on applying transaction")
		return errPass
	}

	// metaState-related checks will be performed within bolt.Tx to guarantee consistency
	return func(tx *bolt.Tx) (err error) {
		log.WithField("tx", t).Debug("processing transaction")

		// Check tx existense
		// TODO(leventeliu): maybe move outside?
		if s.pool.hasTx(t) {
			log.Debug("transaction already in pool, apply failed")
			return
		}
		// Check account nonce
		var nextNonce pi.AccountNonce
		if nextNonce, err = s.nextNonce(addr); err != nil {
			if t.GetTransactionType() != pi.TransactionTypeBaseAccount {
				return
			}
			// Consider the first nonce 0
			err = nil
		}
		if nextNonce != nonce {
			err = ErrInvalidAccountNonce
			log.WithFields(log.Fields{
				"actual":   nonce,
				"expected": nextNonce,
			}).WithError(err).Debug("nonce not match during transaction apply")
			return
		}
		// Try to put transaction before any state change, will be rolled back later
		// if transaction doesn't apply
		tb := tx.Bucket(metaBucket[:]).Bucket(metaTransactionBucket).Bucket(ttype.Bytes())
		if err = tb.Put(hash[:], enc.Bytes()); err != nil {
			log.WithError(err).Debug("store transaction to bucket failed")
			return
		}
		// Try to apply transaction to metaState
		if err = s.applyTransaction(t); err != nil {
			log.WithError(err).Debug("apply transaction failed")
			return
		}
		if err = s.increaseNonce(addr); err != nil {
			// FIXME(leventeliu): should not fail here.
			return
		}
		// Push to pool
		s.pool.addTx(t, nextNonce)
		return
	}
}

func (s *metaState) pullTxs() (txs []pi.Transaction) {
	s.Lock()
	defer s.Unlock()
	for _, v := range s.pool.entries {
		// TODO(leventeliu): check race condition.
		txs = append(txs, v.transactions...)
	}
	return
}
