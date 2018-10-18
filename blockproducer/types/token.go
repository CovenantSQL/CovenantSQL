package types

import (
	"bytes"
	"encoding/binary"

	"github.com/CovenantSQL/HashStablePack/marshalhash"
)

// Token defines token's number.
const SupportTokenNumber int32 = 3
var Token = [SupportTokenNumber]string{
	"Ether",
	"EOS",
	"Bitcoin",
}

type TokenType int32

const (
	// Ether defines Ethereum.
	Ether TokenType = iota
	// EOS defines EOS.
	EOS
	// Bitcoin defines Bitcoin.
	Bitcoin
)

// String returns token's symbol.
func (t TokenType) String() string {
	switch t {
	case Ether:
		return "Ether"
	case EOS:
		return "EOS"
	case Bitcoin:
		return "Bitcoin"
	default:
		return "Unknown"
	}
}

// FromString returns token's number.
func FromString(t string) TokenType {
	switch t {
	case "Ether":
		return Ether
	case "EOS":
		return EOS
	case "Bitcoin":
		return Bitcoin
	default:
		return -1
	}
}

// Listed returns if the token is listed in list.
func (t TokenType) Listed() bool {
	return t >= 0 && int32(t) < SupportTokenNumber
}

// MarshalHash marshals for hash.
func (i *TokenType) MarshalHash() (o []byte, err error) {
	var binBuf bytes.Buffer
	binary.Write(&binBuf, binary.BigEndian, i)
	return binBuf.Bytes(), nil
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message.
func (i *TokenType) Msgsize() (s int) {
	return marshalhash.BytesPrefixSize + 4
}
