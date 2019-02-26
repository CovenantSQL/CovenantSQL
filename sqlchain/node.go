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

package sqlchain

import "github.com/CovenantSQL/CovenantSQL/proto"

// BlockID is the hash of block content.
type BlockID string

// StorageProofBlock records block's status.
type StorageProofBlock struct {
	// Block id
	ID BlockID
	// Nodes with index in the SQL chain.
	Nodes []proto.Node
}
