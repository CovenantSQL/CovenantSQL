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

package sqlchain

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

var (
	testHeight = int32(50)
	rootHash   = hash.Hash{}
)

func testSetup() {
	rand.Seed(time.Now().UnixNano())
	rand.Read(rootHash[:])
	f, err := ioutil.TempFile("", "keystore")

	if err != nil {
		panic(err)
	}

	f.Close()
	kms.InitPublicKeyStore(f.Name(), nil)

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func createRandomBlock(parent hash.Hash, isGenesis bool) (b *Block, err error) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	b = &Block{
		SignedHeader: &SignedHeader{
			Header: Header{
				Version:    0x01000000,
				RootHash:   rootHash,
				ParentHash: parent,
				Timestamp:  time.Now().UTC(),
			},
			Signee:    pub,
			Signature: nil,
		},
		Queries: make([]*Query, rand.Intn(10)+10),
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b.SignedHeader.Header.Producer = proto.NodeID(h.String())

	for i := 0; i < len(b.Queries); i++ {
		b.Queries[i] = new(Query)
		rand.Read(b.Queries[i].TxnID[:])
	}

	// TODO(leventeliu): use merkle package to generate this field from queries.
	rand.Read(b.SignedHeader.Header.MerkleRoot[:])

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
		err = kms.SetPublicKey(proto.NodeID(id.String()), nonce.Nonce, pub)

		if err != nil {
			return nil, err
		}
	}

	err = b.SignHeader(priv)

	if err != nil {
		return nil, err
	}

	return
}

func TestMain(m *testing.M) {
	testSetup()
	os.Exit(m.Run())
}
