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

// AckHeader defines client ack entity.
type AckHeader struct {
	Response  SignedResponseHeader
	NodeID    proto.NodeID // ack node id
	Timestamp time.Time    // time in UTC zone
}

// SignedAckHeader defines client signed ack entity.
type SignedAckHeader struct {
	Header     AckHeader
	HeaderHash hash.Hash
	Signee     *signature.PublicKey
	Signature  *signature.Signature
}

// Ack defines a whole client ack request entity.
type Ack struct {
	Header SignedAckHeader
}

func (h *AckHeader) toPB() *types.QueryAckHeader {
	return &types.QueryAckHeader{
		Response:  h.Response.toPB(),
		NodeID:    &types.NodeID{NodeID: string(h.NodeID)},
		Timestamp: h.Timestamp.UnixNano(),
	}
}

func (h *AckHeader) fromPB(pbh *types.QueryAckHeader) (err error) {
	if pbh == nil {
		return
	}
	if err = h.Response.fromPB(pbh.GetResponse()); err != nil {
		return
	}
	h.NodeID = proto.NodeID(pbh.GetNodeID().GetNodeID())
	h.Timestamp = timeFromTimestamp(pbh.GetTimestamp())
	return
}

func (h *AckHeader) marshal() ([]byte, error) {
	return pb.Marshal(h.toPB())
}

func (h *AckHeader) unmarshal(buffer []byte) (err error) {
	pbh := new(types.QueryAckHeader)
	if err = pb.Unmarshal(buffer, pbh); err != nil {
		return
	}
	return h.fromPB(pbh)
}

func (sh *SignedAckHeader) toPB() *types.SignedQueryAckHeader {
	return &types.SignedQueryAckHeader{
		Header:     sh.Header.toPB(),
		HeaderHash: hashToPB(&sh.HeaderHash),
		Signee:     publicKeyToPB(sh.Signee),
		Signature:  signatureToPB(sh.Signature),
	}
}

func (sh *SignedAckHeader) fromPB(pbsh *types.SignedQueryAckHeader) (err error) {
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

func (sh *SignedAckHeader) marshal() ([]byte, error) {
	return pb.Marshal(sh.toPB())
}

func (sh *SignedAckHeader) unmarshal(buffer []byte) (err error) {
	pbsh := new(types.SignedQueryAckHeader)
	if err = pb.Unmarshal(buffer, pbsh); err != nil {
		return
	}
	return sh.fromPB(pbsh)
}

// Verify checks hash and signature in ack header.
func (sh *SignedAckHeader) Verify() (err error) {
	// verify response
	if err = sh.Header.Response.Verify(); err != nil {
		return
	}
	if err = verifyHash(&sh.Header, &sh.HeaderHash); err != nil {
		return
	}
	// verify sign
	if !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

func (a *Ack) toPB() *types.QueryAck {
	return &types.QueryAck{
		Header: a.Header.toPB(),
	}
}

func (a *Ack) fromPB(pba *types.QueryAck) (err error) {
	if pba == nil {
		return
	}
	return a.Header.fromPB(pba.GetHeader())
}

func (a *Ack) marshal() ([]byte, error) {
	return pb.Marshal(a.toPB())
}

func (a *Ack) unmarshal(buffer []byte) (err error) {
	pba := new(types.QueryAck)
	if err = pb.Unmarshal(buffer, pba); err != nil {
		return
	}
	return a.fromPB(pba)
}

// Verify checks hash and signature in ack.
func (a *Ack) Verify() error {
	return a.Header.Verify()
}
