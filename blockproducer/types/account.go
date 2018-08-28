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
	"sync"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

//go:generate hsp

// SQLChainRole defines roles of account in a SQLChain.
type SQLChainRole byte

// SQL Chain role type.
const (
	Miner SQLChainRole = iota
	Customer
	NumberOfRoles
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

// SQLChainProfile defines a SQLChainProfile related to an account.
type SQLChainProfile struct {
	ID      proto.DatabaseID
	Role    SQLChainRole
	Deposit uint64
}

// Account store its balance, and other mate data.
type Account struct {
	mu                 sync.Mutex
	Address            proto.AccountAddress
	StableCoinBalance  uint64
	ThunderCoinBalance uint64
	Rating             float64
	Profiles           []*SQLChainProfile
	TxBillings         []*hash.Hash
}

// Serialize implements Serializer.
func (a *Account) Serialize() (enc []byte, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	buf, err := utils.EncodeMsgPack(a)
	if err != nil {
		return
	}
	enc = buf.Bytes()
	return
}

// Deserialize implements Deserializer.
func (a *Account) Deserialize(enc []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return utils.DecodeMsgPack(enc, a)
}

// GetStableCoinBalance returns the stable coin balance of account.
func (a *Account) GetStableCoinBalance() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.StableCoinBalance
}

// GetThunderCoinBalance returns the thunder coin balance of account.
func (a *Account) GetThunderCoinBalance() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ThunderCoinBalance
}

// IncreaseAccountStableBalance increases account stable balance by amount.
func (a *Account) IncreaseAccountStableBalance(amount uint64) (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return safeAdd(&a.StableCoinBalance, &amount)
}

// DecreaseAccountStableBalance decreases account stable balance by amount.
func (a *Account) DecreaseAccountStableBalance(amount uint64) (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return safeSub(&a.StableCoinBalance, &amount)
}

// IncreaseAccountThunderBalance increases account thunder balance by amount.
func (a *Account) IncreaseAccountThunderBalance(amount uint64) (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return safeAdd(&a.ThunderCoinBalance, &amount)
}

// DecreaseAccountThunderBalance decreases account thunder balance by amount.
func (a *Account) DecreaseAccountThunderBalance(amount uint64) (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return safeSub(&a.ThunderCoinBalance, &amount)
}

// SendDeposit sends deposit of amount from account balance to SQLChain with id.
func (a *Account) SendDeposit(id proto.DatabaseID, role SQLChainRole, amount uint64) (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err = safeSub(&a.StableCoinBalance, &amount); err != nil {
		return
	}
	for _, v := range a.Profiles {
		if v.ID == id && v.Role == role {
			if err = safeAdd(&v.Deposit, &amount); err != nil {
				a.StableCoinBalance += amount
			}
			return
		}
	}
	a.Profiles = append(a.Profiles, &SQLChainProfile{
		ID:      id,
		Role:    role,
		Deposit: amount,
	})
	return
}

// WithdrawDeposit withdraws deposit from SQLChain with id to account balance.
func (a *Account) WithdrawDeposit(id proto.DatabaseID, role SQLChainRole) (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, v := range a.Profiles {
		if v.ID == id && v.Role == role {
			if err = safeAdd(&a.StableCoinBalance, &v.Deposit); err != nil {
				return
			}
			a.Profiles[i] = a.Profiles[len(a.Profiles)-1]
			a.Profiles = a.Profiles[:len(a.Profiles)-1]
			break
		}
	}
	return
}
