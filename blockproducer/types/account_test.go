/*
 * Copyright 2018 The ThunderDB Authors.
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
	"reflect"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/utils"
)

func TestAccount_MarshalUnmarshaler(t *testing.T) {
	account := generateRandomAccount()
	b, err := utils.EncodeMsgPack(account)
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	dec := &Account{}
	err = utils.DecodeMsgPack(b.Bytes(), dec)
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if !reflect.DeepEqual(account, dec) {
		t.Fatalf("Values don't match:\n\tv1 = %+v\n\tv2 = %+v", account, dec)
	}
}
