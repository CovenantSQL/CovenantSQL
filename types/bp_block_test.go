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
	"bytes"
	"encoding"
	"errors"
	"reflect"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

func TestHeader_MarshalUnmarshalBinary(t *testing.T) {

	block, err := generateRandomBlock(genesisHash, false)
	header := &block.SignedHeader.BPHeader
	if err != nil {
		t.Fatalf("failed to generate block: %v", err)
	}

	enc, err := utils.EncodeMsgPack(header)
	if err != nil {
		t.Fatalf("failed to mashal binary: %v", err)
	}

	dec := &BPHeader{}
	err = utils.DecodeMsgPack(enc.Bytes(), dec)
	if err != nil {
		t.Fatalf("failed to unmashal binary: %v", err)
	}

	if !reflect.DeepEqual(header, dec) {
		t.Fatalf("value not math:\n\tv1 = %+v\n\tv2 = %+v", block, dec)
	}
}

func TestSignedHeader_MarshalUnmashalBinary(t *testing.T) {
	block, err := generateRandomBlock(genesisHash, false)
	signedHeader := &block.SignedHeader
	if err != nil {
		t.Fatalf("failed to generate block: %v", err)
	}

	enc, err := utils.EncodeMsgPack(signedHeader)
	if err != nil {
		t.Fatalf("failed to mashal binary: %v", err)
	}

	dec := &BPSignedHeader{}
	err = utils.DecodeMsgPack(enc.Bytes(), dec)
	if err != nil {
		t.Fatalf("failed to unmashal binary: %v", err)
	}

	if !reflect.DeepEqual(signedHeader, dec) {
		t.Fatalf("value not math:\n\tv1 = %+v\n\tv2 = %+v", signedHeader, dec)
	}

}

func TestBlock_MarshalUnmarshalBinary(t *testing.T) {
	block, err := generateRandomBlock(genesisHash, false)
	if err != nil {
		t.Fatalf("failed to generate block: %v", err)
	}
	h := reflect.TypeOf(block)
	_, ok := h.(encoding.BinaryMarshaler)
	if ok {
		t.Log("dec hash BinaryMashaler interface")
	}

	enc, err := utils.EncodeMsgPack(block)
	if err != nil {
		t.Fatalf("failed to mashal binary: %v", err)
	}

	dec := &BPBlock{}
	err = utils.DecodeMsgPack(enc.Bytes(), dec)
	if err != nil {
		t.Fatalf("failed to unmashal binary: %v", err)
	}

	bts1, err := block.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	bts2, err := dec.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bts1, bts2) {
		t.Fatal("hash not stable")
	}
}

func TestBlock_PackAndSignBlock(t *testing.T) {
	block, err := generateRandomBlock(genesisHash, false)
	if err != nil {
		t.Fatalf("failed to generate block: %v", err)
	}

	err = block.verifyMerkleRoot()
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	err = block.VerifyHash()
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	err = block.Verify()
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	block.SignedHeader.DataHash[0]++
	err = block.Verify()
	if !errors.Is(err, verifier.ErrHashValueNotMatch) {
		t.Fatalf("unexpected error: %v", err)
	}
	err = block.VerifyHash()
	if !errors.Is(err, verifier.ErrHashValueNotMatch) {
		t.Fatalf("unexpected error: %v", err)
	}
	err = block.SetHash()
	if err != nil {
		t.Fatalf("failed to set hash: %v", err)
	}
	err = block.VerifyHash()
	if err != nil {
		t.Fatalf("failed to verify: %v", err)
	}

	block.SignedHeader.MerkleRoot[0]++
	err = block.Verify()
	if err != ErrMerkleRootVerification {
		t.Fatalf("unexpected error: %v", err)
	}
	err = block.VerifyHash()
	if err != ErrMerkleRootVerification {
		t.Fatalf("unexpected error: %v", err)
	}
	err = block.verifyMerkleRoot()
	if err != ErrMerkleRootVerification {
		t.Fatalf("unexpected error: %v", err)
	}

	tb, err := generateRandomTransfer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	block.Transactions = append(block.Transactions, tb)
	err = block.Verify()
	if err != ErrMerkleRootVerification {
		t.Fatalf("unexpected error: %v", err)
	}

	block.Transactions = append(block.Transactions, nil)
	err = block.Verify()
	if err != ErrNilBlockMember {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBPBlockTransactionCommitment(t *testing.T) {
	Convey("A signed block commits to the complete transactions it carries", t, func() {
		signer, _ := asymmetric.PrivKeyFromBytes(bytes.Repeat([]byte{1}, 32))
		relayer, _ := asymmetric.PrivKeyFromBytes(bytes.Repeat([]byte{2}, 32))
		bp, _ := asymmetric.PrivKeyFromBytes(bytes.Repeat([]byte{3}, 32))
		nodeID := (&proto.RawNodeID{}).ToNodeID()
		resign := func(tx pi.Transaction) error { return tx.Sign(relayer) }
		cases := []struct {
			name   string
			tx     pi.Transaction
			mutate func(pi.Transaction) error
		}{
			{"ProvideService signer", NewProvideService(&ProvideServiceHeader{NodeID: nodeID, Nonce: 1}), resign},
			{"UpdatePermission signer", NewUpdatePermission(&UpdatePermissionHeader{Nonce: 1}), resign},
			{"IssueKeys signer", NewIssueKeys(&IssueKeysHeader{Nonce: 1}), resign},
			{"UpdateBilling signer", NewUpdateBilling(&UpdateBillingHeader{Nonce: 1, Version: 1}), resign},
			{"CreateDatabase timestamp", NewCreateDatabase(&CreateDatabaseHeader{Nonce: 1}), func(tx pi.Transaction) error {
				cd := tx.(*CreateDatabase)
				cd.Timestamp = cd.Timestamp.Add(time.Nanosecond)
				return nil
			}},
			{"transaction type", NewCreateDatabase(&CreateDatabaseHeader{Nonce: 1}), func(tx pi.Transaction) error {
				tx.(*CreateDatabase).TxType = pi.TransactionTypeBaseAccount
				return nil
			}},
		}
		for _, tc := range cases {
			Convey(tc.name, func() {
				tc.tx.(interface{ SetTimestamp(time.Time) }).SetTimestamp(time.Unix(1700000000, 123456789).UTC())
				So(tc.tx.Sign(signer), ShouldBeNil)
				So(tc.tx.Verify(), ShouldBeNil)
				block := &BPBlock{Transactions: []pi.Transaction{tc.tx}}
				So(block.PackAndSignBlock(bp), ShouldBeNil)
				So(block.Verify(), ShouldBeNil)

				enc, err := utils.EncodeMsgPack(block)
				So(err, ShouldBeNil)
				var changed BPBlock
				So(utils.DecodeMsgPack(enc.Bytes(), &changed), ShouldBeNil)
				So(changed.Verify(), ShouldBeNil)
				tx := changed.Transactions[0].(*pi.TransactionWrapper).Unwrap()
				So(tc.mutate(tx), ShouldBeNil)
				So(tx.Verify(), ShouldBeNil)
				So(tx.Hash(), ShouldEqual, tc.tx.Hash())
				So(changed.SignedHeader, ShouldResemble, block.SignedHeader)
				So(changed.Verify(), ShouldEqual, ErrMerkleRootVerification)
				So(changed.VerifyHash(), ShouldEqual, ErrMerkleRootVerification)

				// Recomputing the root and block hash cannot reuse the BP signature.
				So(changed.SetHash(), ShouldBeNil)
				So(*changed.BlockHash(), ShouldNotEqual, *block.BlockHash())
				So(changed.Verify(), ShouldEqual, verifier.ErrSignatureNotMatch)
			})
		}
	})
}

func TestBPBlockTransactionCommitmentRoundTrip(t *testing.T) {
	Convey("Complete transaction commitments survive MsgPack and interface wrappers", t, func() {
		key, _ := asymmetric.PrivKeyFromBytes(bytes.Repeat([]byte{1}, 32))
		nodeID := (&proto.RawNodeID{}).ToNodeID()
		txs := []pi.Transaction{
			NewTransfer(&TransferHeader{Amount: 1, Nonce: 1}),
			NewProvideService(&ProvideServiceHeader{NodeID: nodeID, TargetUser: []proto.AccountAddress{{1}}, Nonce: 1}),
			NewCreateDatabase(&CreateDatabaseHeader{ResourceMeta: ResourceMeta{
				TargetMiners: []proto.AccountAddress{{2}}, Node: 1, ConsistencyLevel: 0.5,
			}, Nonce: 1}),
			NewUpdatePermission(&UpdatePermissionHeader{Permission: &UserPermission{
				Role: Admin, Patterns: []string{"SELECT * FROM test"},
			}, Nonce: 1}),
			NewIssueKeys(&IssueKeysHeader{MinerKeys: []MinerKey{{EncryptionKey: "key"}}, Nonce: 1}),
			NewUpdateBilling(&UpdateBillingHeader{Version: 0, Nonce: 1}),
			NewUpdateBilling(&UpdateBillingHeader{Version: 1, Nonce: 1, Users: []*UserCost{
				{Cost: 1, Miners: []*MinerIncome{{Income: 1}, nil}}, nil,
			}}),
		}
		for _, tx := range txs {
			So(tx.Sign(key), ShouldBeNil)
		}
		// Nil verifier fields also occur in unsigned objects and must encode stably.
		unsigned := NewProvideService(&ProvideServiceHeader{NodeID: nodeID})
		keyOnly := NewProvideService(&ProvideServiceHeader{NodeID: nodeID})
		keyOnly.Signee = key.PubKey()
		txs = append(txs, unsigned, keyOnly, NewBaseAccount(&Account{Address: proto.AccountAddress{3}}))
		for _, ts := range []time.Time{
			{}, time.Unix(1700000000, 0).UTC(), time.Unix(-1, 123456789).UTC(),
			time.Unix(1700000000, 123456789).In(time.FixedZone("offset", 8*60*60)),
			time.Now(), // Include a monotonic clock reading and the local location.
		} {
			for _, tx := range txs {
				tx.(interface{ SetTimestamp(time.Time) }).SetTimestamp(ts)
			}
			block := &BPBlock{
				SignedHeader: BPSignedHeader{BPHeader: BPHeader{Timestamp: ts}},
				Transactions: txs,
			}
			So(block.PackAndSignBlock(key), ShouldBeNil)
			// Decode twice to cover concrete-to-wrapper and wrapper-to-wrapper transport.
			for round := 0; round < 2; round++ {
				enc, err := utils.EncodeMsgPack(block)
				So(err, ShouldBeNil)
				var decoded BPBlock
				So(utils.DecodeMsgPack(enc.Bytes(), &decoded), ShouldBeNil)
				So(decoded.Verify(), ShouldBeNil)
				So(decoded.VerifyHash(), ShouldBeNil)
				So(*decoded.BlockHash(), ShouldEqual, *block.BlockHash())
				So(len(decoded.Transactions), ShouldEqual, len(txs))
				for i, tx := range decoded.Transactions {
					wrapped := tx.(*pi.TransactionWrapper)
					So(reflect.TypeOf(wrapped.Unwrap()), ShouldEqual, reflect.TypeOf(txs[i]))
					before, err := txs[i].MarshalHash()
					So(err, ShouldBeNil)
					after, err := wrapped.MarshalHash()
					So(err, ShouldBeNil)
					So(after, ShouldResemble, before)
					So(tx.Hash(), ShouldEqual, txs[i].Hash())
				}
				block = &decoded
			}
		}
	})
}

func TestBPBlockGenesisCommitment(t *testing.T) {
	Convey("Genesis commits to base account contents", t, func() {
		account := &Account{Address: proto.AccountAddress{1}, TokenBalance: [SupportTokenNumber]uint64{100}}
		tx := NewBaseAccount(account)
		block := &BPBlock{Transactions: []pi.Transaction{tx}}
		So(block.SetHash(), ShouldBeNil)
		So(block.VerifyHash(), ShouldBeNil)
		So(block.SignedHeader.MerkleRoot, ShouldNotEqual, hash.Hash{})
		originalHash := *block.BlockHash()

		tx.TokenBalance[0]++
		So(tx.Hash(), ShouldEqual, hash.Hash{})
		So(block.VerifyHash(), ShouldEqual, ErrMerkleRootVerification)
		So(block.SetHash(), ShouldBeNil)
		So(*block.BlockHash(), ShouldNotEqual, originalHash)
	})
}

func TestBPBlockTransactionCommitmentError(t *testing.T) {
	Convey("A transaction encoding error prevents block hashing and verification", t, func() {
		// UpdateBilling rejects unsupported encoding versions in MarshalHash.
		tx := NewUpdateBilling(&UpdateBillingHeader{Version: -1})
		_, err := tx.MarshalHash()
		So(err, ShouldNotBeNil)
		block := &BPBlock{Transactions: []pi.Transaction{tx}}
		So(block.SetHash(), ShouldResemble, err)
		So(block.PackAndSignBlock(testingPrivateKey), ShouldResemble, err)
		So(block.VerifyHash(), ShouldResemble, err)
		So(block.Verify(), ShouldResemble, err)
	})
}
