package main

import (
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
)

//go:generate hsp

// EtherIn defines ether transfer to CovenantSQL account.
type EtherIn struct {
	From string
	Target string
	EthValue cpuminer.Uint256
	SequenceID uint64
}


