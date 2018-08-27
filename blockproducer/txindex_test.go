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

package blockproducer

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/blockproducer/types"
)

func Test_AddAndHasTxBilling(t *testing.T) {
	ti := newTxIndex()
	maxLen := 100
	mlen := rand.Intn(maxLen)
	tbs := make([]*types.TxBilling, mlen)
	for i := range tbs {
		tb, err := generateRandomTxBilling()
		if err != nil {
			t.Fatalf("unexpect error: %v", err)
		}
		tbs[i] = tb
		err = ti.addTxBilling(tb)
		if err != nil {
			t.Fatalf("unexpect error: %v", err)
		}
	}

	for i := range tbs {
		if !ti.hasTxBilling(tbs[i].TxHash) {
			t.Fatalf("tx (%v) should be included in tx index", tbs[i])
		}
		if val := ti.getTxBilling(tbs[i].TxHash); !reflect.DeepEqual(tbs[i], val) {
			t.Fatalf("tx (%v) should be included in tx index", tbs[i])
		}
		tb, err := generateRandomTxBilling()
		if err != nil {
			t.Fatalf("unexpect error: %v", err)
		}
		if ti.hasTxBilling(tb.TxHash) {
			t.Fatalf("tx (%v) should be excluded in tx index", tb)
		}
	}

	fetchedTbs := ti.fetchUnpackedTxBillings()
	for i := range fetchedTbs {
		if !reflect.DeepEqual(tbs[i], fetchedTbs[i]) {
			t.Fatalf("two values should be equal: \n\tv1=%v\n\tv2=%v", tbs[i], fetchedTbs[i])
		}
	}
}
