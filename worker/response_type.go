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

// ResponseRow defines single row of query response.
type ResponseRow struct {
	Values []interface{}
}

// ResponsePayload defines column names and rows of query response.
type ResponsePayload struct {
	Columns []string
	Rows    []ResponseRow
}

// ResponseHeader defines a query response header.
type ResponseHeader struct {
	Request           SignedRequestHeader
	NodeID            proto.NodeID // response node id
	Timestamp         time.Time    // time in UTC zone
	RowCount          uint64       // response row count of payload
	AffectedRowsCount uint64       // response affected rows count
	DataHash          hash.Hash    // hash of query response
}

// SignedResponseHeader defines a signed query response header.
type SignedResponseHeader struct {
	Header     ResponseHeader
	HeaderHash hash.Hash
	Signee     *signature.PublicKey
	Signature  *signature.Signature
}

// Response defines a complete query response.
type Response struct {
	Header  SignedResponseHeader
	Payload ResponsePayload
}

func (h *ResponseHeader) toPB() *types.QueryResponseHeader {
	return &types.QueryResponseHeader{
		Request:           h.Request.toPB(),
		NodeID:            &types.NodeID{NodeID: string(h.NodeID)},
		Timestamp:         timeToTimestamp(h.Timestamp),
		RowCount:          h.RowCount,
		AffectedRowsCount: h.AffectedRowsCount,
		DataHash:          hashToPB(&h.DataHash),
	}
}

func (h *ResponseHeader) fromPB(pbh *types.QueryResponseHeader) (err error) {
	if pbh == nil {
		return
	}

	if err = h.Request.fromPB(pbh.GetRequest()); err != nil {
		return
	}
	h.NodeID = proto.NodeID(pbh.GetNodeID().GetNodeID())
	h.Timestamp = timeFromTimestamp(pbh.GetTimestamp())
	h.RowCount = pbh.GetRowCount()
	h.AffectedRowsCount = pbh.GetAffectedRowsCount()
	return hashFromPB(pbh.GetDataHash(), &h.DataHash)
}

func (h *ResponseHeader) marshal() ([]byte, error) {
	return pb.Marshal(h.toPB())
}

func (h *ResponseHeader) unmarshal(buffer []byte) (err error) {
	pbh := new(types.QueryResponseHeader)
	if err = pb.Unmarshal(buffer, pbh); err != nil {
		return
	}

	return h.fromPB(pbh)
}

func (sh *SignedResponseHeader) toPB() *types.SignedQueryResponseHeader {
	return &types.SignedQueryResponseHeader{
		Header:     sh.Header.toPB(),
		HeaderHash: hashToPB(&sh.HeaderHash),
		Signee:     publicKeyToPB(sh.Signee),
		Signature:  signatureToPB(sh.Signature),
	}
}

func (sh *SignedResponseHeader) fromPB(pbsh *types.SignedQueryResponseHeader) (err error) {
	if pbsh == nil {
		return
	}

	if err = sh.Header.fromPB(pbsh.GetHeader()); err != nil {
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

func (sh *SignedResponseHeader) marshal() ([]byte, error) {
	return pb.Marshal(sh.toPB())
}

func (sh *SignedResponseHeader) unmarshal(buffer []byte) (err error) {
	pbsh := new(types.SignedQueryResponseHeader)
	if err = pb.Unmarshal(buffer, pbsh); err != nil {
		return
	}

	return sh.fromPB(pbsh)
}

// Verify checks hash and signature in response header.
func (sh *SignedResponseHeader) Verify() (err error) {
	// verify original request header
	if err = sh.Header.Request.Verify(); err != nil {
		return
	}
	// verify hash
	if err = verifyHash(&sh.Header, &sh.HeaderHash); err != nil {
		return
	}
	// verify signature
	if !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}

	return nil
}
