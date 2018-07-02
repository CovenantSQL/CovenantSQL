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

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/worker/types"
)

var (
	testHeight = int32(50)
	rootHash   = hash.Hash{}
)

func testSetup() {
	rand.Seed(time.Now().UnixNano())
	rand.Read(rootHash[:])
	f, err := ioutil.TempFile("", "keystore")

	if err != nil {
		panic(err)
	}

	f.Close()
	kms.InitPublicKeyStore(f.Name(), nil)

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

func createRandomQueryRequest(
	reqNode proto.NodeID, reqPriv *asymmetric.PrivateKey, reqPub *asymmetric.PublicKey,
) (r *types.SignedRequestHeader, err error) {
	req := &types.Request{
		Header: types.SignedRequestHeader{
			RequestHeader: types.RequestHeader{
				QueryType:    types.QueryType(rand.Intn(2)),
				NodeID:       reqNode,
				ConnectionID: uint64(rand.Int63()),
				SeqNo:        uint64(rand.Int63()),
				Timestamp:    time.Now().UTC(),
				// BatchCount and QueriesHash will be set by req.Sign()
			},
			Signee:    reqPub,
			Signature: nil,
		},
		Payload: types.RequestPayload{
			Queries: createRandomStrings(10, 10, 10, 10),
		},
	}

	createRandomString(10, 10, (*string)(&req.Header.DatabaseID))

	if err = req.Sign(reqPriv); err != nil {
		return
	}

	r = &req.Header
	return
}

func createRandomQueryResponse(
	reqNode proto.NodeID, reqPriv *asymmetric.PrivateKey, reqPub *asymmetric.PublicKey,
	respNode proto.NodeID, respPriv *asymmetric.PrivateKey, respPub *asymmetric.PublicKey,
) (r *types.SignedResponseHeader, err error) {
	req, err := createRandomQueryRequest(reqNode, reqPriv, reqPub)

	if err != nil {
		return
	}

	resp := &types.Response{
		Header: types.SignedResponseHeader{
			ResponseHeader: types.ResponseHeader{
				Request:   *req,
				NodeID:    respNode,
				Timestamp: time.Now().UTC(),
			},
			Signee:    respPub,
			Signature: nil,
		},
		Payload: types.ResponsePayload{
			Columns:   createRandomStrings(10, 10, 10, 10),
			DeclTypes: createRandomStrings(10, 10, 10, 10),
			Rows:      make([]types.ResponseRow, rand.Intn(10)+10),
		},
	}

	for i := range resp.Payload.Rows {
		s := createRandomStrings(10, 10, 10, 10)
		resp.Payload.Rows[i].Values = make([]interface{}, len(s))
		for j := range resp.Payload.Rows[i].Values {
			resp.Payload.Rows[i].Values[j] = s[j]
		}
	}

	if err = resp.Sign(respPriv); err != nil {
		return
	}

	r = &resp.Header
	return
}

func createRandomQueryResponseWithRequest(
	req *types.SignedRequestHeader,
	respNode proto.NodeID, respPriv *asymmetric.PrivateKey, respPub *asymmetric.PublicKey,
) (r *types.SignedResponseHeader, err error) {
	resp := &types.Response{
		Header: types.SignedResponseHeader{
			ResponseHeader: types.ResponseHeader{
				Request:   *req,
				NodeID:    respNode,
				Timestamp: time.Now().UTC(),
			},
			Signee:    respPub,
			Signature: nil,
		},
		Payload: types.ResponsePayload{
			Columns:   createRandomStrings(10, 10, 10, 10),
			DeclTypes: createRandomStrings(10, 10, 10, 10),
			Rows:      make([]types.ResponseRow, rand.Intn(10)+10),
		},
	}

	for i := range resp.Payload.Rows {
		s := createRandomStrings(10, 10, 10, 10)
		resp.Payload.Rows[i].Values = make([]interface{}, len(s))
		for j := range resp.Payload.Rows[i].Values {
			resp.Payload.Rows[i].Values[j] = s[j]
		}
	}

	if err = resp.Sign(respPriv); err != nil {
		return
	}

	r = &resp.Header
	return
}

func createRandomQueryAckWithResponse(
	resp *types.SignedResponseHeader,
	reqNode proto.NodeID, reqPriv *asymmetric.PrivateKey, reqPub *asymmetric.PublicKey,
) (r *types.SignedAckHeader, err error) {
	ack := &types.Ack{
		Header: types.SignedAckHeader{
			AckHeader: types.AckHeader{
				Response:  *resp,
				NodeID:    reqNode,
				Timestamp: time.Now().UTC(),
			},
			Signee:    reqPub,
			Signature: nil,
		},
	}

	if err = ack.Sign(reqPriv); err != nil {
		return
	}

	r = &ack.Header
	return
}

func createRandomQueryAck(
	reqNode proto.NodeID, reqPriv *asymmetric.PrivateKey, reqPub *asymmetric.PublicKey,
	respNode proto.NodeID, respPriv *asymmetric.PrivateKey, respPub *asymmetric.PublicKey,
) (r *types.SignedAckHeader, err error) {
	resp, err := createRandomQueryResponse(reqNode, reqPriv, reqPub, respNode, respPriv, respPub)

	if err != nil {
		return
	}

	ack := &types.Ack{
		Header: types.SignedAckHeader{
			AckHeader: types.AckHeader{
				Response:  *resp,
				NodeID:    reqNode,
				Timestamp: time.Now().UTC(),
			},
			Signee:    reqPub,
			Signature: nil,
		},
	}

	if err = ack.Sign(reqPriv); err != nil {
		return
	}

	r = &ack.Header
	return
}

func createRandomBlock(parent hash.Hash, isGenesis bool) (b *Block, err error) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &Block{
		SignedHeader: SignedHeader{
			Header: Header{
				Version:     0x01000000,
				Producer:    proto.NodeID(h.String()),
				GenesisHash: rootHash,
				ParentHash:  parent,
				Timestamp:   time.Now().UTC(),
			},
			Signee:    pub,
			Signature: nil,
		},
		Queries: make([]*hash.Hash, rand.Intn(10)+10),
	}

	for i := range b.Queries {
		b.Queries[i] = new(hash.Hash)
		rand.Read(b.Queries[i][:])
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
		err = kms.SetPublicKey(proto.NodeID(id.String()), nonce.Nonce, pub)

		if err != nil {
			return nil, err
		}
	}

	err = b.PackAndSignBlock(priv)

	if err != nil {
		return nil, err
	}

	return
}

func TestMain(m *testing.M) {
	testSetup()
	os.Exit(m.Run())
}
