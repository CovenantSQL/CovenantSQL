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
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/pkg/errors"
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
	qi.heightIndex.mustGet(0).respIndex[resp.Hash()].response = nil

	if err = qi.addResponse(0, resp); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}

	if err = qi.addAck(0, ack); err != ErrCorruptedIndex {
		t.Fatalf("Unexpected error: %v", err)
	}

	qi.heightIndex.mustGet(0).respIndex[resp.Hash()] = nil

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

	ackHash := b1.Acks[0].Header.Hash()
	if _, err := qi.checkAckFromBlock(
		0, b1.BlockHash(), &ackHash,
	); errors.Cause(err) != ErrQueryExpired {
		t.Fatalf("Unexpected error: %v", err)
	}

	if isKnown, err := qi.checkAckFromBlock(
		height, b1.BlockHash(), &ackHash,
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

	b1.Acks[0] = &types.Ack{
		Header: *ack1,
	}
	b2.Acks[0] = &types.Ack{
		Header: *ack1,
	}
	ack1Hash := ack1.Hash()
	qi.setSignedBlock(height, b1)

	if _, err := qi.checkAckFromBlock(
		height, b2.BlockHash(), &ack1Hash,
	); err != ErrQuerySignedByAnotherBlock {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test checking same ack signed by another block
	ack2Hash := ack2.Hash()
	b2.Acks[0] = &types.Ack{
		Header: *ack2,
	}

	if _, err = qi.checkAckFromBlock(
		height, b2.BlockHash(), &ack2Hash,
	); err != ErrQuerySignedByAnotherBlock {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Revert index state for the first block, and test checking again
	qi.heightIndex.mustGet(height).seqIndex[req.GetQueryKey()].firstAck.signedBlock = nil

	if _, err = qi.checkAckFromBlock(
		height, b2.BlockHash(), &ack2Hash,
	); err != nil {
		t.Fatalf("Error occurred: %v", err)
	}

	// Test corrupted index
	qi.heightIndex.mustGet(height).seqIndex[req.GetQueryKey()] = nil

	if _, err = qi.checkAckFromBlock(
		height, b2.BlockHash(), &ack2Hash,
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
