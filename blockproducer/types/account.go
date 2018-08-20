/*
 * Copyright 2018 The ThunderDB Authors.
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

package types

import (
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"

	"gitlab.com/thunderdb/ThunderDB/proto"
)

//go:generate hsp

// SQL Chain role type
const (
	Miner byte = iota
	Customer
)

// Account store its balance, and other mate data
type Account struct {
	Address            proto.AccountAddress
	StableCoinBalance  uint64
	ThunderCoinBalance uint64
	SQLChains          []proto.DatabaseID
	Roles              []byte
	Rating             float64
	TxBillings         []*hash.Hash
}

// Account4test store its balance, and other mate data
type Account4test struct {
	Address1            proto.AccountAddress
	StableCoinBalance1  uint64
	ThunderCoinBalance1 uint64
	SQLChains1          []proto.DatabaseID
	Roles1              []byte
	Rating1             float64
	TxBillings1         []*hash.Hash
}

// AppendSQLChainAndRole add the sql chain include the account and its related role
func (a *Account) AppendSQLChainAndRole(sqlChain *proto.DatabaseID, role byte) {
	a.SQLChains = append(a.SQLChains, *sqlChain)
	a.Roles = append(a.Roles, role)
}
