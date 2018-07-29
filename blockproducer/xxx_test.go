/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package blockproducer

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/utils/log"

	"gitlab.com/thunderdb/ThunderDB/blockproducer/types"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

var (
	genesisHash        = hash.Hash{}
	uuidLen            = 32
	peerNum     uint32 = 32

	testMasterKey = []byte(".9K.sgch!3;C>w0v")

	testDataDir     string
	testPrivKeyFile string
	testPubKeysFile string
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

	err = b.PackAndSignBlock(priv)
	for i := range b.TxBillings {
		b.TxBillings[i].SignedBlock = &b.SignedHeader.BlockHash
	}
	return
}

func generateRandomBillingRequestHeader() *types.BillingRequestHeader {
	return &types.BillingRequestHeader{
		DatabaseID:  *generateRandomDatabaseID(),
		BlockHash:   generateRandomHash(),
		BlockHeight: rand.Int31(),
		GasAmounts:  generateRandomGasAmount(peerNum),
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
		SequenceID:      rand.Uint64(),
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

func generateRandomTxBillingWithSeqID(seqID uint64) (*types.TxBilling, error) {
	txContent, err := generateRandomTxContent()
	txContent.SequenceID = seqID
	if err != nil {
		return nil, err
	}
	accountAddress := proto.AccountAddress(generateRandomHash())
	enc, err := txContent.MarshalBinary()
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
			GasAmount:      rand.Uint32(),
		}
	}

	return gasAmount
}

func generateRandomHash() hash.Hash {
	h := hash.Hash{}
	rand.Read(h[:])
	return h
}

func setup() {
	rand.Seed(time.Now().UnixNano())
	rand.Read(genesisHash[:])

	// Create temp dir
	var err error
	testDataDir, err = ioutil.TempDir("", "thunderdb")

	if err != nil {
		panic(err)
	}

	testPubKeysFile = path.Join(testDataDir, "pub")
	testPrivKeyFile = path.Join(testDataDir, "priv")

	// Setup public key store
	if err = kms.InitPublicKeyStore(testPubKeysFile, nil); err != nil {
		panic(err)
	}

	// Setup local key store
	kms.Unittest = true
	testPrivKey, testPubKey, err = asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		panic(err)
	}

	kms.SetLocalKeyPair(testPrivKey, testPubKey)

	if err = kms.SavePrivateKey(testPrivKeyFile, testPrivKey, testMasterKey); err != nil {
		panic(err)
	}

	// Setup logging
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}
