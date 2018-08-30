package types

import (
	"bytes"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

func TestMarshalHashAccountStable(t *testing.T) {
	v := Account{
		Address:             proto.AccountAddress{0x10},
		StableCoinBalance:   10,
		CovenantCoinBalance: 10,
		Rating:              1110,
		NextNonce:           1,
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

func TestMarshalHashAccountStable2(t *testing.T) {
	v1 := Account{
		Address:             proto.AccountAddress{0x10},
		StableCoinBalance:   10,
		CovenantCoinBalance: 10,
		Rating:              1110,
		NextNonce:           1,
	}
	enc, err := utils.EncodeMsgPack(&v1)
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	v2 := Account{}
	if err = utils.DecodeMsgPack(enc.Bytes(), &v2); err != nil {
		t.Fatalf("Error occurred: %v", err)
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
