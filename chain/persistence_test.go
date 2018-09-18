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

package chain

import (
	"fmt"
	"path"
	"reflect"
	"testing"

	bolt "github.com/coreos/bbolt"
	ci "github.com/CovenantSQL/CovenantSQL/chain/interfaces"
)

func TestBadNewTxPersistence(t *testing.T) {
	fl := path.Join(testDataDir, fmt.Sprintf("%s.db", t.Name()))
	db, err := bolt.Open(fl, 0600, nil)
	if err = db.Close(); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	if _, err = NewTxPersistence(db); err == nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestTxPersistenceWithClosedDB(t *testing.T) {
	fl := path.Join(testDataDir, fmt.Sprintf("%s.db", t.Name()))
	db, err := bolt.Open(fl, 0600, nil)
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	tp, err := NewTxPersistence(db)
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	if err = db.Close(); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	var (
		otx ci.Transaction = newRandomDemoTxImpl()
		rtx ci.Transaction = &DemoTxImpl{}
	)
	if _, err := tp.GetTransaction(otx.GetPersistenceKey(), rtx); err == nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if err = tp.PutTransaction(otx); err == nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if err = tp.DelTransaction(otx.GetPersistenceKey()); err == nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestTxPersistence(t *testing.T) {
	fl := path.Join(testDataDir, fmt.Sprintf("%s.db", t.Name()))
	db, err := bolt.Open(fl, 0600, nil)
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	tp, err := NewTxPersistence(db)
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Test operations: Get -> Put -> Get -> Del -> Get
	var (
		otx ci.Transaction = newRandomDemoTxImpl()
		rtx ci.Transaction = &DemoTxImpl{}
	)
	if ok, err := tp.GetTransaction(otx.GetPersistenceKey(), rtx); err != nil {
		t.Fatalf("Error occurred: %v", err)
	} else if ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if err = tp.PutTransaction(otx); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	if ok, err := tp.GetTransaction(otx.GetPersistenceKey(), rtx); err != nil {
		t.Fatalf("Error occurred: %v", err)
	} else if !ok {
		t.Fatalf("Unexpected query result: %v", ok)
	} else if !reflect.DeepEqual(otx, rtx) {
		t.Fatalf("Unexpected result:\n\torigin = %v\n\toutput = %v", otx, rtx)
	}
	if err = tp.DelTransaction(otx.GetPersistenceKey()); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	if ok, err := tp.GetTransaction(otx.GetPersistenceKey(), rtx); err != nil {
		t.Fatalf("Error occurred: %v", err)
	} else if ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
}

func TestTxPersistenceWithIndex(t *testing.T) {
	fl := path.Join(testDataDir, fmt.Sprintf("%s.db", t.Name()))
	db, err := bolt.Open(fl, 0600, nil)
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	tp, err := NewTxPersistence(db)
	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	ti := NewTxIndex()

	// Test operations: Get -> Put -> Get -> Del -> Get
	var (
		otx ci.Transaction = newRandomDemoTxImpl()
		rtx ci.Transaction = &DemoTxImpl{}
	)
	if ok, err := tp.GetTransaction(otx.GetPersistenceKey(), rtx); err != nil {
		t.Fatalf("Error occurred: %v", err)
	} else if ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if err = tp.PutTransactionAndUpdateIndex(otx, ti); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	if ok, err := tp.GetTransaction(otx.GetPersistenceKey(), rtx); err != nil {
		t.Fatalf("Error occurred: %v", err)
	} else if !ok {
		t.Fatalf("Unexpected query result: %v", ok)
	} else if !reflect.DeepEqual(otx, rtx) {
		t.Fatalf("Unexpected result:\n\torigin = %v\n\toutput = %v", otx, rtx)
	}
	if xtx, ok := ti.LoadTx(otx.GetIndexKey()); !ok {
		t.Fatalf("Unexpected query result: %v", ok)
	} else if !reflect.DeepEqual(otx, xtx) {
		t.Fatalf("Unexpected result:\n\torigin = %v\n\toutput = %v", otx, xtx)
	}
	if err = tp.DelTransactionAndUpdateIndex(
		otx.GetPersistenceKey(), otx.GetIndexKey(), ti,
	); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}
	if ok, err := tp.GetTransaction(otx.GetPersistenceKey(), rtx); err != nil {
		t.Fatalf("Error occurred: %v", err)
	} else if ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
	if _, ok := ti.LoadTx(otx.GetIndexKey()); ok {
		t.Fatalf("Unexpected query result: %v", ok)
	}
}
