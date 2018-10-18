package types

import (
	"bytes"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

//go:generate hsp

// TokenReceiveHeader defines the ether receive transaction header.
type TokenReceiveHeader struct {
	Observer, Receiver proto.AccountAddress
	Nonce            pi.AccountNonce
	Amount           cpuminer.Uint256
	Type TokenType
}

// TokenReceive defines the ether receive transaction.
type TokenReceive struct {
	TokenReceiveHeader
	HeaderHash hash.Hash
	Signee     *asymmetric.PublicKey
	Signature  *asymmetric.Signature
}

// Serialize serializes TxBilling using msgpack.
func (t *TokenReceive) Serialize() (b []byte, err error) {
	var enc *bytes.Buffer
	if enc, err = utils.EncodeMsgPack(t); err != nil {
		return
	}
	b = enc.Bytes()
	return
}

// Deserialize desrializes TxBilling using msgpack.
func (t *TokenReceive) Deserialize(enc []byte) error {
	return utils.DecodeMsgPack(enc, t)
}

// GetAccountAddress implements interfaces/Transaction.GetAccountAddress.
func (t *TokenReceive) GetAccountAddress() proto.AccountAddress {
	return t.Observer
}

// GetAccountNonce implements interfaces/Transaction.GetAccountNonce.
func (t *TokenReceive) GetAccountNonce() pi.AccountNonce {
	return t.Nonce
}

// GetHash implements interfaces/Transaction.GetHash.
func (t *TokenReceive) GetHash() hash.Hash {
	return t.HeaderHash
}

// GetTransactionType implements interfaces/Transaction.GetTransactionType.
func (t *TokenReceive) GetTransactionType() pi.TransactionType {
	return pi.TransactionTypeReceiveEther
}

// Sign implements interfaces/Transaction.Sign.
func (t *TokenReceive) Sign(signer *asymmetric.PrivateKey) (err error) {
	var enc []byte
	if enc, err = t.TokenReceiveHeader.MarshalHash(); err != nil {
		return
	}
	var h = hash.THashH(enc)
	if t.Signature, err = signer.Sign(h[:]); err != nil {
		return
	}
	t.HeaderHash = h
	t.Signee = signer.PubKey()
	return
}

// Verify implements interfaces/Transaction.Verify.
func (t *TokenReceive) Verify() (err error) {
	var enc []byte
	if enc, err = t.TokenReceiveHeader.MarshalHash(); err != nil {
		return
	} else if h := hash.THashH(enc); !t.HeaderHash.IsEqual(&h) {
		err = ErrSignVerification
		return
	} else if !t.Signature.Verify(h[:], t.Signee) {
		err = ErrSignVerification
		return
	}
	return
}

