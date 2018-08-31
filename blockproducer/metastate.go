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

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/coreos/bbolt"
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
	// Use a nil pointer to mark a deletion, which will be later used by commit process.
	s.dirty.accounts[k] = nil
}

func (s *metaState) deleteSQLChainObject(k proto.DatabaseID) {
	s.Lock()
	defer s.Unlock()
	// Use a nil pointer to mark a deletion, which will be later used by commit process.
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

func (s *metaState) clean() {
	s.Lock()
	defer s.Unlock()
	s.dirty = newMetaIndex()
}
