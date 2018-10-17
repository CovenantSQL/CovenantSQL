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

package types

import (
	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// SQLChainRole defines roles of account in a SQLChain.
type SQLChainRole byte

const (
	// Miner defines the miner role as a SQLChain user.
	Miner SQLChainRole = iota
	// Customer defines the customer role as a SQLChain user.
	Customer
	// NumberOfRoles defines the SQLChain roles number.
	NumberOfRoles
)

// UserPermission defines permissions of a SQLChain user.
type UserPermission int32

const (
	// Admin defines the admin user permission.
	Admin UserPermission = iota
	// Read defines the reader user permission.
	Read
	// ReadWrite defines the reader/writer user permission.
	ReadWrite
	// NumberOfUserPermission defines the user permission number.
	NumberOfUserPermission
)

// Token defines token's number.
const SupportTokenNumber int32 = 3
var Token = [SupportTokenNumber]string{
	"Ether",
	"EOS",
	"Bitcoin",
}

type TokenType int32

const (
	// Ether defines Ethereum.
	Ether TokenType = iota
	// EOS defines EOS.
	EOS
	// Bitcoin defines Bitcoin.
	Bitcoin
)

// String returns token's symbol.
func (t TokenType) String() string {
	switch t {
	case Ether:
		return "Ether"
	case EOS:
		return "EOS"
	case Bitcoin:
		return "Bitcoin"
	default:
		return "Unknown"
	}
}

// FromString returns token's number.
func FromString(t string) TokenType {
	switch t {
	case "Ether":
		return Ether
	case "EOS":
		return EOS
	case "Bitcoin":
		return Bitcoin
	default:
		return -1
	}
}

// Listed returns if the token is listed in list.
func (t TokenType) Listed() bool {
	return t >= 0 && int32(t) < SupportTokenNumber
}

// SQLChainUser defines a SQLChain user.
type SQLChainUser struct {
	Address    proto.AccountAddress
	Permission UserPermission
}

// SQLChainProfile defines a SQLChainProfile related to an account.
type SQLChainProfile struct {
	ID      proto.DatabaseID
	Deposit uint64
	Owner   proto.AccountAddress
	Miners  []proto.AccountAddress
	Users   []*SQLChainUser
}

// TokenList stores other tokens except StableCoin and CovenantCoin
type TokenList struct {
	Names [SupportTokenNumber]string
	Balances [SupportTokenNumber]*cpuminer.Uint256
}

func NewTokenList() *TokenList {
	tl := &TokenList{}
	for i, s := range Token {
		tl.Names[i] = s
		tl.Balances[i] = cpuminer.Zero()
	}
	return tl
}

// Account stores its balance, and other mate data.
type Account struct {
	Address             proto.AccountAddress
	StableCoinBalance   uint64
	CovenantCoinBalance uint64
	TokenList 			*TokenList
	Rating              float64
	NextNonce           pi.AccountNonce
}

