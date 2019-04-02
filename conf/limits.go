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

package conf

const (
	// MaxPendingTxsPerAccount defines the limit of pending transactions of one account.
	MaxPendingTxsPerAccount = 1000
	// MaxTransactionsPerBlock defines the limit of transactions per block.
	MaxTransactionsPerBlock = 10000
	// MaxRPCPoolPhysicalConnection defines max physical connection for one node pair.
	MaxRPCPoolPhysicalConnection = 1024
	// MaxRPCMuxPoolPhysicalConnection defines max underlying physical connection of mux component
	// for one node pair.
	MaxRPCMuxPoolPhysicalConnection = 2
)

// These limits will not cause inconsistency within certain range.
const (
	// MaxTxBroadcastTTL defines the TTL limit of a AddTx request broadcasting within the
	// block producers.
	MaxTxBroadcastTTL = 1
	MaxCachedBlock    = 1000
)
