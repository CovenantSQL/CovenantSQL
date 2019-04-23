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

// WithReceipt returns a context and a pointer pointed to a atomic value, where a *Receipt will be
// attached to once the request succeeds.
func WithReceipt(ctx context.Context) (context.Context, *atomic.Value) {
	var value atomic.Value
	value.Store((*Receipt)(nil))
	return context.WithValue(ctx, &ctxReceiptKey, &value), &value
}

// GetReceipt tries to get *Receipt from value.
func GetReceipt(value *atomic.Value) (rec *Receipt, ok bool) {
	reci := value.Load()
	rec, ok = reci.(*Receipt)
	if rec == nil {
		ok = false
	}
	return
}
