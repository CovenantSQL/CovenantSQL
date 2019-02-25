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

package interfaces

import "time"

//go:generate hsp

// TransactionTypeMixin provide type heuristic features to transaction wrapper.
type TransactionTypeMixin struct {
	TxType    TransactionType
	Timestamp time.Time
}

// NewTransactionTypeMixin returns new instance.
func NewTransactionTypeMixin(txType TransactionType) *TransactionTypeMixin {
	return &TransactionTypeMixin{
		TxType:    txType,
		Timestamp: time.Now().UTC(),
	}
}

// ContainsTransactionTypeMixin interface defines interface to detect transaction type mixin.
type ContainsTransactionTypeMixin interface {
	SetTransactionType(TransactionType)
}

// GetTransactionType implements Transaction.GetTransactionType.
func (m *TransactionTypeMixin) GetTransactionType() TransactionType {
	return m.TxType
}

// SetTransactionType is a helper function for derived types.
func (m *TransactionTypeMixin) SetTransactionType(t TransactionType) {
	m.TxType = t
}

// GetTimestamp implements Transaciton.GetTimestamp().
func (m *TransactionTypeMixin) GetTimestamp() time.Time {
	return m.Timestamp
}

// SetTimestamp is a helper function for derived types.
func (m *TransactionTypeMixin) SetTimestamp(t time.Time) {
	m.Timestamp = t
}
