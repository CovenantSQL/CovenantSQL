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
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	pt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	genesisHash        = hash.Hash{}
	uuidLen            = 32
	peerNum     uint32 = 32

	testAddress1      = proto.AccountAddress{0x0, 0x0, 0x0, 0x1}
	testAddress2      = proto.AccountAddress{0x0, 0x0, 0x0, 0x2}
	testInitBalance   = uint64(10000)
	testDifficulty    = 4
	testDataDir       string
	testAddress1Nonce pi.AccountNonce
	testPrivKey       *asymmetric.PrivateKey
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func generateRandomBytes(n int32) []byte {
	s := make([]byte, n)
	for i := range s {
		s[i] = byte(rand.Int31n(2))
	}
	return s
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

func generateRandomBlock(parent hash.Hash, isGenesis bool) (b *pt.Block, err error) {
	// Generate key pair
	priv, _, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &pt.Block{
		SignedHeader: pt.SignedHeader{
			Header: pt.Header{
				Version:    0x01000000,
				Producer:   proto.AccountAddress(h),
				ParentHash: parent,
				Timestamp:  time.Now().UTC(),
			},
		},
	}

	if !isGenesis {
		for i, n := 0, rand.Intn(10)+10; i < n; i++ {
			ba, tb, err := generateRandomBillingAndBaseAccount()
			if err != nil {
				return nil, err
			}
			b.Transactions = append(b.Transactions, ba, tb)
		}
	} else {
		// Create base accounts
		var (
			ba1 = pt.NewBaseAccount(
				&pt.Account{
					Address:             testAddress1,
					StableCoinBalance:   testInitBalance,
					CovenantCoinBalance: testInitBalance,
				},
			)
			ba2 = pt.NewBaseAccount(
				&pt.Account{
					Address:             testAddress2,
					StableCoinBalance:   testInitBalance,
					CovenantCoinBalance: testInitBalance,
				},
			)
		)
		if err = ba1.Sign(testPrivKey); err != nil {
			return
		}
		if err = ba2.Sign(testPrivKey); err != nil {
			return
		}
		b.Transactions = append(b.Transactions, ba1, ba2)
	}

	err = b.PackAndSignBlock(priv)
	return
}

func generateRandomBlockWithTransactions(parent hash.Hash, tbs []pi.Transaction) (b *pt.Block, err error) {
	// Generate key pair
	priv, _, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &pt.Block{
		SignedHeader: pt.SignedHeader{
			Header: pt.Header{
				Version:    0x01000000,
				Producer:   proto.AccountAddress(h),
				ParentHash: parent,
				Timestamp:  time.Now().UTC(),
			},
		},
	}

	for _, tb := range tbs {
		b.Transactions = append(b.Transactions, tb)
	}

	testAddress1Nonce++
	var tr = pt.NewTransfer(
		&pt.TransferHeader{
			Sender:   testAddress1,
			Receiver: testAddress2,
			Nonce:    testAddress1Nonce,
			Amount:   1,
		},
	)
	if err = tr.Sign(priv); err != nil {
		return
	}
	b.Transactions = append(b.Transactions, tr)

	err = b.PackAndSignBlock(priv)

	return
}

func generateRandomBillingRequestHeader() *pt.BillingRequestHeader {
	return &pt.BillingRequestHeader{
		DatabaseID: *generateRandomDatabaseID(),
		LowBlock:   generateRandomHash(),
		LowHeight:  rand.Int31(),
		HighBlock:  generateRandomHash(),
		HighHeight: rand.Int31(),
		GasAmounts: generateRandomGasAmount(peerNum),
	}
}

func generateRandomBillingRequest() (*pt.BillingRequest, error) {
	reqHeader := generateRandomBillingRequestHeader()
	req := pt.BillingRequest{
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

func generateRandomBillingHeader() (tc *pt.BillingHeader, err error) {
	var req *pt.BillingRequest
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

	tc = pt.NewBillingHeader(pi.AccountNonce(rand.Uint32()), req, producer, receivers, fees, rewards)
	return tc, nil
}

func generateRandomBillingAndBaseAccount() (*pt.BaseAccount, *pt.Billing, error) {
	header, err := generateRandomBillingHeader()
	if err != nil {
		return nil, nil, err
	}
	priv, _, err := asymmetric.GenSecp256k1KeyPair()
	header.Producer, _ = crypto.PubKeyHash(priv.PubKey())

	txBilling := pt.NewBilling(header)

	if err := txBilling.Sign(priv); err != nil {
		return nil, nil, err
	}

	txBaseAccount := pt.NewBaseAccount(
		&pt.Account{
			Address:             header.Producer,
			StableCoinBalance:   testInitBalance,
			CovenantCoinBalance: testInitBalance,
		},
	)

	if err := txBaseAccount.Sign(priv); err != nil {
		return nil, nil, err
	}

	return txBaseAccount, txBilling, nil
}

func generateRandomAccountBilling() (*pt.Billing, error) {
	header, err := generateRandomBillingHeader()
	if err != nil {
		return nil, err
	}
	header.Producer = testAddress1
	testAddress1Nonce++
	header.Nonce = testAddress1Nonce
	txBilling := pt.NewBilling(header)

	if err := txBilling.Sign(testPrivKey); err != nil {
		return nil, err
	}

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

func generateRandomHash() hash.Hash {
	h := hash.Hash{}
	rand.Read(h[:])
	return h
}

func registerNodesWithPublicKey(pub *asymmetric.PublicKey, diff int, num int) (
	nis []cpuminer.NonceInfo, err error) {
	nis = make([]cpuminer.NonceInfo, num)

	miner := cpuminer.NewCPUMiner(nil)
	nCh := make(chan cpuminer.NonceInfo)
	defer close(nCh)
	block := cpuminer.MiningBlock{
		Data:      pub.Serialize(),
		NonceChan: nCh,
		Stop:      nil,
	}
	next := cpuminer.Uint256{}
	wg := &sync.WaitGroup{}

	for i := range nis {
		wg.Add(1)
		go func() {
			defer wg.Done()
			miner.ComputeBlockNonce(block, next, diff)
		}()
		n := <-nCh
		nis[i] = n
		next = n.Nonce
		next.Inc()

		if err = kms.SetPublicKey(proto.NodeID(n.Hash.String()), n.Nonce, pub); err != nil {
			return
		}

		wg.Wait()
	}

	// Register a local nonce, don't know what is the matter though
	kms.SetLocalNodeIDNonce(nis[0].Hash[:], &nis[0].Nonce)
	return
}

func createRandomString(offset, length int, s *string) {
	buff := make([]byte, rand.Intn(length)+offset)
	rand.Read(buff)
	*s = string(buff)
}

func createTestPeersWithPrivKeys(priv *asymmetric.PrivateKey, num int) (nis []cpuminer.NonceInfo, p *proto.Peers, err error) {
	if num <= 0 {
		return
	}

	pub := priv.PubKey()

	nis, err = registerNodesWithPublicKey(pub, testDifficulty, num)

	if err != nil {
		return
	}

	s := make([]proto.NodeID, num)
	h := &hash.Hash{}

	for i := range s {
		rand.Read(h[:])
		s[i] = proto.NodeID(nis[i].Hash.String())
	}

	p = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Term:    0,
			Leader:  s[0],
			Servers: s,
		},
	}

	if err = p.Sign(priv); err != nil {
		return
	}

	return
}

func createTestPeers(num int) (nis []cpuminer.NonceInfo, p *proto.Peers, err error) {
	if num <= 0 {
		return
	}

	// Use a same key pair for all the servers, so that we can run multiple instances of sql-chain
	// locally without breaking the LocalKeyStore
	pub, err := kms.GetLocalPublicKey()

	if err != nil {
		return
	}

	priv, err := kms.GetLocalPrivateKey()

	if err != nil {
		return
	}

	nis, err = registerNodesWithPublicKey(pub, testDifficulty, num)

	if err != nil {
		return
	}

	s := make([]proto.NodeID, num)
	h := &hash.Hash{}

	for i := range s {
		rand.Read(h[:])
		s[i] = proto.NodeID(nis[i].Hash.String())
	}

	p = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Term:    0,
			Leader:  s[0],
			Servers: s,
		},
	}

	if err = p.Sign(priv); err != nil {
		return
	}

	return
}

func setup() {
	var err error

	rand.Seed(time.Now().UnixNano())
	rand.Read(genesisHash[:])

	// Create key paire for test
	if testPrivKey, _, err = asymmetric.GenSecp256k1KeyPair(); err != nil {
		panic(err)
	}

	// Create temp dir for test data
	if testDataDir, err = ioutil.TempDir("", "covenantsql"); err != nil {
		panic(err)
	}

	// Setup logging
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func teardown() {
	if err := os.RemoveAll(testDataDir); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	os.Exit(func() int {
		setup()
		defer teardown()
		return m.Run()
	}())
}
