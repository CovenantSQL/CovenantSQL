package types

import (
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
)

type TxHeader struct {

}

type SignedTxHeader struct {
	TxHeader
	TxHash *hash.Hash
	Signee *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

type Transaction struct {
	SignedHeader SignedTxHeader

}
