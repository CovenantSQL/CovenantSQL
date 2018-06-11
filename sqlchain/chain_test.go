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

package sqlchain

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/crypto/kms"
	"github.com/thunderdb/ThunderDB/crypto/signature"
	"github.com/thunderdb/ThunderDB/pow/cpuminer"
	"github.com/thunderdb/ThunderDB/proto"
)

var (
	testHeight = int32(20)
	rootHash   = hash.Hash{
		0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
		0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
		0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
		0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
	f, err := ioutil.TempFile("", "keystore")

	if err != nil {
		panic(err)
	}

	f.Close()
	kms.InitPublicKeyStore(f.Name())

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
				Timestamp:  time.Now(),
			},
			Signee:    (*signature.PublicKey)(pub),
			Signature: nil,
		},
		Queries: make([]*Query, rand.Intn(10)+10),
	}

	h := hash.Hash{}
	rand.Read(h[:])
	copy(b.SignedHeader.Header.Producer[:], h.String())

	for i := 0; i < len(b.Queries); i++ {
		b.Queries[i] = new(Query)
		rand.Read(b.Queries[i].TxnID[:])
	}

	// TODO(leventeliu): use merkle package to generate this field from queries.
	rand.Read(b.SignedHeader.Header.MerkleRoot[:])

	if isGenesis {
		// Compute nonce with public key
		nonceCh := make(chan cpuminer.Nonce)
		quitCh := make(chan struct{})
		miner := cpuminer.NewCPUMiner(quitCh)
		go miner.ComputeBlockNonce(cpuminer.MiningBlock{
			Data:      pub.SerializeCompressed(),
			NonceChan: nonceCh,
			Stop:      nil,
		}, cpuminer.Uint256{0, 0, 0, 0}, 4)
		nonce := <-nonceCh
		close(quitCh)
		close(nonceCh)
		// Add public key to KMS
		id := cpuminer.HashBlock(pub.SerializeCompressed(), nonce.Nonce)
		copy(b.SignedHeader.Header.Producer[:], id.String())
		err = kms.SetPublicKey(proto.NodeID(id.String()), nonce.Nonce, pub)

		if err != nil {
			return nil, err
		}
	}

	err = b.SignHeader((*signature.PrivateKey)(priv))

	if err != nil {
		return nil, err
	}

	return
}

func TestChain(t *testing.T) {
	fl, err := ioutil.TempFile("", "chain")

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	fl.Close()

	// Create new chain
	genesis, err := createRandomBlock(rootHash, true)

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	chain, err := NewChain(&Config{
		DataDir: fl.Name(),
		Genesis: genesis,
	})

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	t.Logf("Create new chain: genesis hash = %s", genesis.SignedHeader.BlockHash.String())

	// Push blocks
	for block, err := createRandomBlock(
		genesis.SignedHeader.BlockHash, false,
	); err == nil; block, err = createRandomBlock(block.SignedHeader.BlockHash, false) {
		err = chain.PushBlock(block.SignedHeader)

		if err != nil {
			t.Fatalf("Error occurred: %s", err.Error())
		}

		t.Logf("Pushed new block: height = %d,  %s <- %s",
			chain.state.Height,
			block.SignedHeader.ParentHash.String(),
			block.SignedHeader.BlockHash.String())

		if chain.state.Height >= testHeight {
			break
		}
	}

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}

	// Reload chain from DB file and rebuild memory cache
	chain.db.Close()
	chain, err = LoadChain(&Config{DataDir: fl.Name()})

	if err != nil {
		t.Fatalf("Error occurred: %s", err.Error())
	}
}
