/*
 * Copyright 2019 The CovenantSQL Authors.
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

package client

import (
	"context"
	"sync/atomic"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
)

var (
	ctxReceiptKey = "_cql_receipt"
)

// Receipt defines a receipt of CovenantSQL query request.
type Receipt struct {
	RequestHash hash.Hash
}

// WithReceipt returns a context who holds a *atomic.Value. A *Receipt will be set to this value
// after the query succeeds.
//
// Note that this context is safe for concurrent queries, but the value may be reset in another
// goroutines. So if you want to make use of Receipt in several goroutines, you should call this
// method to get separated child context in each goroutine.
func WithReceipt(ctx context.Context) context.Context {
	var value atomic.Value
	value.Store((*Receipt)(nil))
	return context.WithValue(ctx, &ctxReceiptKey, &value)
}

// GetReceipt tries to get *Receipt from context.
func GetReceipt(ctx context.Context) (rec *Receipt, ok bool) {
	vali := ctx.Value(&ctxReceiptKey)
	if vali == nil {
		return
	}
	value, ok := vali.(*atomic.Value)
	if !ok {
		return
	}
	reci := value.Load()
	rec, ok = reci.(*Receipt)
	if rec == nil {
		ok = false
	}
	return
}
