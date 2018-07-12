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
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

var (
	testHeight  = int32(50)
	genesisHash = hash.Hash{}
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

func setup() {
	rand.Seed(time.Now().UnixNano())
	rand.Read(genesisHash[:])
	f, err := ioutil.TempFile("", "keystore")

	if err != nil {
		panic(err)
	}

	f.Close()

	if err = kms.InitPublicKeyStore(f.Name(), nil); err != nil {
		panic(err)
	}

	kms.Unittest = true

	if priv, pub, err := asymmetric.GenSecp256k1KeyPair(); err == nil {
		kms.SetLocalKeyPair(priv, pub)
	} else {
		panic(err)
	}

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func createRandomString(offset, length int, s *string) {
	buff := make([]byte, rand.Intn(length)+offset)
	rand.Read(buff)
	*s = string(buff)
}

func createRandomStrings(offset, length, soffset, slength int) (s []string) {
	s = make([]string, rand.Intn(length)+offset)

	for i := range s {
		createRandomString(soffset, slength, &s[i])
	}

	return
}

func createRandomStorageQueries(offset, length, soffset, slength int) (qs []storage.Query) {
	qs = make([]storage.Query, rand.Intn(length)+offset)

	for i := range qs {
		createRandomString(soffset, slength, &qs[i].Pattern)
	}

	return
}

func createRandomTimeAfter(now time.Time, maxDelayMillisecond int) time.Time {
	return now.Add(time.Duration(rand.Intn(maxDelayMillisecond)+1) * time.Millisecond)
}

func createRandomQueryRequest(cli *nodeProfile) (r *wt.SignedRequestHeader, err error) {
	req := &wt.Request{
		Header: wt.SignedRequestHeader{
			RequestHeader: wt.RequestHeader{
				QueryType:    wt.QueryType(rand.Intn(2)),
				NodeID:       cli.NodeID,
				ConnectionID: uint64(rand.Int63()),
				SeqNo:        uint64(rand.Int63()),
				Timestamp:    time.Now().UTC(),
				// BatchCount and QueriesHash will be set by req.Sign()
			},
			Signee:    cli.PublicKey,
			Signature: nil,
		},
		Payload: wt.RequestPayload{
			Queries: createRandomStorageQueries(10, 10, 10, 10),
		},
	}

	createRandomString(10, 10, (*string)(&req.Header.DatabaseID))

	if err = req.Sign(cli.PrivateKey); err != nil {
		return
	}

	r = &req.Header
	return
}

func createRandomQueryResponse(cli, worker *nodeProfile) (
	r *wt.SignedResponseHeader, err error,
) {
	req, err := createRandomQueryRequest(cli)

	if err != nil {
		return
	}

	resp := &wt.Response{
		Header: wt.SignedResponseHeader{
			ResponseHeader: wt.ResponseHeader{
				Request:   *req,
				NodeID:    worker.NodeID,
				Timestamp: createRandomTimeAfter(req.Timestamp, 100),
			},
			Signee:    worker.PublicKey,
			Signature: nil,
		},
		Payload: wt.ResponsePayload{
			Columns:   createRandomStrings(10, 10, 10, 10),
			DeclTypes: createRandomStrings(10, 10, 10, 10),
			Rows:      make([]wt.ResponseRow, rand.Intn(10)+10),
		},
	}

	for i := range resp.Payload.Rows {
		s := createRandomStrings(10, 10, 10, 10)
		resp.Payload.Rows[i].Values = make([]interface{}, len(s))
		for j := range resp.Payload.Rows[i].Values {
			resp.Payload.Rows[i].Values[j] = s[j]
		}
	}

	if err = resp.Sign(worker.PrivateKey); err != nil {
		return
	}

	r = &resp.Header
	return
}

func createRandomQueryResponseWithRequest(req *wt.SignedRequestHeader, worker *nodeProfile) (
	r *wt.SignedResponseHeader, err error,
) {
	resp := &wt.Response{
		Header: wt.SignedResponseHeader{
			ResponseHeader: wt.ResponseHeader{
				Request:   *req,
				NodeID:    worker.NodeID,
				Timestamp: createRandomTimeAfter(req.Timestamp, 100),
			},
			Signee:    worker.PublicKey,
			Signature: nil,
		},
		Payload: wt.ResponsePayload{
			Columns:   createRandomStrings(10, 10, 10, 10),
			DeclTypes: createRandomStrings(10, 10, 10, 10),
			Rows:      make([]wt.ResponseRow, rand.Intn(10)+10),
		},
	}

	for i := range resp.Payload.Rows {
		s := createRandomStrings(10, 10, 10, 10)
		resp.Payload.Rows[i].Values = make([]interface{}, len(s))
		for j := range resp.Payload.Rows[i].Values {
			resp.Payload.Rows[i].Values[j] = s[j]
		}
	}

	if err = resp.Sign(worker.PrivateKey); err != nil {
		return
	}

	r = &resp.Header
	return
}

func createRandomQueryAckWithResponse(resp *wt.SignedResponseHeader, cli *nodeProfile) (
	r *wt.SignedAckHeader, err error,
) {
	ack := &wt.Ack{
		Header: wt.SignedAckHeader{
			AckHeader: wt.AckHeader{
				Response:  *resp,
				NodeID:    cli.NodeID,
				Timestamp: createRandomTimeAfter(resp.Timestamp, 100),
			},
			Signee:    cli.PublicKey,
			Signature: nil,
		},
	}

	if err = ack.Sign(cli.PrivateKey); err != nil {
		return
	}

	r = &ack.Header
	return
}

func createRandomQueryAck(cli, worker *nodeProfile) (r *wt.SignedAckHeader, err error) {
	resp, err := createRandomQueryResponse(cli, worker)

	if err != nil {
		return
	}

	ack := &wt.Ack{
		Header: wt.SignedAckHeader{
			AckHeader: wt.AckHeader{
				Response:  *resp,
				NodeID:    cli.NodeID,
				Timestamp: createRandomTimeAfter(resp.Timestamp, 100),
			},
			Signee:    cli.PublicKey,
			Signature: nil,
		},
	}

	if err = ack.Sign(cli.PrivateKey); err != nil {
		return
	}

	r = &ack.Header
	return
}

func createRandomNodesAndAck() (r *wt.SignedAckHeader, err error) {
	cli, err := newRandomNode()

	if err != nil {
		return
	}

	worker, err := newRandomNode()

	if err != nil {
		return
	}

	return createRandomQueryAck(cli, worker)
}

func createRandomBlock(parent hash.Hash, isGenesis bool) (b *ct.Block, err error) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &ct.Block{
		SignedHeader: ct.SignedHeader{
			Header: ct.Header{
				Version:     0x01000000,
				Producer:    proto.NodeID(h.String()),
				GenesisHash: genesisHash,
				ParentHash:  parent,
				Timestamp:   time.Now().UTC(),
			},
			Signee:    pub,
			Signature: nil,
		},
	}

	for i, n := 0, rand.Intn(10)+10; i < n; i++ {
		h := &hash.Hash{}
		rand.Read(h[:])
		b.PushAckedQuery(h)
	}

	if isGenesis {
		// Compute nonce with public key
		nonceCh := make(chan cpuminer.NonceInfo)
		quitCh := make(chan struct{})
		miner := cpuminer.NewCPUMiner(quitCh)
		go miner.ComputeBlockNonce(cpuminer.MiningBlock{
			Data:      pub.Serialize(),
			NonceChan: nonceCh,
			Stop:      nil,
		}, cpuminer.Uint256{A: 0, B: 0, C: 0, D: 0}, 4)
		nonce := <-nonceCh
		close(quitCh)
		close(nonceCh)
		// Add public key to KMS
		id := cpuminer.HashBlock(pub.Serialize(), nonce.Nonce)
		b.SignedHeader.Header.Producer = proto.NodeID(id.String())

		if err = kms.SetPublicKey(proto.NodeID(id.String()), nonce.Nonce, pub); err != nil {
			return nil, err
		}

		// Set genesis hash as zero value
		b.SignedHeader.GenesisHash = hash.Hash{}
	}

	err = b.PackAndSignBlock(priv)
	return
}

func createRandomQueries(x int) (acks []*wt.SignedAckHeader, err error) {
	n := rand.Intn(x)
	acks = make([]*wt.SignedAckHeader, n)

	for i := range acks {
		if acks[i], err = createRandomNodesAndAck(); err != nil {
			return
		}
	}

	return
}

func createRandomBlockWithQueries(genesis, parent hash.Hash, acks []*wt.SignedAckHeader) (
	b *ct.Block, err error,
) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &ct.Block{
		SignedHeader: ct.SignedHeader{
			Header: ct.Header{
				Version:     0x01000000,
				Producer:    proto.NodeID(h.String()),
				GenesisHash: genesis,
				ParentHash:  parent,
				Timestamp:   time.Now().UTC(),
			},
			Signee:    pub,
			Signature: nil,
		},
	}

	for _, ack := range acks {
		b.PushAckedQuery(&ack.HeaderHash)
	}

	err = b.PackAndSignBlock(priv)
	return
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}
