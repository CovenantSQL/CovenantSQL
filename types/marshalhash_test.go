/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import (
	"bytes"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

func TestMarshalHashAccountStable(t *testing.T) {
	v := Account{
		Address:      proto.AccountAddress{0x10},
		TokenBalance: [SupportTokenNumber]uint64{10, 10},
		Rating:       1110,
		NextNonce:    1,
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
		Address:      proto.AccountAddress{0x10},
		TokenBalance: [SupportTokenNumber]uint64{10, 10},
		Rating:       1110,
		NextNonce:    1,
	}
	enc, err := utils.EncodeMsgPack(&v1)
	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}
	v2 := Account{}
	if err = utils.DecodeMsgPack(enc.Bytes(), &v2); err != nil {
		t.Fatalf("error occurred: %v", err)
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
