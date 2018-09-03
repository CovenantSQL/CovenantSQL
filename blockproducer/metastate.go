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

	bt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/coreos/bbolt"
	"github.com/ulule/deepcopier"
)

// TODO(leventeliu): lock optimization.

type metaState struct {
	sync.RWMutex
	dirty, readonly *metaIndex
}

func newMetaState() *metaState {
	return &metaState{
		dirty:    newMetaIndex(),
		readonly: newMetaIndex(),
	}
}

func (s *metaState) loadAccountObject(k proto.AccountAddress) (o *accountObject, loaded bool) {
	s.RLock()
	defer s.RUnlock()
	if o, loaded = s.dirty.accounts[k]; loaded {
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
	if o, loaded = s.dirty.accounts[k]; loaded {
		return
	}
	if o, loaded = s.readonly.accounts[k]; loaded {
		return
	}
	s.dirty.accounts[k] = v
	return
}

func (s *metaState) loadSQLChainObject(k proto.DatabaseID) (o *sqlchainObject, loaded bool) {
	s.RLock()
	defer s.RUnlock()
	if o, loaded = s.dirty.databases[k]; loaded {
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
	if o, loaded = s.dirty.databases[k]; loaded {
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
				if err = ab.Put(v.Address[:], enc.Bytes()); err != nil {
					return
				}
			} else {
				// Delete object
				delete(s.readonly.accounts, k)
				if err = ab.Delete(v.Address[:]); err != nil {
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
		// Clean dirty map
		s.dirty = newMetaIndex()
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

func (s *metaState) increaseAccountcovenantBalance(k proto.AccountAddress, amount uint64) error {
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
		SQLChainProfile: bt.SQLChainProfile{
			ID:     id,
			Owner:  addr,
			Miners: make([]proto.AccountAddress, 0),
			Users: []*bt.SQLChainUser{
				&bt.SQLChainUser{
					Address:    addr,
					Permission: bt.Admin,
				},
			},
		},
	}
	return nil
}

func (s *metaState) addSQLChainUser(
	k proto.DatabaseID, addr proto.AccountAddress, perm bt.UserPermission) (_ error,
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
	dst.SQLChainProfile.Users = append(dst.SQLChainProfile.Users, &bt.SQLChainUser{
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
	k proto.DatabaseID, addr proto.AccountAddress, perm bt.UserPermission) (_ error,
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
