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

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	genesisHash       = hash.Hash{}
	testingPrivateKey *asymmetric.PrivateKey
	testingPublicKey  *asymmetric.PublicKey
)

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
				Request:      *header,
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
	if err = r.Sign(testingPrivateKey); err != nil {
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

func createRandomString(offset, length int, s *string) {
	buff := make([]byte, rand.Intn(length)+offset)
	rand.Read(buff)
	*s = string(buff)
}

func createRandomStrings(offset, length, soffset, slength int) (s []string) {
	s = make([]string, rand.Intn(length)+offset)

	for i := range s {
		createRandomString(soffset, slength, &s[i])
	}

	return
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
