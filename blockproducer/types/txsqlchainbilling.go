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
	"sync"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

//go:generate hsp
//hsp:ignore sync.Mutex Mutex

// TxContent defines the customer's billing and block rewards in transaction.
type TxContent struct {
	SequenceID     uint32
	BillingRequest BillingRequest
	Receivers      []*proto.AccountAddress
	// Fee paid by stable coin
	Fees []uint64
	// Reward is share coin
	Rewards         []uint64
	BillingResponse BillingResponse
}

// NewTxContent generates new TxContent.
func NewTxContent(seqID uint32,
	bReq *BillingRequest,
	receivers []*proto.AccountAddress,
	fees []uint64,
	rewards []uint64,
	bResp *BillingResponse) *TxContent {
	return &TxContent{
		SequenceID:      seqID,
		BillingRequest:  *bReq,
		Receivers:       receivers,
		Fees:            fees,
		Rewards:         rewards,
		BillingResponse: *bResp,
	}
}

// GetHash returns the hash of transaction.
func (tb *TxContent) GetHash() (*hash.Hash, error) {
	b, err := tb.MarshalHash()
	if err != nil {
		return nil, err
	}
	h := hash.THashH(b)
	return &h, nil
}

// TxBilling is a type of tx, that is used to record sql chain billing and block rewards.
type TxBilling struct {
	mutex          sync.Mutex
	TxContent      TxContent
	TxType         byte
	AccountAddress *proto.AccountAddress
	TxHash         *hash.Hash
	Signee         *asymmetric.PublicKey
	Signature      *asymmetric.Signature
	SignedBlock    *hash.Hash
}

// NewTxBilling generates a new TxBilling.
func NewTxBilling(txContent *TxContent, txType TxType, addr *proto.AccountAddress) *TxBilling {
	return &TxBilling{
		TxContent:      *txContent,
		TxType:         txType.ToByte(),
		AccountAddress: addr,
	}
}

// Serialize serializes TxBilling using msgpack.
func (tb *TxBilling) Serialize() (b []byte, err error) {
	var enc *bytes.Buffer
	if enc, err = utils.EncodeMsgPack(tb); err != nil {
		return
	}
	b = enc.Bytes()
	return
}

// Deserialize desrializes TxBilling using msgpack.
func (tb *TxBilling) Deserialize(enc []byte) error {
	return utils.DecodeMsgPack(enc, tb)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (tb *TxBilling) GetAccountAddress() proto.AccountAddress {
	return *tb.AccountAddress
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (tb *TxBilling) GetAccountNonce() pi.AccountNonce {
	return pi.AccountNonce(tb.TxContent.SequenceID)
}

// GetHash implements interfaces/Transaction.GetHash.
func (tb *TxBilling) GetHash() hash.Hash {
	return *tb.TxHash
}

// GetTransactionType implements interfaces/Transaction.GetTransactionType.
func (tb *TxBilling) GetTransactionType() pi.TransactionType {
	return pi.TransactionTypeBilling
}

// PackAndSignTx computes tx of TxContent and signs it.
func (tb *TxBilling) PackAndSignTx(signer *asymmetric.PrivateKey) error {
	enc, err := tb.TxContent.MarshalHash()
	if err != nil {
		return err
	}
	h := hash.THashH(enc)
	tb.TxHash = &h

	pub := asymmetric.PublicKey(signer.PublicKey)
	tb.Signee = &pub

	signature, err := signer.Sign(h[:])
	if err != nil {
		return err
	}
	tb.Signature = signature

	return nil
}

// Verify verifies the signature of TxBilling.
func (tb *TxBilling) Verify() (err error) {
	var enc []byte
	if enc, err = tb.TxContent.MarshalHash(); err != nil {
		return
	} else if h := hash.THashH(enc); !tb.TxHash.IsEqual(&h) {
		return
	} else if !tb.Signature.Verify(h[:], tb.Signee) {
		return
	}
	return
}

// GetDatabaseID gets the database ID.
func (tb *TxBilling) GetDatabaseID() *proto.DatabaseID {
	return &tb.TxContent.BillingRequest.Header.DatabaseID
}

// GetSequenceID gets the sequence ID.
func (tb *TxBilling) GetSequenceID() uint32 {
	return tb.TxContent.SequenceID
}

// IsSigned shows whether the tx billing is signed.
func (tb *TxBilling) IsSigned() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	return tb.SignedBlock != nil
}

// SetSignedBlock sets the tx billing with block hash.
func (tb *TxBilling) SetSignedBlock(h *hash.Hash) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	tb.SignedBlock = h
}

// GetSignedBlock gets the block hash.
func (tb *TxBilling) GetSignedBlock() *hash.Hash {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	return tb.SignedBlock
}
