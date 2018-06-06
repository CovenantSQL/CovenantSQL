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

package kayak

import (
	"reflect"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/crypto/signature"
)

func TestLog_ComputeHash(t *testing.T) {
	log1 := &Log{
		Index: 1,
		Term:  1,
		Data:  []byte("happy"),
	}

	log2 := &Log{
		Index: 1,
		Term:  1,
		Data:  []byte("happy"),
	}

	log1.ComputeHash()
	log2.ComputeHash()

	Convey("same hash result on identical field value", t, func() {
		equalHash := log1.Hash.IsEqual(&log2.Hash)
		So(equalHash, ShouldBeTrue)
	})
}

func TestLog_VerifyHash(t *testing.T) {
	// Test with no LastHash
	log1 := &Log{
		Index: 1,
		Term:  1,
		Data:  []byte("happy"),
	}

	log1.ComputeHash()

	Convey("verify correct hash", t, func() {
		So(log1.VerifyHash(), ShouldBeTrue)
	})

	// Test including LastHash
	log2 := &Log{
		Index:    2,
		Term:     1,
		Data:     []byte("happy2"),
		LastHash: &log1.Hash,
	}

	log2.ComputeHash()

	Convey("verify correct hash", t, func() {
		So(log2.VerifyHash(), ShouldBeTrue)
	})

	log2.Hash.SetBytes(hash.HashB([]byte("test generation")))

	Convey("verify incorrect hash", t, func() {
		So(log2.VerifyHash(), ShouldBeFalse)
	})
}

func TestServer_Serialize(t *testing.T) {
	testKey := []byte{
		0x04, 0x11, 0xdb, 0x93, 0xe1, 0xdc, 0xdb, 0x8a,
		0x01, 0x6b, 0x49, 0x84, 0x0f, 0x8c, 0x53, 0xbc, 0x1e,
		0xb6, 0x8a, 0x38, 0x2e, 0x97, 0xb1, 0x48, 0x2e, 0xca,
		0xd7, 0xb1, 0x48, 0xa6, 0x90, 0x9a, 0x5c, 0xb2, 0xe0,
		0xea, 0xdd, 0xfb, 0x84, 0xcc, 0xf9, 0x74, 0x44, 0x64,
		0xf8, 0x2e, 0x16, 0x0b, 0xfa, 0x9b, 0x8b, 0x64, 0xf9,
		0xd4, 0xc0, 0x3f, 0x99, 0x9b, 0x86, 0x43, 0xf6, 0x56,
		0xb4, 0x12, 0xa3,
	}

	pubKey, err := signature.ParsePubKey(testKey, btcec.S256())

	if err != nil {
		t.Fatalf("parse pubkey failed: %v", err.Error())
	}

	s := &Server{
		Role:    Leader,
		ID:      "happy",
		Address: "happy2",
		PubKey:  pubKey,
	}
	data := s.Serialize()

	// try to load data from serialization
	s2 := &Server{
		Role:    Leader,
		ID:      "happy",
		Address: "happy2",
		PubKey:  pubKey,
	}
	data2 := s2.Serialize()

	Convey("test serialization", t, func() {
		So(reflect.DeepEqual(data, data2), ShouldBeTrue)
	})

	Convey("test serialize with nil PubKey", t, func() {
		s.PubKey = nil
		So(reflect.DeepEqual(s.Serialize(), data2), ShouldBeFalse)
	})
}

func TestPeers_Clone(t *testing.T) {
	testPriv := []byte{
		0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
		0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
		0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
		0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
	}
	_, pubKey := signature.PrivKeyFromBytes(btcec.S256(), testPriv)

	samplePeersConf := &Peers{
		Term: 1,
		Leader: &Server{
			Role:    Leader,
			ID:      "happy",
			Address: "happy_address",
			PubKey:  pubKey,
		},
		Servers: []*Server{
			{
				Role:    Leader,
				ID:      "happy",
				Address: "happy_address",
				PubKey:  pubKey,
			},
		},
		PubKey: pubKey,
	}

	Convey("clone peers", t, func() {
		peers := samplePeersConf.Clone()
		So(peers.Term, ShouldEqual, samplePeersConf.Term)
		So(reflect.DeepEqual(peers.Leader, samplePeersConf.Leader), ShouldBeTrue)
		So(reflect.DeepEqual(peers.Servers, samplePeersConf.Servers), ShouldBeTrue)
		So(reflect.DeepEqual(peers.PubKey, samplePeersConf.PubKey), ShouldBeTrue)
		So(reflect.DeepEqual(peers.Signature, samplePeersConf.Signature), ShouldBeTrue)
	})
}

func TestPeers_Sign(t *testing.T) {
	testPriv := []byte{
		0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
		0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
		0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
		0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
	}
	privKey, pubKey := signature.PrivKeyFromBytes(btcec.S256(), testPriv)
	peers := &Peers{
		Term: 1,
		Leader: &Server{
			Role:    Leader,
			ID:      "happy",
			Address: "happy_address",
			PubKey:  pubKey,
		},
		Servers: []*Server{
			{
				Role:    Leader,
				ID:      "happy",
				Address: "happy_address",
				PubKey:  pubKey,
			},
		},
		PubKey: pubKey,
	}

	if err := peers.Sign(privKey); err != nil {
		t.Fatalf("sign peer conf failed: %v", err.Error())
	}
	Convey("verify signed peers", t, func() {
		So(peers.Verify(), ShouldBeTrue)
	})
}
