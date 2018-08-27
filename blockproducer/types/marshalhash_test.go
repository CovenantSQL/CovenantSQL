package types

import (
	"bytes"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

func TestMarshalHashAccountStable(t *testing.T) {
	v := Account{
		Address:            proto.AccountAddress{0x10},
		StableCoinBalance:  10,
		ThunderCoinBalance: 10,
		SQLChains:          []proto.DatabaseID{"aaa", "bbb"},
		Roles:              []byte{0x10, 0x10},
		Rating:             1110,
		TxBillings: []*hash.Hash{
			{0x10},
			{0x20},
		},
	}
	bts1, err := v.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	bts2, err := v.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bts1, bts2) {
		t.Fatal("hash not stable")
	}
}

// test different type and member name but same data type and content hash identical
func TestMarshalHashAccountStable2(t *testing.T) {
	v1 := Account{
		Address:            proto.AccountAddress{0x10},
		StableCoinBalance:  10,
		ThunderCoinBalance: 10,
		SQLChains:          []proto.DatabaseID{"aaa", "bbb"},
		Roles:              []byte{0x10, 0x10},
		Rating:             1110,
		TxBillings: []*hash.Hash{
			{0x10},
			{0x20},
		},
	}
	v2 := Account4test{
		Address1:            proto.AccountAddress{0x10},
		StableCoinBalance1:  10,
		ThunderCoinBalance1: 10,
		SQLChains1:          []proto.DatabaseID{"aaa", "bbb"},
		Roles1:              []byte{0x10, 0x10},
		Rating1:             1110,
		TxBillings1: []*hash.Hash{
			{0x10},
			{0x20},
		},
	}
	bts1, err := v1.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	bts2, err := v2.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bts1, bts2) {
		t.Fatal("hash not stable")
	}
}
