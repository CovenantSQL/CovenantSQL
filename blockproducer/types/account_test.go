/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import (
	"reflect"
	"testing"

	"gitlab.com/thunderdb/ThunderDB/proto"
)

func TestAccount_MarshalUnmarshaler(t *testing.T) {
	account := generateRandomAccount()
	b, err := account.MarshalBinary()
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	dec := &Account{}
	err = dec.UnmarshalBinary(b)
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if !reflect.DeepEqual(account, dec) {
		t.Fatalf("Values don't match:\n\tv1 = %+v\n\tv2 = %+v", account, dec)
	}
}

func TestAccount_AppendSQLChainAndRole(t *testing.T) {
	account := generateRandomAccount()
	if len(account.Roles) != len(account.SQLChains) {
		t.Fatalf("length not match: %+v", account)
	}

	databaseID := proto.DatabaseID(randStringBytes(32))
	role := Customer
	account.AppendSQLChainAndRole(&databaseID, role)

	if len(account.Roles) != len(account.SQLChains) {
		t.Fatalf("length not match: %+v", account)
	}

	if account.Roles[len(account.Roles)-1] != role {
		t.Fatalf("value not math:\n\tv1 = %+v\n\tv2 = %+v",
			account.Roles[len(account.Roles)-1], role)
	}

	if account.SQLChains[len(account.SQLChains)-1] != databaseID {
		t.Fatalf("value not math:\n\tv1 = %+v\n\tv2 = %+v",
			account.SQLChains[len(account.SQLChains)-1], databaseID)
	}
}
