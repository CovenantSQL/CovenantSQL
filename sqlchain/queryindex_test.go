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

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

const (
	testBucketNumber         = 10
	testQueryNumberPerHeight = 10
	testClientNumber         = 10
	testWorkerNumber         = 10
	testQueryWorkerNumber    = 3
)

type nodeProfile struct {
	NodeID     proto.NodeID
	PrivateKey *asymmetric.PrivateKey
	PublicKey  *asymmetric.PublicKey
}

func newRandomNode() (node *nodeProfile, err error) {
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	node = &nodeProfile{
		PrivateKey: priv,
		PublicKey:  pub,
	}

	createRandomString(10, 10, (*string)(&node.NodeID))
	return
}

func newRandomNodes(n int) (nodes []*nodeProfile, err error) {
	nodes = make([]*nodeProfile, n)

	for i := range nodes {
		if nodes[i], err = newRandomNode(); err != nil {
			return
		}
	}

	return
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
	parent := rootHash

	// Create some responses and acknowledgements and insert to index
	for i := 0; i < testBucketNumber; i++ {
		qi.AdvanceBarrier(int32(i))
		block, err := createRandomBlock(parent, false)

		if err != nil {
			t.Fatalf("Error occurred: %v", err)
		}

		block.Queries = block.Queries[:0]

		for j := 0; j < testQueryNumberPerHeight; j++ {
			cli := clients[rand.Intn(testClientNumber)]
			req, err := createRandomQueryRequest(cli.NodeID, cli.PrivateKey, cli.PublicKey)

			if err != nil {
				t.Fatalf("Error occurred: %v", err)
			}

			ackNumber := rand.Intn(testQueryWorkerNumber + 1)

			for k := 0; k < testQueryWorkerNumber; k++ {
				worker := workers[(rand.Intn(testWorkerNumber)+k)%testWorkerNumber]
				resp, err := createRandomQueryResponseWithRequest(
					req, worker.NodeID, worker.PrivateKey, worker.PublicKey)

				t.Logf("i = %d, j = %d, k = %d\n\tseqno = %d, req = %v, resp = %v", i, j, k,
					resp.Request.SeqNo, &req.HeaderHash, &resp.HeaderHash)

				if err != nil {
					t.Fatalf("Error occurred: %v", err)
				}

				if err = qi.AddResponse(int32(i), resp); err != nil {
					t.Fatalf("Error occurred: %v", err)
				}

				if k < ackNumber {
					dupAckNumber := 1 + rand.Intn(2)

					for l := 0; l < dupAckNumber; l++ {
						ack, err := createRandomQueryAckWithResponse(
							resp, cli.NodeID, cli.PrivateKey, cli.PublicKey)

						t.Logf("i = %d, j = %d, k = %d, l = %d\n\tseqno = %d, req = %v, resp = %v, ack = %v",
							i, j, k, l,
							ack.SignedRequestHeader().SeqNo,
							&ack.SignedRequestHeader().HeaderHash,
							&ack.SignedResponseHeader().HeaderHash,
							&ack.HeaderHash,
						)

						if err != nil {
							t.Fatalf("Error occurred: %v", err)
						}

						if err = qi.AddAck(int32(i), ack); k == 0 && l == 0 && err != nil {
							t.Fatalf("Error occurred: %v", err)
						} else if k > 0 && l == 0 && err != ErrMultipleAckOfSeqNo {
							t.Fatalf("Unexpected error: %v", err)
						} else if l > 0 && err != ErrMultipleAckOfResponse {
							t.Fatalf("Unexpected error: %v", err)
						}

						if err != nil {
							block.PushAckedQuery(&ack.HeaderHash)
						}

						if l > 0 {
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
		}
	}
}
