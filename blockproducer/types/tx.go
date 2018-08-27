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

//go:generate hsp

// TxType represents the type of tx.
type TxType byte

const (
	// TxTypeBilling defines TxType for database service billing.
	TxTypeBilling TxType = 0
)

// String returns the TxType name.
func (tt *TxType) String() string {
	switch *tt {
	case TxTypeBilling:
		return "TxBilling"
	default:
		return "TxUnknown"
	}
}

// ToByte returns the the that represents the TxType.
func (tt *TxType) ToByte() byte {
	return byte(*tt)
}
