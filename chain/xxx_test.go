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

package chain

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils"
)

var (
	testDataDir string
	testPrivKey *asymmetric.PrivateKey
	testPubKey  *asymmetric.PublicKey
)

func createRandomString(offset, length int) string {
	buff := make([]byte, rand.Intn(length)+offset)
	rand.Read(buff)
	for i, v := range buff {
		buff[i] = v%(0x7f-0x20) + 0x20
	}
	return string(buff)
}

type DemoHeader struct {
	DatabaseID proto.DatabaseID
	SequenceID uint32
	Timestamp  time.Time
}

type DemoTxImpl struct {
	DemoHeader
	HeaderHash hash.Hash
	Signee     *asymmetric.PublicKey
	Signature  *asymmetric.Signature
}

func newRandomDemoTxImpl() (i *DemoTxImpl) {
	header := DemoHeader{
		DatabaseID: proto.DatabaseID(createRandomString(10, 10)),
		SequenceID: rand.Uint32(),
		Timestamp:  time.Now().UTC(),
	}

	enc, err := header.MarshalHash()
	if err != nil {
		panic(err)
	}

	hh := hash.HashH(enc)
	sig, err := testPrivKey.Sign(hh[:])
	if err != nil {
		panic(err)
	}

	i = &DemoTxImpl{
		DemoHeader: header,
		HeaderHash: hh,
		Signee:     testPubKey,
		Signature:  sig,
	}
	return
}

func (i *DemoTxImpl) Serialize() (enc []byte, err error) {
	if b, err := utils.EncodeMsgPack(i); err == nil {
		enc = b.Bytes()
	}
	return
}

func (i *DemoTxImpl) Deserialize(enc []byte) error {
	return utils.DecodeMsgPack(enc, i)
}

func (i *DemoTxImpl) GetDatabaseID() *proto.DatabaseID {
	return &i.DatabaseID
}

func (i *DemoTxImpl) GetHash() hash.Hash {
	return i.HeaderHash
}

func (i *DemoTxImpl) GetIndexKey() interface{} {
	return i.HeaderHash
}

func (i *DemoTxImpl) GetPersistenceKey() []byte {
	return i.HeaderHash[:]
}

func (i *DemoTxImpl) GetSequenceID() uint32 {
	return i.SequenceID
}

func (i *DemoTxImpl) GetTime() time.Time {
	return i.Timestamp
}

func (i *DemoTxImpl) Verify() (err error) {
	var enc []byte
	if enc, err = i.DemoHeader.MarshalHash(); err != nil {
		return
	} else if h := hash.THashH(enc); !i.HeaderHash.IsEqual(&h) {
		return
	} else if !i.Signature.Verify(h[:], i.Signee) {
		return
	}
	return
}

func setup() {
	// Setup RNG
	rand.Seed(time.Now().UnixNano())

	var err error
	// Create temp directory
	testDataDir, err = ioutil.TempDir("", "covenantsql")
	if err != nil {
		panic(err)
	}
	// Create key pair for test
	testPrivKey, testPubKey, err = asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		panic(err)
	}
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
