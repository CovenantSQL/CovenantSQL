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

package chain

import (
	"reflect"
	"testing"

	ci "gitlab.com/thunderdb/ThunderDB/chain/interfaces"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

func TestTxIndex(t *testing.T) {
	var (
		ti                 = NewTxIndex()
		otx ci.Transaction = newRandomDemoTxImpl()
	)
	// Test operations: Get -> Put -> Get -> Del -> Get
	if ok := ti.HasTx(otx.GetIndexKey()); ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if _, ok := ti.LoadTx(otx.GetIndexKey()); ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if ok := ti.SetBlock(otx.GetIndexKey(), hash.Hash{}); ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if err := ti.CheckTxState(otx.GetIndexKey()); err != ErrUnknownTx {
		t.Fatalf("Unexpected error: %v", err)
	}
	ti.StoreTx(otx)
	if ok := ti.HasTx(otx.GetIndexKey()); !ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if xtx, ok := ti.LoadTx(otx.GetIndexKey()); !ok {
		t.Fatalf("Unexpected query result: %v", ok)
	} else if !reflect.DeepEqual(otx, xtx) {
		t.Fatalf("Unexpected result:\n\torigin = %v\n\toutput = %v", otx, xtx)
	}
	if err := ti.CheckTxState(otx.GetIndexKey()); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ok := ti.SetBlock(otx.GetIndexKey(), hash.Hash{}); !ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if err := ti.CheckTxState(otx.GetIndexKey()); err != ErrDuplicateTx {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ok := ti.ResetBlock(otx.GetIndexKey()); !ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if txes := ti.FetchUnpackedTxes(); len(txes) != 1 {
		t.Fatalf("Unexpected query result: %v", txes)
	} else if !reflect.DeepEqual(otx, txes[0]) {
		t.Fatalf("Unexpected result:\n\torigin = %v\n\toutput = %v", otx, txes[0])
	}
	ti.DelTx(otx.GetIndexKey())
	if ok := ti.HasTx(otx.GetIndexKey()); ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if _, ok := ti.LoadTx(otx.GetIndexKey()); ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if ok := ti.SetBlock(otx.GetIndexKey(), hash.Hash{}); ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if err := ti.CheckTxState(otx.GetIndexKey()); err != ErrUnknownTx {
		t.Fatalf("Unexpected error: %v", err)
	}
}
