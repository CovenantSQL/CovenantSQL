package types

import (
	"bytes"
	"testing"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"
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
