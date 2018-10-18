package types

import (
"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
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

