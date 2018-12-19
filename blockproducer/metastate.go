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
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	pt "github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/coreos/bbolt"
	"github.com/pkg/errors"
	"github.com/ulule/deepcopier"
)

var (
	sqlchainPeriod   uint64 = 60 * 24 * 30
	sqlchainGasPrice uint64 = 10
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
		b = o.TokenBalance[pt.Particle]
		return
	}
	if o, loaded = s.readonly.accounts[addr]; loaded {
		b = o.TokenBalance[pt.Particle]
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
		b = o.TokenBalance[pt.Wave]
		return
	}
	if o, loaded = s.readonly.accounts[addr]; loaded {
		b = o.TokenBalance[pt.Wave]
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
			cb = ao.TokenBalance[pt.Wave]
			sb = ao.TokenBalance[pt.Particle]
		)
		if err = safeAdd(&cb, &v.Account.TokenBalance[pt.Wave]); err != nil {
			return
		}
		if err = safeAdd(&sb, &v.Account.TokenBalance[pt.Particle]); err != nil {
			return
		}
		ao.TokenBalance[pt.Wave] = cb
		ao.TokenBalance[pt.Particle] = sb
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

func (s *metaState) loadProviderObject(k proto.AccountAddress) (o *providerObject, loaded bool) {
	s.RLock()
	defer s.RUnlock()
	if o, loaded = s.dirty.provider[k]; loaded {
		if o == nil {
			loaded = false
		}
		return
	}
	if o, loaded = s.readonly.provider[k]; loaded {
		return
	}
	return
}

func (s *metaState) loadOrStoreProviderObject(k proto.AccountAddress, v *providerObject) (o *providerObject, loaded bool) {
	s.Lock()
	defer s.Unlock()
	if o, loaded = s.dirty.provider[k]; loaded && o != nil {
		return
	}
	if o, loaded = s.readonly.provider[k]; loaded {
		return
	}
	s.dirty.provider[k] = v
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

func (s *metaState) deleteProviderObject(k proto.AccountAddress) {
	s.Lock()
	defer s.Unlock()
	// Use a nil pointer to mark a deletion, which will be later used by commit procedure.
	s.dirty.provider[k] = nil
}

func (s *metaState) commit() {
	s.Lock()
	defer s.Unlock()
	for k, v := range s.dirty.accounts {
		if v != nil {
			// New/update object
			s.readonly.accounts[k] = v
		} else {
			// Delete object
			delete(s.readonly.accounts, k)
		}
	}
	for k, v := range s.dirty.databases {
		if v != nil {
			// New/update object
			s.readonly.databases[k] = v
		} else {
			// Delete object
			delete(s.readonly.databases, k)
		}
	}
	for k, v := range s.dirty.provider {
		if v != nil {
			// New/update object
			s.readonly.provider[k] = v
		} else {
			// Delete object
			delete(s.readonly.provider, k)
		}
	}
	// Clean dirty map and tx pool
	s.dirty = newMetaIndex()
	s.pool = newTxPool()
	return
}

func (s *metaState) commitProcedure() (_ func(*bolt.Tx) error) {
	return func(tx *bolt.Tx) (err error) {
		var (
			enc *bytes.Buffer
			ab  = tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
			cb  = tx.Bucket(metaBucket[:]).Bucket(metaSQLChainIndexBucket)
			pb  = tx.Bucket(metaBucket[:]).Bucket(metaProviderIndexBucket)
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
		for k, v := range s.dirty.provider {
			if v != nil {
				// New/update object
				s.readonly.provider[k] = v
				if enc, err = utils.EncodeMsgPack(v.ProviderProfile); err != nil {
					return
				}
				if err = pb.Put(k[:], enc.Bytes()); err != nil {
					return
				}
			} else {
				// Delete object
				delete(s.readonly.provider, k)
				if err = pb.Delete(k[:]); err != nil {
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
			pb  = tx.Bucket(metaBucket[:]).Bucket(metaProviderIndexBucket)
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
		for k, v := range cm.dirty.provider {
			if v != nil {
				// New/update object
				cm.readonly.provider[k] = v
				if enc, err = utils.EncodeMsgPack(v.Provider); err != nil {
					return
				}
				if err = pb.Put(k[:], enc.Bytes()); err != nil {
					return
				}
			} else {
				// Delete object
				delete(cm.readonly.provider, k)
				if err = pb.Delete(k[:]); err != nil {
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
			pb = tx.Bucket(metaBucket[:]).Bucket(metaProviderIndexBucket)
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
		if err = pb.ForEach(func(k, v []byte) (err error) {
			ao := &providerObject{}
			if err = utils.DecodeMsgPack(v, &ao.Provider); err != nil {
				return
			}
			s.readonly.provider[ao.ProviderProfile.Provider] = ao
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

func (s *metaState) increaseAccountToken(k proto.AccountAddress, amount uint64, tokenType pt.TokenType) error {
	s.Lock()
	defer s.Unlock()
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			err := errors.Wrap(ErrAccountNotFound, "increase stable balance fail")
			return err
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeAdd(&dst.Account.TokenBalance[tokenType], &amount)
}

func (s *metaState) decreaseAccountToken(k proto.AccountAddress, amount uint64, tokenType pt.TokenType) error {
	s.Lock()
	defer s.Unlock()
	var (
		src, dst *accountObject
		ok       bool
	)
	if dst, ok = s.dirty.accounts[k]; !ok {
		if src, ok = s.readonly.accounts[k]; !ok {
			err := errors.Wrap(ErrAccountNotFound, "increase stable balance fail")
			return err
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeSub(&dst.Account.TokenBalance[tokenType], &amount)
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
			err := errors.Wrap(ErrAccountNotFound, "increase stable balance fail")
			return err
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeAdd(&dst.Account.TokenBalance[pt.Particle], &amount)
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
	return safeSub(&dst.Account.TokenBalance[pt.Particle], &amount)
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
		sb = so.TokenBalance[pt.Particle]
		rb = ro.TokenBalance[pt.Particle]
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
	so.TokenBalance[pt.Particle] = sb
	ro.TokenBalance[pt.Particle] = rb
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
			err := errors.Wrap(ErrAccountNotFound, "increase covenant balance fail")
			return err
		}
		dst = &accountObject{}
		deepcopier.Copy(&src.Account).To(&dst.Account)
		s.dirty.accounts[k] = dst
	}
	return safeAdd(&dst.Account.TokenBalance[pt.Wave], &amount)
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
	return safeSub(&dst.Account.TokenBalance[pt.Wave], &amount)
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
			Miners: make([]*pt.MinerInfo, 0),
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
			log.WithFields(log.Fields{
				"addr": addr.String(),
			}).WithError(err).Error("unexpected error")
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

func (s *metaState) updateProviderList(tx *pt.ProvideService) (err error) {
	sender, err := crypto.PubKeyHash(tx.Signee)
	if err != nil {
		err = errors.Wrap(err, "updateProviderList failed")
		return
	}

	// deposit
	var (
		minDeposit = conf.GConf.MinProviderDeposit
	)
	if err = s.decreaseAccountStableBalance(sender, minDeposit); err != nil {
		return
	}

	pp := pt.ProviderProfile{
		Provider:      sender,
		Space:         tx.Space,
		Memory:        tx.Memory,
		LoadAvgPerCPU: tx.LoadAvgPerCPU,
		TargetUser:    tx.TargetUser,
		Deposit:       minDeposit,
		GasPrice:      tx.GasPrice,
		NodeID:        tx.NodeID,
	}
	s.loadOrStoreProviderObject(sender, &providerObject{ProviderProfile: pp})
	return
}

func (s *metaState) matchProvidersWithUser(tx *pt.CreateDatabase) (err error) {
	sender, err := crypto.PubKeyHash(tx.Signee)
	if err != nil {
		err = errors.Wrap(err, "matchProviders failed")
		return
	}
	if sender != tx.Owner {
		err = errors.Wrapf(ErrInvalidSender, "match failed with real sender: %s, sender: %s",
			sender.String(), tx.Owner.String())
		return
	}

	if tx.GasPrice <= 0 {
		err = ErrInvalidGasPrice
		return
	}

	var (
		minAdvancePayment = uint64(tx.GasPrice) * uint64(conf.GConf.QPS) *
			uint64(conf.GConf.Period) * uint64(len(tx.ResourceMeta.TargetMiners))
	)
	if tx.AdvancePayment < minAdvancePayment {
		err = ErrInsufficientAdvancePayment
		return
	}

	var (
		miners [len(tx.ResourceMeta.TargetMiners)]*pt.MinerInfo
	)

	for i := range tx.ResourceMeta.TargetMiners {
		if po, loaded := s.loadProviderObject(tx.ResourceMeta.TargetMiners[i]); !loaded {
			log.WithFields(log.Fields{
				"miner_addr": tx.ResourceMeta.TargetMiners[i].String(),
				"user_addr":  sender.String(),
			}).Error(err)
			err = ErrNoSuchMiner
			break
		} else {
			if po.TargetUser != sender {
				log.WithFields(log.Fields{
					"miner_addr": tx.ResourceMeta.TargetMiners[i].String(),
					"user_addr":  sender.String(),
				}).Error(ErrMinerUserNotMatch)
				err = ErrMinerUserNotMatch
				break
			}
			if po.GasPrice > tx.GasPrice {
				err = ErrGasPriceMismatch
				break
			}
			miners[i] = &pt.MinerInfo{
				Address: po.Provider,
				NodeID:  po.NodeID,
				Deposit: po.Deposit,
			}
		}
	}
	if err != nil {
		return
	}

	// generate new sqlchain id and address
	dbID := proto.FromAccountAndNonce(tx.Owner, uint32(tx.Nonce))
	dbAddr, err := dbID.AccountAddress()
	if err != nil {
		err = errors.Wrapf(err, "unexpected error when convert dbid: %v", dbID)
		return
	}
	// generate userinfo
	var all = minAdvancePayment
	err = safeAdd(&all, &tx.AdvancePayment)
	if err != nil {
		return
	}
	err = s.decreaseAccountToken(sender, all, tx.TokenType)
	if err != nil {
		return
	}
	users := make([]*pt.SQLChainUser, 1)
	users[0] = &pt.SQLChainUser{
		Address:        sender,
		Permission:     pt.Admin,
		Deposit:        minAdvancePayment,
		AdvancePayment: tx.AdvancePayment,
	}
	// generate genesis block
	gb, err := s.generateGenesisBlock(*dbID, tx.ResourceMeta)
	if err != nil {
		log.WithFields(log.Fields{
			"dbID":         dbID,
			"resourceMeta": tx.ResourceMeta,
		}).WithError(err).Error("unexpected error")
		return err
	}

	// create sqlchain
	sp := &pt.SQLChainProfile{
		ID:        *dbID,
		Address:   dbAddr,
		Period:    sqlchainPeriod,
		GasPrice:  sqlchainGasPrice,
		TokenType: pt.Particle,
		Owner:     sender,
		Users:     users,
		Genesis:   gb,
		Miners:    miners[:],
	}

	if _, loaded := s.loadSQLChainObject(*dbID); loaded {
		err = errors.Wrapf(ErrDatabaseExists, "database exists: %s", string(*dbID))
		return
	}
	s.loadOrStoreAccountObject(dbAddr, &accountObject{
		Account: pt.Account{Address: dbAddr},
	})
	s.loadOrStoreSQLChainObject(*dbID, &sqlchainObject{SQLChainProfile: *sp})
	for _, miner := range tx.ResourceMeta.TargetMiners {
		s.deleteProviderObject(miner)
	}
	return
}

func (s *metaState) updatePermission(tx *pt.UpdatePermission) (err error) {
	sender, err := crypto.PubKeyHash(tx.Signee)
	if err != nil {
		log.WithFields(log.Fields{
			"tx": tx.Hash().String(),
		}).WithError(err).Error("unexpected err")
		return
	}
	if sender == tx.TargetUser {
		err = errors.Wrap(ErrInvalidSender, "user cannot update its permission by itself")
		return
	}
	so, loaded := s.loadSQLChainObject(tx.TargetSQLChain.DatabaseID())
	if !loaded {
		log.WithFields(log.Fields{
			"dbID": tx.TargetSQLChain.DatabaseID(),
		}).WithError(ErrDatabaseNotFound).Error("unexpected error in updatePermission")
		return ErrDatabaseNotFound
	}
	if tx.Permission >= pt.NumberOfUserPermission {
		log.WithFields(log.Fields{
			"permission": tx.Permission,
			"dbID":       tx.TargetSQLChain.DatabaseID(),
		}).WithError(ErrInvalidPermission).Error("unexpected error in updatePermission")
		return ErrInvalidPermission
	}

	// check whether sender is admin and find targetUser
	isAdmin := false
	targetUserIndex := -1
	for i, u := range so.Users {
		isAdmin = isAdmin || (sender == u.Address && u.Permission == pt.Admin)
		if tx.TargetUser == u.Address {
			targetUserIndex = i
		}
	}

	if !isAdmin {
		log.WithFields(log.Fields{
			"sender": sender,
			"dbID":   tx.TargetSQLChain,
		}).WithError(ErrAccountPermissionDeny).Error("unexpected error in updatePermission")
		return ErrAccountPermissionDeny
	}

	// update targetUser's permission
	if targetUserIndex == -1 {
		u := pt.SQLChainUser{
			Address:    tx.TargetUser,
			Permission: tx.Permission,
			Status:     pt.Normal,
		}
		so.Users = append(so.Users, &u)
	} else {
		so.Users[targetUserIndex].Permission = tx.Permission
	}
	return
}

func (s *metaState) updateKeys(tx *pt.IssueKeys) (err error) {
	sender := tx.GetAccountAddress()
	so, loaded := s.loadSQLChainObject(tx.TargetSQLChain.DatabaseID())
	if !loaded {
		log.WithFields(log.Fields{
			"dbID": tx.TargetSQLChain.DatabaseID(),
		}).WithError(ErrDatabaseNotFound).Error("unexpected error in updateKeys")
		return ErrDatabaseNotFound
	}

	// check sender's permission
	if so.Owner != sender {
		log.WithFields(log.Fields{
			"sender": sender,
			"dbID":   tx.TargetSQLChain,
		}).WithError(ErrAccountPermissionDeny).Error("unexpected error in updateKeys")
		return ErrAccountPermissionDeny
	}
	isAdmin := false
	for _, user := range so.Users {
		if sender == user.Address && user.Permission == pt.Admin {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		log.WithFields(log.Fields{
			"sender": sender,
			"dbID":   tx.TargetSQLChain,
		}).WithError(ErrAccountPermissionDeny).Error("unexpected error in updateKeys")
		return ErrAccountPermissionDeny
	}

	// update miner's key
	keyMap := make(map[proto.AccountAddress]string)
	for i := range tx.MinerKeys {
		keyMap[tx.MinerKeys[i].Miner] = tx.MinerKeys[i].EncryptionKey
	}
	for _, miner := range so.Miners {
		if key, ok := keyMap[miner.Address]; ok {
			miner.EncryptionKey = key
		}
	}
	return
}

func (s *metaState) updateBilling(tx *pt.UpdateBilling) (err error) {
	sqlchainObj, loaded := s.loadSQLChainObject(tx.Receiver.DatabaseID())
	if !loaded {
		err = errors.Wrap(ErrDatabaseNotFound, "update billing failed")
		return
	}

	sqlchainObj.Lock()
	defer sqlchainObj.Unlock()

	if sqlchainObj.GasPrice == 0 {
		return
	}

	// pending income to income
	for _, miner := range sqlchainObj.Miners {
		miner.ReceivedIncome += miner.PendingIncome
	}

	var (
		costMap = make(map[proto.AccountAddress]uint64)
		userMap = make(map[proto.AccountAddress]map[proto.AccountAddress]uint64)
	)
	for _, userCost := range tx.Users {
		costMap[userCost.User] = userCost.Cost
		if _, ok := userMap[userCost.User]; !ok {
			userMap[userCost.User] = make(map[proto.AccountAddress]uint64)
		}
		for _, minerIncome := range userCost.Miners {
			userMap[userCost.User][minerIncome.Miner] += minerIncome.Income
		}
	}
	for _, user := range sqlchainObj.Users {
		if user.AdvancePayment >= costMap[user.Address] * sqlchainObj.GasPrice {
			user.AdvancePayment -= costMap[user.Address] * sqlchainObj.GasPrice
			for _, miner := range sqlchainObj.Miners {
				miner.PendingIncome += userMap[user.Address][miner.Address] * sqlchainObj.GasPrice
			}
		} else {
			rate := 1 - float64(user.AdvancePayment) / float64(costMap[user.Address] * sqlchainObj.GasPrice)
			user.AdvancePayment = 0
			for _, miner := range sqlchainObj.Miners {
				income := userMap[user.Address][miner.Address] * sqlchainObj.GasPrice
				minerIncome := uint64(float64(income) * rate)
				miner.PendingIncome += minerIncome
				for i := range miner.UserArrears {
					miner.UserArrears[i].Arrears += (income - minerIncome)
				}
			}
		}
	}
}

func (s *metaState) applyTransaction(tx pi.Transaction) (err error) {
	switch t := tx.(type) {
	case *pt.Transfer:
		realSender, err := crypto.PubKeyHash(t.Signee)
		if err != nil {
			err = errors.Wrap(err, "applyTx failed")
			return err
		}
		if realSender != t.Sender {
			err = errors.Wrapf(ErrInvalidSender,
				"applyTx failed: real sender %s, sender %s", realSender.String(), t.Sender.String())
			// TODO(lambda): update test cases and return err
			log.Debug(err)
		}
		err = s.transferAccountStableBalance(t.Sender, t.Receiver, t.Amount)
	case *pt.Billing:
		err = s.applyBilling(t)
	case *pt.BaseAccount:
		err = s.storeBaseAccount(t.Address, &accountObject{Account: t.Account})
	case *pt.ProvideService:
		err = s.updateProviderList(t)
	case *pt.CreateDatabase:
		err = s.matchProvidersWithUser(t)
	case *pt.UpdatePermission:
		err = s.updatePermission(t)
	case *pt.IssueKeys:
		err = s.updateKeys(t)
	case *pt.UpdateBilling:
		err = s.updateBilling(t)
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
		hash  = t.Hash()
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

func (s *metaState) generateGenesisBlock(dbID proto.DatabaseID, resourceMeta pt.ResourceMeta) (genesisBlock *pt.Block, err error) {
	// TODO(xq262144): following is stub code, real logic should be implemented in the future
	emptyHash := hash.Hash{}

	var privKey *asymmetric.PrivateKey
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	genesisBlock = &pt.Block{
		SignedHeader: pt.SignedHeader{
			Header: pt.Header{
				Version:     0x01000000,
				Producer:    nodeID,
				GenesisHash: emptyHash,
				ParentHash:  emptyHash,
				Timestamp:   time.Now().UTC(),
			},
		},
	}

	err = genesisBlock.PackAndSignBlock(privKey)

	return
}

func (s *metaState) apply(t pi.Transaction) (err error) {
	// NOTE(leventeliu): bypass pool in this method.
	var (
		addr  = t.GetAccountAddress()
		nonce = t.GetAccountNonce()
	)
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
	// Try to apply transaction to metaState
	if err = s.applyTransaction(t); err != nil {
		log.WithError(err).Debug("apply transaction failed")
		return
	}
	if err = s.increaseNonce(addr); err != nil {
		return
	}
	return
}

func (s *metaState) makeCopy() *metaState {
	return &metaState{
		dirty:    newMetaIndex(),
		readonly: s.readonly.deepCopy(),
		pool:     newTxPool(),
	}
}
