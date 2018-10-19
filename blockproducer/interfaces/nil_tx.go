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

package interfaces

import (
	"bytes"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

// NilTx defines empty transaction for transaction decode failure
type NilTx struct {
	TransactionTypeMixin
}

// NewNilTx returns new instance.
func NewNilTx() *NilTx {
	return &NilTx{
		TransactionTypeMixin: *NewTransactionTypeMixin(TransactionTypeNumber),
	}
}

// Serialize serializes NilTx using msgpack.
func (t *NilTx) Serialize() (b []byte, err error) {
	var enc *bytes.Buffer
	if enc, err = utils.EncodeMsgPack(t); err != nil {
		return
	}
	b = enc.Bytes()
	return
}

// Deserialize desrializes NilTx using msgpack.
func (t *NilTx) Deserialize(enc []byte) error {
	return utils.DecodeMsgPack(enc, t)
}

// GetAccountAddress implements Transaction.GetAccountAddress.
func (t *NilTx) GetAccountAddress() (addr proto.AccountAddress) {
	return
}

// GetAccountNonce implements Transaction.GetAccountNonce.
func (t *NilTx) GetAccountNonce() (nonce AccountNonce) {
	// BaseAccount nonce is not counted, always return 0.
	return
}

// GetHash implements Transaction.GetHash.
func (t *NilTx) GetHash() (h hash.Hash) {
	return
}

// Sign implements interfaces/Transaction.Sign.
func (t *NilTx) Sign(signer *asymmetric.PrivateKey) (err error) {
	return
}

// Verify implements interfaces/Transaction.Verify.
func (t *NilTx) Verify() (err error) {
	return
}

func init() {
	RegisterTransaction(TransactionTypeNumber, (*NilTx)(nil))
}
