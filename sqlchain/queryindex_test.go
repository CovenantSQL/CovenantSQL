/*
 * Copyright 2018 The ThunderDB Authors.
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

import (
	"math/rand"
	"reflect"
	"testing"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

const (
	testBucketNumber         = 10
	testQueryNumberPerHeight = 10
	testClientNumber         = 10
	testWorkerNumber         = 10
	testQueryWorkerNumber    = 3
)

func TestCorruptedIndex(t *testing.T) {
	ack, err := createRandomNodesAndAck()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	resp := ack.SignedResponseHeader()

	// Create index
	qi := NewQueryIndex()

	if err = qi.AddResponse(0, resp); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = qi.AddAck(0, ack); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Test repeatedly add
	if err = qi.AddResponse(0, resp); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = qi.AddAck(0, ack); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Test corrupted index
	qi.heightIndex[0].RespIndex[resp.HeaderHash].Response = nil

	if err = qi.AddResponse(0, resp); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}

	if err = qi.AddAck(0, ack); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}

	qi.heightIndex[0].RespIndex[resp.HeaderHash] = nil

	if err = qi.AddResponse(0, resp); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}

	if err = qi.AddAck(0, ack); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestSingleAck(t *testing.T) {
	ack, err := createRandomNodesAndAck()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	qi := NewQueryIndex()

	if err = qi.AddAck(0, ack); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Check not signed ack
	qi.heightIndex[0].CheckBeforeExpire()
}

func TestEnsureRange(t *testing.T) {
	qi := NewQueryIndex()
	qi.heightIndex.EnsureRange(0, 10)

	for i := 0; i < 10; i++ {
		if _, ok := qi.heightIndex[int32(i)]; !ok {
			t.Fatalf("Failed to ensure height %d", i)
		}
	}
}

func TestCheckAckFromBlock(t *testing.T) {
	qi := NewQueryIndex()
	qi.AdvanceBarrier(10)
	b1, err := createRandomBlock(rootHash, false)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if _, err := qi.CheckAckFromBlock(
		0, &b1.SignedHeader.BlockHash, b1.Queries[0],
	); err != ErrQueryExpired {
		t.Fatalf("Unexpected error: %v", err)
	}

	if isKnown, err := qi.CheckAckFromBlock(
		10, &b1.SignedHeader.BlockHash, b1.Queries[0],
	); err != nil {
		t.Fatalf("Error occurred: %v", err)
	} else if isKnown {
		t.Fatal("Unexpected result: index should not know this query")
	}

	// Test a query signed by another block
	b2, err := createRandomBlock(rootHash, false)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	ack, err := createRandomNodesAndAck()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = qi.AddAck(10, ack); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	b1.Queries[0] = &ack.HeaderHash
	b2.Queries[0] = &ack.HeaderHash
	qi.SetSignedBlock(10, b1)

	if _, err := qi.CheckAckFromBlock(
		10, &b2.SignedHeader.BlockHash, b2.Queries[0],
	); err != ErrQuerySignedByAnotherBlock {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestGetAck(t *testing.T) {
	qi := NewQueryIndex()
	qh := &hash.Hash{}

	if _, err := qi.GetAck(-1, qh); err != ErrQueryExpired {
		t.Fatalf("Unexpected error: %v", err)
	}

	if _, err := qi.GetAck(0, qh); err != ErrQueryNotCached {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestQueryIndex(t *testing.T) {
	// Initialize clients and workers
	clients, err := newRandomNodes(testClientNumber)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	workers, err := newRandomNodes(testWorkerNumber)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Initialize index
	qi := NewQueryIndex()

	// Create some responses and acknowledgements and insert to index
	for i := 0; i < testBucketNumber; i++ {
		qi.AdvanceBarrier(int32(i))
		block, err := createRandomBlock(rootHash, false)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		block.Queries = block.Queries[:0]

		for j := 0; j < testQueryNumberPerHeight; j++ {
			cli := clients[rand.Intn(testClientNumber)]
			req, err := createRandomQueryRequest(cli)
			hasFirstAck := false

			if err != nil {
				t.Fatalf("Error occurred: %v", err)
			}

			ackNumber := rand.Intn(testQueryWorkerNumber + 1)

			for k := 0; k < testQueryWorkerNumber; k++ {
				worker := workers[(rand.Intn(testWorkerNumber)+k)%testWorkerNumber]
				resp, err := createRandomQueryResponseWithRequest(req, worker)

				if err != nil {
					t.Fatalf("Error occurred: %v", err)
				}

				t.Logf("i = %d, j = %d, k = %d\n\tseqno = %d, req = %v, resp = %v", i, j, k,
					resp.Request.SeqNo, &req.HeaderHash, &resp.HeaderHash)

				if err = qi.AddResponse(int32(i), resp); err != nil {
					t.Fatalf("Error occurred: %v", err)
				}

				if k < ackNumber {
					dupAckNumber := 1 + rand.Intn(2)

					for l := 0; l < dupAckNumber; l++ {
						ack, err := createRandomQueryAckWithResponse(resp, cli)

						t.Logf("i = %d, j = %d, k = %d, l = %d\n\tseqno = %d, "+
							"req = %v, resp = %v, ack = %v",
							i, j, k, l,
							ack.SignedRequestHeader().SeqNo,
							&ack.SignedRequestHeader().HeaderHash,
							&ack.SignedResponseHeader().HeaderHash,
							&ack.HeaderHash,
						)

						if err != nil {
							t.Fatalf("Error occurred: %v", err)
						}

						err = qi.AddAck(int32(i), ack)

						if !hasFirstAck {
							if l == 0 && err != nil ||
								l > 0 && err != nil && err != ErrMultipleAckOfResponse {
								t.Fatalf("Error occurred: %v", err)
							}
						} else {
							if l == 0 && err == nil {
								t.Fatalf("Unexpected error: %v", err)
							}
						}

						if err == nil {
							hasFirstAck = true
							block.PushAckedQuery(&ack.HeaderHash)
						} else {
							continue
						}

						if rAck, err := qi.GetAck(int32(i), &ack.HeaderHash); err != nil {
							t.Fatalf("Error occurred: %v", err)
						} else if !reflect.DeepEqual(ack, rAck) {
							t.Fatalf("Unexpected result:\n\torigin = %+v\n\toutput = %+v",
								ack, rAck)
						} else if !reflect.DeepEqual(
							ack.SignedResponseHeader(), rAck.SignedResponseHeader()) {
							t.Fatalf("Unexpected result:\n\torigin = %+v\n\toutput = %+v",
								ack.SignedResponseHeader(), rAck.SignedResponseHeader())
						}
					}
				}
			}

			qi.SetSignedBlock(int32(i), block)

			for j := range block.Queries {
				if isKnown, err := qi.CheckAckFromBlock(
					int32(i), &block.SignedHeader.BlockHash, block.Queries[j],
				); err != nil {
					t.Fatalf("Error occurred: %v", err)
				} else if !isKnown {
					t.Logf("Failed to check known ack: %s", block.Queries[j])
				}
			}
		}
	}
}
