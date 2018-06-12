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
	"math/big"

	"time"

	"testing"

	"reflect"

	"github.com/btcsuite/btcd/btcec"
	"github.com/thunderdb/ThunderDB/crypto/asymmetric"
	"github.com/thunderdb/ThunderDB/crypto/hash"
	"github.com/thunderdb/ThunderDB/types"
)

var (
	voidTxSlice []Tx
	txSlice     []Tx
	header      SignedHeader
	block       Block
	voidBlock   Block
)

func init() {
	voidTxSlice = []Tx{}
	txSlice = make([]Tx, 10)

	var i int64
	for i = 0; i < 10; i++ {
		_, pub, err := asymmetric.GenSecp256k1KeyPair()

		if err != nil {
			return
		}
		txSlice[i] = Tx{
			TxHash: hash.DoubleHashH([]byte{byte(i)}),
			TxData: TxData{
				AccountNonce: uint64(i),
				Recipient: &types.AccountAddress{
					AccountAddress: hash.DoubleHashH([]byte{byte(i * i)}).String(),
				},
				Amount:  big.NewInt(int64(i)),
				Payload: hash.DoubleHashB([]byte{byte(i / 2)}),
				Signature: &btcec.Signature{
					R: big.NewInt(1238 * i),
					S: big.NewInt(890321 / (i + 1)),
				},
				PublicKey: pub,
			},
		}
	}

	_, pub, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		return
	}

	header = SignedHeader{
		BlockHash: hash.DoubleHashH([]byte{1, 2, 3}),
		PublicKey: pub,
		Signature: &btcec.Signature{
			R: big.NewInt(8391),
			S: big.NewInt(2371),
		},
	}
	header.Version = 2
	header.Producer = &types.AccountAddress{
		AccountAddress: hash.DoubleHashH([]byte{9, 1, 4, 2, 1, 10}).String(),
	}
	header.Root = hash.DoubleHashH([]byte{4, 2, 1, 10})
	header.Parent = hash.DoubleHashH([]byte{1, 9, 2, 22})
	header.MerkleRoot = hash.DoubleHashH([]byte{9, 2, 1, 11})
	header.Timestamp = time.Now().UTC()

	voidBlock = Block{
		Header: &header,
		Tx:     voidTxSlice,
	}
	block = Block{
		Header: &header,
		Tx:     txSlice,
	}
}

func TestTx(t *testing.T) {
	for _, tx := range txSlice {
		_, err := tx.TxData.marshal()
		if err != nil {
			t.Errorf("cannot serialized TxData: %v", tx.TxData)
		}

		serializedTx, err := tx.marshal()
		if err != nil {
			t.Errorf("cannot serialized Tx: %v", tx)
		}
		deserializedTx := Tx{}
		err = deserializedTx.unmarshal(serializedTx)
		if err != nil {
			t.Errorf("cannot deserialized Tx buff: %v", serializedTx)
		}
		if !reflect.DeepEqual(deserializedTx, tx) {
			t.Errorf("deserialized tx is not equal to tx: \ntx: %v\ndeserializedTx: %v", tx, deserializedTx)
		}
	}
}

func TestSignedHeader(t *testing.T) {
	serializedSignedHeader, err := header.marshal()
	if err != nil {
		t.Errorf("cannot serialized signedHeader: %v", header)
	}

	deserializedSignedHeader := SignedHeader{}
	err = deserializedSignedHeader.unmarshal(serializedSignedHeader)
	if err != nil {
		t.Errorf("cannot deserialized signedHeader: %v", serializedSignedHeader)
	}
	if !reflect.DeepEqual(deserializedSignedHeader, header) {
		t.Errorf("deserialized header is not equal to header: \nheader: %v\ndeserializedBlock: %v",
			header, deserializedSignedHeader)
	}
}

func TestBlock(t *testing.T) {
	serializedBlock, err := block.marshal()
	if err != nil {
		t.Errorf("cannot serialized block: %v", block)
	}

	deserializedBlock := Block{}
	err = deserializedBlock.unmarshal(serializedBlock)
	if err != nil {
		t.Errorf("cannot deserialized: %v", serializedBlock)
	}
	if !reflect.DeepEqual(deserializedBlock, block) {
		t.Errorf("deserialized block is not equal to block: \nblock: %v\ndeserializedBlock: %v",
			block, deserializedBlock)
	}
}
