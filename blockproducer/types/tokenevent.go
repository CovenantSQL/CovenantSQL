package types

import (
	"bytes"

	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

//go:generate hsp

// TokenEvent defines ether transfer to CovenantSQL account.
type TokenEvent struct {
	From string
	Target string
	Value cpuminer.Uint256
	SequenceID uint64
	Token TokenType
}

// Serialize serializes TxBilling using msgpack.
func (t *TokenEvent) Serialize() (b []byte, err error) {
	var enc *bytes.Buffer
	if enc, err = utils.EncodeMsgPack(t); err != nil {
		return
	}
	b = enc.Bytes()
	return
}

// Deserialize desrializes TxBilling using msgpack.
func (t *TokenEvent) Deserialize(enc []byte) error {
	return utils.DecodeMsgPack(enc, t)
}
