/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the “License”);
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an “AS IS” BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package worker

import (
	"time"

	pb "github.com/golang/protobuf/proto"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/signature"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/types"
)

// QueryType enumerates available query type, currently read/write.
type QueryType int32

const (
	// ReadQuery defines a read query type.
	ReadQuery QueryType = iota
	// WriteQuery defines a write query type.
	WriteQuery
)

// RequestPayload defines a queries payload.
type RequestPayload struct {
	Queries []string
}

// RequestHeader defines a query request header.
type RequestHeader struct {
	QueryType    QueryType
	NodeID       proto.NodeID // request node id
	ConnectionID uint64
	SeqNo        uint64
	Timestamp    time.Time // time in UTC zone
	BatchCount   uint64    // query count in this request
	QueriesHash  hash.Hash // hash of query payload
}

// SignedRequestHeader defines a signed query request header.
type SignedRequestHeader struct {
	RequestHeader
	HeaderHash hash.Hash
	Signee     *signature.PublicKey
	Signature  *signature.Signature
}

// Request defines a complete query request.
type Request struct {
	Header  SignedRequestHeader
	Payload RequestPayload
}

func (p *RequestPayload) toPB() *types.QueryRequestPayload {
	return &types.QueryRequestPayload{
		Queries: p.Queries,
	}
}

func (p *RequestPayload) fromPB(pbp *types.QueryRequestPayload) (err error) {
	if pbp == nil {
		return
	}

	copy(p.Queries, pbp.GetQueries())
	return
}

func (p *RequestPayload) marshal() ([]byte, error) {
	return pb.Marshal(p.toPB())
}

func (p *RequestPayload) unmarshal(buffer []byte) (err error) {
	pbRp := new(types.QueryRequestPayload)
	err = pb.Unmarshal(buffer, pbRp)
	if err != nil {
		return err
	}

	return p.fromPB(pbRp)
}

func (h *RequestHeader) toPB() *types.QueryRequestHeader {
	return &types.QueryRequestHeader{
		QueryType:    types.QueryType(h.QueryType),
		NodeID:       &types.NodeID{NodeID: string(h.NodeID)},
		ConnectionID: h.ConnectionID,
		SeqNo:        h.SeqNo,
		Timestamp:    timeToTimestamp(h.Timestamp),
		BatchCount:   h.BatchCount,
		QueriesHash:  hashToPB(&h.QueriesHash),
	}
}

func (h *RequestHeader) fromPB(pbh *types.QueryRequestHeader) (err error) {
	if pbh == nil {
		return
	}

	h.QueryType = QueryType(pbh.GetQueryType())
	h.NodeID = proto.NodeID(pbh.GetNodeID().GetNodeID())
	h.ConnectionID = pbh.GetConnectionID()
	h.SeqNo = pbh.GetSeqNo()
	h.Timestamp = timeFromTimestamp(pbh.GetTimestamp())
	h.BatchCount = pbh.GetBatchCount()
	return hashFromPB(pbh.GetQueriesHash(), &h.QueriesHash)
}

func (h *RequestHeader) marshal() ([]byte, error) {
	return pb.Marshal(h.toPB())
}

func (h *RequestHeader) unmarshal(buffer []byte) (err error) {
	pbh := new(types.QueryRequestHeader)
	if err = pb.Unmarshal(buffer, pbh); err != nil {
		return
	}
	return h.fromPB(pbh)
}

func (sh *SignedRequestHeader) toPB() *types.SignedQueryRequestHeader {
	return &types.SignedQueryRequestHeader{
		Header:     sh.RequestHeader.toPB(),
		HeaderHash: hashToPB(&sh.HeaderHash),
		Signee:     publicKeyToPB(sh.Signee),
		Signature:  signatureToPB(sh.Signature),
	}
}

func (sh *SignedRequestHeader) fromPB(pbsh *types.SignedQueryRequestHeader) (err error) {
	if pbsh == nil {
		return
	}
	if err = sh.RequestHeader.fromPB(pbsh.GetHeader()); err != nil {
		return
	}
	if err = hashFromPB(pbsh.GetHeaderHash(), &sh.HeaderHash); err != nil {
		return
	}
	if sh.Signee, err = publicKeyFromPB(pbsh.GetSignee()); err != nil {
		return
	}
	if sh.Signature, err = signatureFromPB(pbsh.GetSignature()); err != nil {
		return
	}
	return
}

func (sh *SignedRequestHeader) marshal() ([]byte, error) {
	return pb.Marshal(sh.toPB())
}

func (sh *SignedRequestHeader) unmarshal(buffer []byte) (err error) {
	pbsh := new(types.SignedQueryRequestHeader)
	if err = pb.Unmarshal(buffer, pbsh); err != nil {
		return
	}
	return sh.fromPB(pbsh)
}

// Verify checks hash and signature in request header.
func (sh *SignedRequestHeader) Verify() (err error) {
	// verify hash
	if err = verifyHash(&sh.RequestHeader, &sh.HeaderHash); err != nil {
		return
	}
	// verify sign
	if !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}
	return nil
}

func (r *Request) toPB() *types.QueryRequest {
	return &types.QueryRequest{
		Header:  r.Header.toPB(),
		Payload: r.Payload.toPB(),
	}
}

func (r *Request) fromPB(pbr *types.QueryRequest) (err error) {
	if pbr == nil {
		return
	}
	if err = r.Header.fromPB(pbr.GetHeader()); err != nil {
		return
	}
	if err = r.Payload.fromPB(pbr.GetPayload()); err != nil {
		return
	}
	return
}

func (r *Request) marshal() ([]byte, error) {
	return pb.Marshal(r.toPB())
}

func (r *Request) unmarshal(buffer []byte) (err error) {
	pbr := new(types.QueryRequest)
	if err = pb.Unmarshal(buffer, pbr); err != nil {
		return
	}
	return r.fromPB(pbr)
}

// Verify checks hash and signature in whole request.
func (r *Request) Verify() (err error) {
	// verify payload hash in signed header
	if err = verifyHash(&r.Payload, &r.Header.QueriesHash); err != nil {
		return
	}
	// verify header sign
	return r.Header.Verify()
}
