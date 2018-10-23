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
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	genesisHash = hash.Hash{}
	uuidLen     = 32
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func generateRandomSQLChainUser() *SQLChainUser {
	return &SQLChainUser{
		Address:    proto.AccountAddress(generateRandomHash()),
		Permission: UserPermission(rand.Int31n(int32(NumberOfUserPermission))),
	}
}

func generateRandomSQLChainUsers(n int) (users []*SQLChainUser) {
	users = make([]*SQLChainUser, n)
	for i := range users {
		users[i] = generateRandomSQLChainUser()
	}
	return
}

func generateRandomAccountAddresses(n int) (s []proto.AccountAddress) {
	s = make([]proto.AccountAddress, n)
	for i := range s {
		s[i] = proto.AccountAddress(generateRandomHash())
	}
	return
}

func generateRandomProfile() *SQLChainProfile {
	return &SQLChainProfile{
		ID:      *generateRandomDatabaseID(),
		Deposit: rand.Uint64(),
		Owner:   proto.AccountAddress(generateRandomHash()),
		Miners:  generateRandomAccountAddresses(rand.Intn(10) + 1),
		Users:   generateRandomSQLChainUsers(rand.Intn(10) + 1),
	}
}

func generateRandomAccount() *Account {
	return &Account{
		Address:             proto.AccountAddress(generateRandomHash()),
		StableCoinBalance:   rand.Uint64(),
		CovenantCoinBalance: rand.Uint64(),
		Rating:              rand.Float64(),
	}
}

func generateRandomBytes(n int32) []byte {
	s := make([]byte, n)
	for i := range s {
		s[i] = byte(rand.Int31n(2))
	}
	return s
}

func generateRandomHash() hash.Hash {
	h := hash.Hash{}
	rand.Read(h[:])
	return h
}

func generateRandomDatabaseID() *proto.DatabaseID {
	id := proto.DatabaseID(randStringBytes(uuidLen))
	return &id
}

func generateRandomDatabaseIDs(n int32) []proto.DatabaseID {
	s := make([]proto.DatabaseID, n)
	for i := range s {
		s[i] = proto.DatabaseID(randStringBytes(uuidLen))
	}
	return s
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func generateRandomBlock(parent hash.Hash, isGenesis bool) (b *Block, err error) {
	// Generate key pair
	priv, _, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &Block{
		SignedHeader: SignedHeader{
			Header: Header{
				Version:    0x01000000,
				Producer:   proto.AccountAddress(h),
				ParentHash: parent,
				Timestamp:  time.Now().UTC(),
			},
		},
	}

	for i, n := 0, rand.Intn(10)+10; i < n; i++ {
		tb, err := generateRandomBilling()
		if err != nil {
			return nil, err
		}
		b.Transactions = append(b.Transactions, tb)
	}

	err = b.PackAndSignBlock(priv)
	return
}

func generateRandomBillingRequestHeader() *BillingRequestHeader {
	return &BillingRequestHeader{
		DatabaseID: *generateRandomDatabaseID(),
		LowBlock:   generateRandomHash(),
		LowHeight:  rand.Int31(),
		HighBlock:  generateRandomHash(),
		HighHeight: rand.Int31(),
		GasAmounts: generateRandomGasAmount(peerNum),
	}
}

func generateRandomBillingRequest() (req *BillingRequest, err error) {
	reqHeader := generateRandomBillingRequestHeader()
	req = &BillingRequest{
		Header: *reqHeader,
	}
	if _, err = req.PackRequestHeader(); err != nil {
		return nil, err
	}

	for i := 0; i < peerNum; i++ {
		// Generate key pair
		var priv *asymmetric.PrivateKey

		if priv, _, err = asymmetric.GenSecp256k1KeyPair(); err != nil {
			return
		}

		if _, _, err = req.SignRequestHeader(priv, false); err != nil {
			return
		}
	}

	return
}

func generateRandomBillingHeader() (tc *BillingHeader, err error) {
	var req *BillingRequest
	if req, err = generateRandomBillingRequest(); err != nil {
		return
	}

	var priv *asymmetric.PrivateKey
	if priv, _, err = asymmetric.GenSecp256k1KeyPair(); err != nil {
		return
	}

	if _, _, err = req.SignRequestHeader(priv, false); err != nil {
		return
	}

	receivers := make([]*proto.AccountAddress, peerNum)
	fees := make([]uint64, peerNum)
	rewards := make([]uint64, peerNum)
	for i := range fees {
		h := generateRandomHash()
		accountAddress := proto.AccountAddress(h)
		receivers[i] = &accountAddress
		fees[i] = rand.Uint64()
		rewards[i] = rand.Uint64()
	}

	producer := proto.AccountAddress(generateRandomHash())
	tc = NewBillingHeader(pi.AccountNonce(rand.Uint32()), req, producer, receivers, fees, rewards)
	return tc, nil
}

func generateRandomBilling() (*Billing, error) {
	header, err := generateRandomBillingHeader()
	if err != nil {
		return nil, err
	}
	priv, _, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		return nil, err
	}
	txBilling := NewBilling(header)
	if err := txBilling.Sign(priv); err != nil {
		return nil, err
	}
	return txBilling, nil
}

func generateRandomGasAmount(n int) []*proto.AddrAndGas {
	gasAmount := make([]*proto.AddrAndGas, n)

	for i := range gasAmount {
		gasAmount[i] = &proto.AddrAndGas{
			AccountAddress: proto.AccountAddress(generateRandomHash()),
			RawNodeID:      proto.RawNodeID{Hash: generateRandomHash()},
			GasAmount:      rand.Uint64(),
		}
	}

	return gasAmount
}

func setup() {
	rand.Seed(time.Now().UnixNano())
	rand.Read(genesisHash[:])
	f, err := ioutil.TempFile("", "keystore")

	if err != nil {
		panic(err)
	}

	f.Close()

	if err = kms.InitPublicKeyStore(f.Name(), nil); err != nil {
		panic(err)
	}

	kms.Unittest = true

	if priv, pub, err := asymmetric.GenSecp256k1KeyPair(); err == nil {
		kms.SetLocalKeyPair(priv, pub)
	} else {
		panic(err)
	}

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}
