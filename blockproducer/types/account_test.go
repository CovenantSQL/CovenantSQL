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
	"math"
	"reflect"
	"sync"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/proto"
)

func TestAccountSerializeDeserializer(t *testing.T) {
	account := generateRandomAccount()
	enc, err := account.Serialize()
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	dec := &Account{}
	if err = dec.Deserialize(enc); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	if !reflect.DeepEqual(account, dec) {
		t.Fatalf("Values don't match:\n\tv1 = %v\n\tv2 = %v", account, dec)
	}
}

func TestAccountBalanceOverflow(t *testing.T) {
	account := &Account{}
	db1 := proto.DatabaseID("database#1")

	account.StableCoinBalance = 0
	if err := account.DecreaseAccountStableBalance(1); err != ErrInsufficientBalance {
		t.Fatalf("Unexpected error: %v", err)
	}
	if balance := account.GetStableCoinBalance(); balance != 0 {
		t.Fatalf("Unexpected balance: %d", balance)
	}

	account.StableCoinBalance = math.MaxUint64
	if err := account.IncreaseAccountStableBalance(1); err != ErrBalanceOverflow {
		t.Fatalf("Unexpected error: %v", err)
	}
	if balance := account.GetStableCoinBalance(); balance != math.MaxUint64 {
		t.Fatalf("Unexpected balance: %d", balance)
	}

	account.CovenantCoinBalance = 0
	if err := account.DecreaseAccountCovenantBalance(1); err != ErrInsufficientBalance {
		t.Fatalf("Unexpected error: %v", err)
	}
	if balance := account.GetCovenantCoinBalance(); balance != 0 {
		t.Fatalf("Unexpected balance: %d", balance)
	}

	account.CovenantCoinBalance = math.MaxUint64
	if err := account.IncreaseAccountCovenantBalance(1); err != ErrBalanceOverflow {
		t.Fatalf("Unexpected error: %v", err)
	}
	if balance := account.GetCovenantCoinBalance(); balance != math.MaxUint64 {
		t.Fatalf("Unexpected balance: %d", balance)
	}

	account.StableCoinBalance = 0
	if err := account.SendDeposit(db1, Miner, 1); err != ErrInsufficientBalance {
		t.Fatalf("Unexpected error: %v", err)
	}

	account.StableCoinBalance = 1
	if err := account.SendDeposit(db1, Miner, 1); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	account.StableCoinBalance = math.MaxUint64
	if err := account.SendDeposit(db1, Miner, math.MaxUint64); err != ErrBalanceOverflow {
		t.Fatalf("Unexpected error: %v", err)
	}

	account.StableCoinBalance = math.MaxUint64
	if err := account.WithdrawDeposit(db1, Miner); err != ErrBalanceOverflow {
		t.Fatalf("Unexpected error: %v", err)
	}

	account.StableCoinBalance = 0
	if err := account.WithdrawDeposit(db1, Miner); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
}

func (a *Account) testIncreaseAccountCovenantBalance(t *testing.T, wg *sync.WaitGroup, amount uint64) {
	defer wg.Done()
	if err := a.IncreaseAccountCovenantBalance(amount); err != nil {
		t.Errorf("Error occurred: %v", err)
	}
}

func (a *Account) testDecreaseAccountCovenantBalance(t *testing.T, wg *sync.WaitGroup, amount uint64) {
	defer wg.Done()
	if err := a.DecreaseAccountCovenantBalance(amount); err != nil {
		t.Errorf("Error occurred: %v", err)
	}
}

func (a *Account) testIncreaseAccountStableBalance(t *testing.T, wg *sync.WaitGroup, amount uint64) {
	defer wg.Done()
	if err := a.IncreaseAccountStableBalance(amount); err != nil {
		t.Errorf("Error occurred: %v", err)
	}
}

func (a *Account) testDecreaseAccountStableBalance(t *testing.T, wg *sync.WaitGroup, amount uint64) {
	defer wg.Done()
	if err := a.DecreaseAccountStableBalance(amount); err != nil {
		t.Errorf("Error occurred: %v", err)
	}
}

func TestAccountCovenantBalance(t *testing.T) {
	wg := &sync.WaitGroup{}
	account := &Account{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go account.testIncreaseAccountCovenantBalance(t, wg, 1)
	}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go account.testDecreaseAccountCovenantBalance(t, wg, 1)
	}
	wg.Wait()
	if account.CovenantCoinBalance != 0 {
		t.Fatalf("Unexpected result: %d", account.CovenantCoinBalance)
	}
}

func TestAccountStableBalance(t *testing.T) {
	wg := &sync.WaitGroup{}
	account := &Account{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go account.testIncreaseAccountStableBalance(t, wg, 1)
	}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go account.testDecreaseAccountStableBalance(t, wg, 1)
	}
	wg.Wait()
	if balance := account.GetStableCoinBalance(); balance != 0 {
		t.Fatalf("Unexpected balance: %d", balance)
	}
}
