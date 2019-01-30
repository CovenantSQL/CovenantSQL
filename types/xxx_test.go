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
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	uuidLen           = 32
	genesisHash       = hash.Hash{}
	testingPrivateKey *asymmetric.PrivateKey
	testingPublicKey  *asymmetric.PublicKey
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func generateRandomHash() hash.Hash {
	h := hash.Hash{}
	rand.Read(h[:])
	return h
}

func generateRandomDatabaseID() proto.DatabaseID {
	return proto.DatabaseID(randStringBytes(uuidLen))
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]

	}
	return string(b)
}

func generateRandomBlock(parent hash.Hash, isGenesis bool) (b *BPBlock, err error) {
	// Generate key pair
	priv, _, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &BPBlock{
		SignedHeader: BPSignedHeader{
			BPHeader: BPHeader{
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
		DatabaseID: generateRandomDatabaseID(),
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

func randBytes(n int) (b []byte) {
	b = make([]byte, n)
	rand.Read(b)
	return
}

func buildQuery(query string, args ...interface{}) Query {
	var nargs = make([]NamedArg, len(args))
	for i := range args {
		nargs[i] = NamedArg{
			Name:  "",
			Value: args[i],
		}
	}
	return Query{
		Pattern: query,
		Args:    nargs,
	}
}

func buildRequest(qt QueryType, qs []Query) (r *Request) {
	var (
		id  proto.NodeID
		err error
	)
	if id, err = kms.GetLocalNodeID(); err != nil {
		id = proto.NodeID("00000000000000000000000000000000")
	}
	r = &Request{
		Header: SignedRequestHeader{
			RequestHeader: RequestHeader{
				NodeID:    id,
				Timestamp: time.Now().UTC(),
				QueryType: qt,
			},
		},
		Payload: RequestPayload{Queries: qs},
	}
	if err = r.Sign(testingPrivateKey); err != nil {
		panic(err)
	}
	return
}

func buildResponse(header *SignedRequestHeader, cols []string, types []string, rows []ResponseRow) (r *Response) {
	var (
		id  proto.NodeID
		err error
	)
	if id, err = kms.GetLocalNodeID(); err != nil {
		id = proto.NodeID("00000000000000000000000000000000")
	}
	r = &Response{
		Header: SignedResponseHeader{
			ResponseHeader: ResponseHeader{
				Request:      header.RequestHeader,
				RequestHash:  header.Hash(),
				NodeID:       id,
				Timestamp:    time.Now().UTC(),
				RowCount:     0,
				LogOffset:    0,
				LastInsertID: 0,
				AffectedRows: 0,
			},
		},
		Payload: ResponsePayload{
			Columns:   cols,
			DeclTypes: types,
			Rows:      rows,
		},
	}
	if err = r.BuildHash(); err != nil {
		panic(err)
	}
	return
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

	if testingPrivateKey, testingPublicKey, err = asymmetric.GenSecp256k1KeyPair(); err == nil {
		kms.SetLocalKeyPair(testingPrivateKey, testingPublicKey)
	} else {
		panic(err)
	}

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func createRandomBlock(parent hash.Hash, isGenesis bool) (b *Block, err error) {
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
				Version:     0x01000000,
				Producer:    proto.NodeID(h.String()),
				GenesisHash: genesisHash,
				ParentHash:  parent,
				Timestamp:   time.Now().UTC(),
			},
		},
	}

	if isGenesis {
		// Compute nonce with public key
		nonceCh := make(chan cpuminer.NonceInfo)
		quitCh := make(chan struct{})
		miner := cpuminer.NewCPUMiner(quitCh)
		go miner.ComputeBlockNonce(cpuminer.MiningBlock{
			Data:      pub.Serialize(),
			NonceChan: nonceCh,
			Stop:      nil,
		}, cpuminer.Uint256{A: 0, B: 0, C: 0, D: 0}, 4)
		nonce := <-nonceCh
		close(quitCh)
		close(nonceCh)
		// Add public key to KMS
		id := cpuminer.HashBlock(pub.Serialize(), nonce.Nonce)
		b.SignedHeader.Header.Producer = proto.NodeID(id.String())

		if err = kms.SetPublicKey(proto.NodeID(id.String()), nonce.Nonce, pub); err != nil {
			return nil, err
		}

		// Set genesis hash as zero value
		b.SignedHeader.GenesisHash = hash.Hash{}
	}

	err = b.PackAndSignBlock(priv)
	return
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}
