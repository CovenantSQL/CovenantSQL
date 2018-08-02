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
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

var (
	genesisHash = hash.Hash{}
	uuidLen     = 32
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func generateRandomAccount() *Account {
	n := rand.Int31n(100) + 1
	sqlChains := generateRandomDatabaseIDs(n)
	roles := generateRandomBytes(n)

	h := generateRandomHash()

	txBillings := make([]*hash.Hash, n)

	for i := range txBillings {
		tmpHash := generateRandomHash()
		txBillings[i] = &tmpHash
	}

	account := &Account{
		Address:            proto.AccountAddress(h),
		StableCoinBalance:  rand.Uint64(),
		ThunderCoinBalance: rand.Uint64(),
		SQLChains:          sqlChains,
		Roles:              roles,
		Rating:             rand.Float64(),
		TxBillings:         txBillings,
	}

	return account
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

func generateRandomUint32s(n int32) []uint32 {
	s := make([]uint32, n)
	for i := range s {
		s[i] = (uint32)(rand.Int31())
	}
	return s
}

func generateRandomBlock(parent hash.Hash, isGenesis bool) (b *Block, err error) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

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
			Signee: pub,
		},
	}

	for i, n := 0, rand.Intn(10)+10; i < n; i++ {
		tb, err := generateRandomTxBilling()
		if err != nil {
			return nil, err
		}
		b.PushTx(tb)
	}

	err = b.PackAndSignBlock(priv)
	return
}

func generateRandomBillingRequestHeader() *BillingRequestHeader {
	return &BillingRequestHeader{
		DatabaseID:  *generateRandomDatabaseID(),
		BlockHash:   generateRandomHash(),
		BlockHeight: rand.Int31(),
		GasAmounts:  generateRandomGasAmount(peerNum),
	}
}

func generateRandomBillingRequest() (*BillingRequest, error) {
	reqHeader := generateRandomBillingRequestHeader()
	req := BillingRequest{
		Header: *reqHeader,
	}
	h, err := req.PackRequestHeader()
	if err != nil {
		return nil, err
	}

	signees := make([]*asymmetric.PublicKey, peerNum)
	signatures := make([]*asymmetric.Signature, peerNum)

	for i := range signees {
		// Generate key pair
		priv, pub, err := asymmetric.GenSecp256k1KeyPair()
		if err != nil {
			return nil, err
		}
		signees[i] = pub
		signatures[i], err = priv.Sign(h[:])
		if err != nil {
			return nil, err
		}
	}
	req.RequestHash = *h
	req.Signatures = signatures
	req.Signees = signees

	return &req, nil
}

func generateRandomBillingResponse() (*BillingResponse, error) {
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		return nil, err
	}
	h := generateRandomHash()
	sign, err := priv.Sign(h[:])
	if err != nil {
		return nil, err
	}
	resp := BillingResponse{
		AccountAddress: proto.AccountAddress(generateRandomHash()),
		RequestHash:    h,
		Signee:         pub,
		Signature:      sign,
	}
	return &resp, nil
}

func generateRandomTxContent() (*TxContent, error) {
	req, err := generateRandomBillingRequest()
	if err != nil {
		return nil, err
	}
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()
	sign, err := priv.Sign(req.RequestHash[:])
	if err != nil {
		return nil, err
	}
	resp := &BillingResponse{
		AccountAddress: proto.AccountAddress(generateRandomHash()),
		RequestHash:    req.RequestHash,
		Signee:         pub,
		Signature:      sign,
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

	tc := NewTxContent(rand.Uint32(), req, receivers, fees, rewards, resp)
	return tc, nil
}

func generateRandomTxBilling() (*TxBilling, error) {
	txContent, err := generateRandomTxContent()
	if err != nil {
		return nil, err
	}
	accountAddress := proto.AccountAddress(generateRandomHash())
	txHash := generateRandomHash()
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()
	sign, err := priv.Sign(txHash[:])
	if err != nil {
		return nil, err
	}
	blockHash := generateRandomHash()

	txBilling := NewTxBilling(txContent, TxTypeBilling, &accountAddress)
	txBilling.TxHash = &txHash
	txBilling.Signee = pub
	txBilling.Signature = sign
	txBilling.SignedBlock = &blockHash
	return txBilling, nil
}

func generateRandomGasAmount(n uint32) []*proto.AddrAndGas {
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
