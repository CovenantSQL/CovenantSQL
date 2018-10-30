/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
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
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

func TestMarshalServiceInstance(t *testing.T) {
	rawID := &proto.RawNodeID{}
	s := wt.ServiceInstance{
		DatabaseID: "happy",
		Peers: &kayak.Peers{
			Term: 1,
			Leader: &kayak.Server{
				Role:   1,
				ID:     "",
				PubKey: nil,
			},
			Servers:   nil,
			PubKey:    nil,
			Signature: nil,
		},
		ResourceMeta: wt.ResourceMeta{
			Node:          2,
			Space:         0,
			Memory:        0,
			LoadAvgPerCPU: 1,
			EncryptionKey: "",
		},
		GenesisBlock: &types.Block{
			SignedHeader: types.SignedHeader{
				Header: types.Header{
					Version:     1,
					Producer:    rawID.ToNodeID(),
					GenesisHash: hash.Hash{},
					ParentHash:  hash.Hash{},
					MerkleRoot:  hash.Hash{},
					Timestamp:   time.Time{},
				},
				BlockHash: hash.Hash{},
				Signee:    nil,
				Signature: nil,
			},
			Queries: nil,
		},
	}

	r := GetDatabaseResponse{}
	r.Header.InstanceMeta = s

	buf, err := utils.EncodeMsgPack(r)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	var r2 interface{}

	err = utils.DecodeMsgPack(buf.Bytes(), &r2)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	fmt.Printf("%#v\n", r2)
}
