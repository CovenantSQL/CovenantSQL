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
	"sync"

	bt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

type accountIndex struct {
	index sync.Map
}

func newAccountIndex() *accountIndex {
	return &accountIndex{}
}

func (i *accountIndex) storeAccount(account *bt.Account) {
	i.index.Store(account.Address, account)
}

func (i *accountIndex) loadAccount(addr proto.AccountAddress) (account *bt.Account, ok bool) {
	var value interface{}
	if value, ok = i.index.Load(addr); ok {
		account = value.(*bt.Account)
	}
	return
}

func (i *accountIndex) increaseAccountStableBalance(addr proto.AccountAddress, amount uint64) (ok bool, err error) {
	var value interface{}
	if value, ok = i.index.Load(addr); ok {
		err = value.(*bt.Account).IncreaseAccountStableBalance(amount)
	}
	return
}

func (i *accountIndex) decreaseAccountStableBalance(addr proto.AccountAddress, amount uint64) (ok bool, err error) {
	var value interface{}
	if value, ok = i.index.Load(addr); ok {
		err = value.(*bt.Account).DecreaseAccountStableBalance(amount)
	}
	return
}

func (i *accountIndex) increaseAccountThunderBalance(addr proto.AccountAddress, amount uint64) (ok bool, err error) {
	var value interface{}
	if value, ok = i.index.Load(addr); ok {
		err = value.(*bt.Account).IncreaseAccountThunderBalance(amount)
	}
	return
}

func (i *accountIndex) decreaseAccountThunderBalance(addr proto.AccountAddress, amount uint64) (ok bool, err error) {
	var value interface{}
	if value, ok = i.index.Load(addr); ok {
		err = value.(*bt.Account).DecreaseAccountThunderBalance(amount)
	}
	return
}

func (i *accountIndex) sendDeposit(addr proto.AccountAddress, id proto.DatabaseID, role bt.SQLChainRole, amount uint64) (ok bool, err error) {
	var value interface{}
	if value, ok = i.index.Load(addr); ok {
		err = value.(*bt.Account).SendDeposit(id, role, amount)
	}
	return
}

func (i *accountIndex) withdrawDeposit(addr proto.AccountAddress, id proto.DatabaseID, role bt.SQLChainRole) (ok bool, err error) {
	var value interface{}
	if value, ok = i.index.Load(addr); ok {
		err = value.(*bt.Account).WithdrawDeposit(id, role)
	}
	return
}
