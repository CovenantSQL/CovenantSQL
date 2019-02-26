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
	"github.com/mohae/deepcopy"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
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

type metaIndex struct {
	accounts  map[proto.AccountAddress]*types.Account
	databases map[proto.DatabaseID]*types.SQLChainProfile
	provider  map[proto.AccountAddress]*types.ProviderProfile
}

func newMetaIndex() *metaIndex {
	return &metaIndex{
		accounts:  make(map[proto.AccountAddress]*types.Account),
		databases: make(map[proto.DatabaseID]*types.SQLChainProfile),
		provider:  make(map[proto.AccountAddress]*types.ProviderProfile),
	}
}

func (i *metaIndex) deepCopy() (cpy *metaIndex) {
	cpy = newMetaIndex()
	for k, v := range i.accounts {
		cpy.accounts[k] = deepcopy.Copy(v).(*types.Account)
	}
	for k, v := range i.databases {
		cpy.databases[k] = deepcopy.Copy(v).(*types.SQLChainProfile)
	}
	for k, v := range i.provider {
		cpy.provider[k] = deepcopy.Copy(v).(*types.ProviderProfile)
	}
	return
}
