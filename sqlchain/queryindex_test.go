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

import (
	"github.com/pkg/errors"
	"math/rand"
	"reflect"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	testBucketNumber         = 10
	testQueryNumberPerHeight = 10
	testClientNumber         = 10
	testWorkerNumber         = 10
	testQueryWorkerNumber    = 3
)

func (i *heightIndex) mustGet(k int32) *multiIndex {
	i.Lock()
	defer i.Unlock()
	return i.index[k]
}

func TestCorruptedIndex(t *testing.T) {
	ack, err := createRandomNodesAndAck()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	resp := ack.SignedResponseHeader()

	// Create index
	qi := newQueryIndex()

	if err = qi.addResponse(0, resp); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = qi.addAck(0, ack); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Test repeatedly add
	if err = qi.addResponse(0, resp); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = qi.addAck(0, ack); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Test corrupted index
	qi.heightIndex.mustGet(0).respIndex[resp.Hash].response = nil

	if err = qi.addResponse(0, resp); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}

	if err = qi.addAck(0, ack); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}

	qi.heightIndex.mustGet(0).respIndex[resp.Hash] = nil

	if err = qi.addResponse(0, resp); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}

	if err = qi.addAck(0, ack); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestSingleAck(t *testing.T) {
	ack, err := createRandomNodesAndAck()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	qi := newQueryIndex()

	if err = qi.addAck(0, ack); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Check not signed ack
	qi.heightIndex.mustGet(0).checkBeforeExpire()
}

func TestEnsureRange(t *testing.T) {
	qi := newQueryIndex()
	qi.heightIndex.ensureRange(0, 10)

	for i := 0; i < 10; i++ {
		if _, ok := qi.heightIndex.get(int32(i)); !ok {
			t.Fatalf("Failed to ensure height %d", i)
		}
	}
}

func TestCheckAckFromBlock(t *testing.T) {
	var height int32 = 10
	qi := newQueryIndex()
	qi.advanceBarrier(height)
	b1, err := createRandomBlock(genesisHash, false)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if _, err := qi.checkAckFromBlock(
		0, b1.BlockHash(), b1.Queries[0],
	); errors.Cause(err) != ErrQueryExpired {
		t.Fatalf("Unexpected error: %v", err)
	}

	if isKnown, err := qi.checkAckFromBlock(
		height, b1.BlockHash(), b1.Queries[0],
	); err != nil {
		t.Fatalf("Error occurred: %v", err)
	} else if isKnown {
		t.Fatal("Unexpected result: index should not know this query")
	}

	// Create a group of query for test
	cli, err := newRandomNode()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	worker1, err := newRandomNode()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	worker2, err := newRandomNode()

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	req, err := createRandomQueryRequest(cli)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	resp1, err := createRandomQueryResponseWithRequest(req, worker1)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	ack1, err := createRandomQueryAckWithResponse(resp1, cli)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	resp2, err := createRandomQueryResponseWithRequest(req, worker2)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	ack2, err := createRandomQueryAckWithResponse(resp2, cli)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Test a query signed by another block
	if err = qi.addAck(height, ack1); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	if err = qi.addAck(height, ack2); err != ErrMultipleAckOfSeqNo {
		t.Fatalf("Unexpected error: %v", err)
	}

	b2, err := createRandomBlock(genesisHash, false)

	if err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	b1.Queries[0] = &ack1.Hash
	b2.Queries[0] = &ack1.Hash
	qi.setSignedBlock(height, b1)

	if _, err := qi.checkAckFromBlock(
		height, b2.BlockHash(), b2.Queries[0],
	); err != ErrQuerySignedByAnotherBlock {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test checking same ack signed by another block
	b2.Queries[0] = &ack2.Hash

	if _, err = qi.checkAckFromBlock(
		height, b2.BlockHash(), b2.Queries[0],
	); err != ErrQuerySignedByAnotherBlock {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Revert index state for the first block, and test checking again
	qi.heightIndex.mustGet(height).seqIndex[req.GetQueryKey()].firstAck.signedBlock = nil

	if _, err = qi.checkAckFromBlock(
		height, b2.BlockHash(), b2.Queries[0],
	); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Test corrupted index
	qi.heightIndex.mustGet(height).seqIndex[req.GetQueryKey()] = nil

	if _, err = qi.checkAckFromBlock(
		height, b2.BlockHash(), b2.Queries[0],
	); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestGetAck(t *testing.T) {
	qi := newQueryIndex()
	qh := &hash.Hash{}

	if _, err := qi.getAck(-1, qh); errors.Cause(err) != ErrQueryExpired {
		t.Fatalf("Unexpected error: %v", err)
	}

	if _, err := qi.getAck(0, qh); errors.Cause(err) != ErrQueryNotCached {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestQueryIndex(t *testing.T) {
	log.SetLevel(log.InfoLevel)
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
	qi := newQueryIndex()

	// Create some responses and acknowledgements and insert to index
	for i := 0; i < testBucketNumber; i++ {
		qi.advanceBarrier(int32(i))
		block, err := createRandomBlock(genesisHash, false)

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

				log.Debugf("i = %d, j = %d, k = %d\n\tseqno = %+v, req = %v, resp = %v", i, j, k,
					resp.Request.GetQueryKey(), &req.Hash, &resp.Hash)

				if err = qi.addResponse(int32(i), resp); err != nil {
					t.Fatalf("Error occurred: %v", err)
				}

				if k < ackNumber {
					dupAckNumber := 1 + rand.Intn(2)

					for l := 0; l < dupAckNumber; l++ {
						ack, err := createRandomQueryAckWithResponse(resp, cli)

						log.Debugf("i = %d, j = %d, k = %d, l = %d\n\tseqno = %+v, "+
							"req = %v, resp = %v, ack = %v",
							i, j, k, l,
							ack.SignedRequestHeader().GetQueryKey(),
							&ack.SignedRequestHeader().Hash,
							&ack.SignedResponseHeader().Hash,
							&ack.Hash,
						)

						if err != nil {
							t.Fatalf("Error occurred: %v", err)
						}

						err = qi.addAck(int32(i), ack)

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
							block.PushAckedQuery(&ack.Hash)
						} else {
							continue
						}

						if rAck, err := qi.getAck(int32(i), &ack.Hash); err != nil {
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

			qi.setSignedBlock(int32(i), block)

			for j := range block.Queries {
				if isKnown, err := qi.checkAckFromBlock(
					int32(i), block.BlockHash(), block.Queries[j],
				); err != nil {
					t.Fatalf("Error occurred: %v", err)
				} else if !isKnown {
					t.Logf("Failed to check known ack: %s", block.Queries[j])
				}
			}

			qi.resetSignedBlock(int32(i), block)

			for j := range block.Queries {
				if isKnown, err := qi.checkAckFromBlock(
					int32(i), block.BlockHash(), block.Queries[j],
				); err != nil {
					t.Fatalf("Error occurred: %v", err)
				} else if !isKnown {
					t.Fatal("Unexpected result: block is known")
				}
			}
		}
	}
}
