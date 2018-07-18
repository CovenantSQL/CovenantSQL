package types

import (
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

type TxHeader struct {
}

type SignedTxHeader struct {
	TxHeader
	TxHash    *hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

type Transaction struct {
	SignedHeader SignedTxHeader
}
