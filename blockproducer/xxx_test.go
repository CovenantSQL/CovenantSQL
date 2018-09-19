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
	"github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	pt "github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	genesisHash        = hash.Hash{}
	uuidLen            = 32
	peerNum     uint32 = 32

	testAddress1    = proto.AccountAddress{0x0, 0x0, 0x0, 0x1}
	testAddress2    = proto.AccountAddress{0x0, 0x0, 0x0, 0x2}
	testInitBalance = uint64(10000)
	testMasterKey   = []byte(".9K.sgch!3;C>w0v")
	testDifficulty  = 4
	testDataDir     string
	testPrivKeyFile string
	testPubKeysFile string
	testNonce       pi.AccountNonce
	testPrivKey     *asymmetric.PrivateKey
	testPubKey      *asymmetric.PublicKey
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

func generateRandomBlock(parent hash.Hash, isGenesis bool) (b *types.Block, err error) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &types.Block{
		SignedHeader: types.SignedHeader{
			Header: types.Header{
				Version:    0x01000000,
				Producer:   proto.AccountAddress(h),
				ParentHash: parent,
				Timestamp:  time.Now().UTC(),
			},
			Signee: pub,
		},
	}

	if !isGenesis {
		for i, n := 0, rand.Intn(10)+10; i < n; i++ {
			tb, err := generateRandomTxBilling()
			if err != nil {
				return nil, err
			}
			b.PushTx(tb)
		}
	} else {
		// Create base accounts
		var (
			ba1 = &pt.BaseAccount{
				Account: pt.Account{
					Address:             testAddress1,
					StableCoinBalance:   testInitBalance,
					CovenantCoinBalance: testInitBalance,
				},
			}
			ba2 = &pt.BaseAccount{
				Account: pt.Account{
					Address:             testAddress2,
					StableCoinBalance:   testInitBalance,
					CovenantCoinBalance: testInitBalance,
				},
			}
		)
		if err = ba1.Sign(testPrivKey); err != nil {
			return
		}
		if err = ba2.Sign(testPrivKey); err != nil {
			return
		}
		b.Transactions = append(b.Transactions, ba1)
		b.Transactions = append(b.Transactions, ba2)
	}

	err = b.PackAndSignBlock(priv)
	return
}

func generateRandomBlockWithTxBillings(parent hash.Hash, tbs []*types.TxBilling) (b *types.Block, err error) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &types.Block{
		SignedHeader: types.SignedHeader{
			Header: types.Header{
				Version:    0x01000000,
				Producer:   proto.AccountAddress(h),
				ParentHash: parent,
				Timestamp:  time.Now().UTC(),
			},
			Signee: pub,
		},
	}

	b.TxBillings = tbs

	testNonce++
	var tr = &pt.Transfer{
		TransferHeader: pt.TransferHeader{
			Sender:   testAddress1,
			Receiver: testAddress2,
			Nonce:    testNonce,
			Amount:   1,
		},
	}
	if err = tr.Sign(priv); err != nil {
		return
	}
	b.Transactions = append(b.Transactions, tr)

	err = b.PackAndSignBlock(priv)
	for i := range b.TxBillings {
		b.TxBillings[i].SignedBlock = &b.SignedHeader.BlockHash
	}

	return
}

func generateRandomBillingRequestHeader() *types.BillingRequestHeader {
	return &types.BillingRequestHeader{
		DatabaseID: *generateRandomDatabaseID(),
		LowBlock:   generateRandomHash(),
		LowHeight:  rand.Int31(),
		HighBlock:  generateRandomHash(),
		HighHeight: rand.Int31(),
		GasAmounts: generateRandomGasAmount(peerNum),
	}
}

func generateRandomBillingRequest() (*types.BillingRequest, error) {
	reqHeader := generateRandomBillingRequestHeader()
	req := types.BillingRequest{
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

func generateRandomBillingResponse() (*types.BillingResponse, error) {
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		return nil, err
	}
	h := generateRandomHash()
	sign, err := priv.Sign(h[:])
	if err != nil {
		return nil, err
	}
	resp := types.BillingResponse{
		AccountAddress: proto.AccountAddress(generateRandomHash()),
		RequestHash:    h,
		Signee:         pub,
		Signature:      sign,
	}
	return &resp, nil
}

func generateRandomTxContent() (*types.TxContent, error) {
	req, err := generateRandomBillingRequest()
	if err != nil {
		return nil, err
	}
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()
	sign, err := priv.Sign(req.RequestHash[:])
	if err != nil {
		return nil, err
	}
	resp := &types.BillingResponse{
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

	tc := &types.TxContent{
		SequenceID:      rand.Uint32(),
		BillingRequest:  *req,
		BillingResponse: *resp,
		Receivers:       receivers,
		Fees:            fees,
		Rewards:         rewards,
	}
	return tc, nil
}

func generateRandomTxBilling() (*types.TxBilling, error) {
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

	txBilling := &types.TxBilling{
		TxContent:      *txContent,
		AccountAddress: &accountAddress,
		TxHash:         &txHash,
		Signee:         pub,
		Signature:      sign,
		SignedBlock:    &blockHash,
	}
	return txBilling, nil
}

func generateRandomTxBillingWithSeqID(seqID uint32) (*types.TxBilling, error) {
	txContent, err := generateRandomTxContent()
	txContent.SequenceID = seqID
	if err != nil {
		return nil, err
	}
	accountAddress := testAddress1
	enc, err := txContent.MarshalHash()
	if err != nil {
		return nil, err
	}
	txHash := hash.THashH(enc)
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()
	sign, err := priv.Sign(txHash[:])
	if err != nil {
		return nil, err
	}

	txBilling := &types.TxBilling{
		TxContent:      *txContent,
		AccountAddress: &accountAddress,
		TxHash:         &txHash,
		Signee:         pub,
		Signature:      sign,
		SignedBlock:    nil,
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

func generateRandomNode() (node *nodeProfile, err error) {
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	node = &nodeProfile{
		PrivateKey: priv,
		PublicKey:  pub,
	}

	createRandomString(10, 10, (*string)(&node.NodeID))
	return
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

// createNodes assign Node asymmetric key pair and generate Node.NonceInfo
// Node.ID = Node.NonceInfo.Hash.
func createNodes(pubKey *asymmetric.PublicKey, timeThreshold time.Duration) *proto.Node {
	node := proto.NewNode()
	nonce := asymmetric.GetPubKeyNonce(pubKey, proto.NewNodeIDDifficulty, timeThreshold, nil)
	node.ID = proto.NodeID(nonce.Hash.String())
	node.Nonce = nonce.Nonce
	log.Debugf("Node: %v", node)
	return node
}

func createTestPeersWithPrivKeys(priv *asymmetric.PrivateKey, num int) (nis []cpuminer.NonceInfo, p *kayak.Peers, err error) {
	if num <= 0 {
		return
	}

	pub := priv.PubKey()

	nis, err = registerNodesWithPublicKey(pub, testDifficulty, num)

	if err != nil {
		return
	}

	s := make([]*kayak.Server, num)
	h := &hash.Hash{}

	for i := range s {
		rand.Read(h[:])
		s[i] = &kayak.Server{
			Role: func() proto.ServerRole {
				if i == 0 {
					return proto.Leader
				}
				return proto.Follower
			}(),
			ID:     proto.NodeID(nis[i].Hash.String()),
			PubKey: pub,
		}
	}

	p = &kayak.Peers{
		Term:      0,
		Leader:    s[0],
		Servers:   s,
		PubKey:    pub,
		Signature: nil,
	}

	if err = p.Sign(priv); err != nil {
		return
	}

	return
}

func createTestPeers(num int) (nis []cpuminer.NonceInfo, p *kayak.Peers, err error) {
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

	s := make([]*kayak.Server, num)
	h := &hash.Hash{}

	for i := range s {
		rand.Read(h[:])
		s[i] = &kayak.Server{
			Role: func() proto.ServerRole {
				if i == 0 {
					return proto.Leader
				}
				return proto.Follower
			}(),
			ID:     proto.NodeID(nis[i].Hash.String()),
			PubKey: pub,
		}
	}

	p = &kayak.Peers{
		Term:      0,
		Leader:    s[0],
		Servers:   s,
		PubKey:    pub,
		Signature: nil,
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
	if testPrivKey, testPubKey, err = asymmetric.GenSecp256k1KeyPair(); err != nil {
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
