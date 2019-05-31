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
	"fmt"

	"github.com/CovenantSQL/CovenantSQL/proto"
)

type blockProducerInfo struct {
	rank   uint32
	total  uint32
	role   string
	nodeID proto.NodeID
}

// String implements fmt.Stringer.
func (i *blockProducerInfo) String() string {
	return fmt.Sprintf("[%d/%d|%s] %s", i.rank+1, i.total, i.role, i.nodeID)
}

func buildBlockProducerInfos(
	localNodeID proto.NodeID, peers *proto.Peers, isAPINode bool,
) (
	localBPInfo *blockProducerInfo, bpInfos []*blockProducerInfo, err error,
) {
	var (
		total = len(peers.PeersHeader.Servers)
		index int32
		found bool
	)

	bpInfos = make([]*blockProducerInfo, total)
	for i, v := range peers.PeersHeader.Servers {
		var role = "F"
		if v == peers.Leader {
			role = "L"
		}
		bpInfos[i] = &blockProducerInfo{
			rank:   uint32(i),
			total:  uint32(total),
			role:   role,
			nodeID: v,
		}
	}

	if isAPINode {
		localBPInfo = &blockProducerInfo{
			rank:   0,
			total:  uint32(total),
			role:   "A",
			nodeID: localNodeID,
		}
		return localBPInfo, bpInfos, nil
	}

	if index, found = peers.Find(localNodeID); !found {
		err = ErrLocalNodeNotFound
		return
	}

	localBPInfo = bpInfos[index]

	return
}
