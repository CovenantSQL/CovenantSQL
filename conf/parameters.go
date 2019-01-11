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

import "time"

// This parameters should be kept consistent in all BPs.
const (
	// BPPeriod is the block producer block produce period.
	BPPeriod = 3 * time.Second
	// BPTick is the block produce block fetch tick.
	BPTick = 1 * time.Second
	// SQLChainPeriod is the sqlchain block produce period.
	SQLChainPeriod = 3 * time.Second
	// SQLChainTick is the sqlchain block fetch tick.
	SQLChainTick = 1 * time.Second
	// SQLChainTTL is the sqlchain unack query billing ttl.
	SQLChainTTL = 10

	DefaultConfirmThreshold = float64(2) / 3.0
)

// This parameters will not cause inconsistency within certain range.
const (
	BPStartupRequiredReachableCount = 2 // NOTE: this includes myself
)
