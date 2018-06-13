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
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/crypto/signature"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/types"
)

// NoAckReportHeader defines worker issued client no ack report.
type NoAckReportHeader struct {
	NodeID    proto.NodeID // reporter node id
	Timestamp time.Time    // time in UTC zone
	Request   SignedRequestHeader
	Response  SignedResponseHeader
}

// SignedNoAckReportHeader defines worker worker issued/signed client no ack report.
type SignedNoAckReportHeader struct {
	NoAckReportHeader
	HeaderHash hash.Hash
	Signee     *signature.PublicKey
	Signature  *signature.Signature
}

// NoAckReport defines whole worker no client ack report.
type NoAckReport struct {
	Header SignedNoAckReportHeader
}

// AggrNoAckReportHeader defines worker leader aggregated client no ack report.
type AggrNoAckReportHeader struct {
	NodeID    proto.NodeID              // aggregated report node id
	Timestamp time.Time                 // time in UTC zone
	Request   SignedRequestHeader       // original request
	Reports   []SignedNoAckReportHeader // no-ack reports
	Peers     kayak.Peers               // serving peers during report
}

// SignedAggrNoAckReportHeader defines worker leader aggregated/signed client no ack report.
type SignedAggrNoAckReportHeader struct {
	AggrNoAckReportHeader
	HeaderHash hash.Hash
	Signee     *signature.PublicKey
	Signature  *signature.Signature
}

// AggrNoAckReport defines whole worker leader no client ack report.
type AggrNoAckReport struct {
	Header SignedAggrNoAckReportHeader
}

func (h *NoAckReportHeader) toPB() *types.NoQueryAckReportHeader {
	return &types.NoQueryAckReportHeader{
		NodeID:    &types.NodeID{NodeID: string(h.NodeID)},
		Timestamp: timeToTimestamp(h.Timestamp),
		Request:   h.Request.toPB(),
		Response:  h.Response.toPB(),
	}
}

func (h *NoAckReportHeader) fromPB(pbh *types.NoQueryAckReportHeader) (err error) {
	if pbh == nil {
		return
	}
	h.NodeID = proto.NodeID(pbh.GetNodeID().GetNodeID())
	h.Timestamp = timeFromTimestamp(pbh.GetTimestamp())
	if err = h.Request.fromPB(pbh.GetRequest()); err != nil {
		return
	}
	return h.Response.fromPB(pbh.GetResponse())
}

func (h *NoAckReportHeader) marshal() ([]byte, error) {
	return pb.Marshal(h.toPB())
}

func (h *NoAckReportHeader) unmarshal(buffer []byte) (err error) {
	pbh := new(types.NoQueryAckReportHeader)
	if err = pb.Unmarshal(buffer, pbh); err != nil {
		return
	}
	return h.fromPB(pbh)
}

func (sh *SignedNoAckReportHeader) toPB() *types.SignedNoQueryAckReportHeader {
	return &types.SignedNoQueryAckReportHeader{
		Header:     sh.NoAckReportHeader.toPB(),
		HeaderHash: hashToPB(&sh.HeaderHash),
		Signee:     publicKeyToPB(sh.Signee),
		Signature:  signatureToPB(sh.Signature),
	}
}

func (sh *SignedNoAckReportHeader) fromPB(pbsh *types.SignedNoQueryAckReportHeader) (err error) {
	if pbsh == nil {
		return
	}
	if err = sh.NoAckReportHeader.fromPB(pbsh.GetHeader()); err != nil {
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

func (sh *SignedNoAckReportHeader) marshal() ([]byte, error) {
	return pb.Marshal(sh.toPB())
}

func (sh *SignedNoAckReportHeader) unmarshal(buffer []byte) (err error) {
	pbsh := new(types.SignedNoQueryAckReportHeader)
	if err = pb.Unmarshal(buffer, pbsh); err != nil {
		return
	}
	return sh.fromPB(pbsh)
}

// Verify checks hash and signature in signed no ack report header.
func (sh *SignedNoAckReportHeader) Verify() (err error) {
	// verify original request
	if err = sh.Request.Verify(); err != nil {
		return
	}
	// verify original response
	if err = sh.Response.Verify(); err != nil {
		return
	}
	// verify hash
	if err = verifyHash(&sh.NoAckReportHeader, &sh.HeaderHash); err != nil {
		return
	}
	// validate signature
	if !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

func (r *NoAckReport) toPB() *types.NoQueryAckReport {
	return &types.NoQueryAckReport{
		Header: r.Header.toPB(),
	}
}

func (r *NoAckReport) fromPB(pbr *types.NoQueryAckReport) (err error) {
	if pbr == nil {
		return
	}
	return r.Header.fromPB(pbr.GetHeader())
}

func (r *NoAckReport) marshal() ([]byte, error) {
	return pb.Marshal(r.toPB())
}

func (r *NoAckReport) unmarshal(buffer []byte) (err error) {
	pbr := new(types.NoQueryAckReport)
	if err = pb.Unmarshal(buffer, pbr); err != nil {
		return
	}
	return r.fromPB(pbr)
}

// Verify checks hash and signature in whole no ack report.
func (r *NoAckReport) Verify() error {
	return r.Header.Verify()
}

func (h *AggrNoAckReportHeader) toPB() *types.AggrNoQueryAckReportHeader {
	return &types.AggrNoQueryAckReportHeader{
		NodeID:    &types.NodeID{NodeID: string(h.NodeID)},
		Timestamp: timeToTimestamp(h.Timestamp),
		Request:   h.Request.toPB(),
		Reports: func(reports []SignedNoAckReportHeader) []*types.SignedNoQueryAckReportHeader {
			res := make([]*types.SignedNoQueryAckReportHeader, 0, len(reports))

			for _, report := range reports {
				res = append(res, report.toPB())
			}

			return res
		}(h.Reports),
		// TODO peers
	}
}

func (h *AggrNoAckReportHeader) fromPB(pbh *types.AggrNoQueryAckReportHeader) (err error) {
	if pbh == nil {
		return
	}
	h.NodeID = proto.NodeID(pbh.GetNodeID().GetNodeID())
	h.Timestamp = timeFromTimestamp(pbh.GetTimestamp())
	if err = h.Request.fromPB(pbh.GetRequest()); err != nil {
		return
	}
	h.Reports = make([]SignedNoAckReportHeader, len(pbh.GetReports()))
	for i, r := range pbh.GetReports() {
		if err = h.Reports[i].fromPB(r); err != nil {
			return
		}
	}
	// TODO peers
	return
}

func (h *AggrNoAckReportHeader) marshal() ([]byte, error) {
	return pb.Marshal(h.toPB())
}

func (h *AggrNoAckReportHeader) unmarshal(buffer []byte) (err error) {
	pbh := new(types.AggrNoQueryAckReportHeader)
	if err = pb.Unmarshal(buffer, pbh); err != nil {
		return
	}
	return h.fromPB(pbh)
}

func (sh *SignedAggrNoAckReportHeader) toPB() *types.SignedAggrNoQueryAckReportHeader {
	return &types.SignedAggrNoQueryAckReportHeader{
		Header:     sh.AggrNoAckReportHeader.toPB(),
		HeaderHash: hashToPB(&sh.HeaderHash),
		Signee:     publicKeyToPB(sh.Signee),
		Signature:  signatureToPB(sh.Signature),
	}
}

func (sh *SignedAggrNoAckReportHeader) fromPB(pbsh *types.SignedAggrNoQueryAckReportHeader) (err error) {
	if pbsh == nil {
		return
	}
	if err = sh.AggrNoAckReportHeader.fromPB(pbsh.GetHeader()); err != nil {
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

func (sh *SignedAggrNoAckReportHeader) marshal() ([]byte, error) {
	return pb.Marshal(sh.toPB())
}

func (sh *SignedAggrNoAckReportHeader) unmarshal(buffer []byte) (err error) {
	pbsh := new(types.SignedAggrNoQueryAckReportHeader)
	if err = pb.Unmarshal(buffer, pbsh); err != nil {
		return
	}
	return sh.fromPB(pbsh)
}

// Verify checks hash and signature in aggregated no ack report.
func (sh *SignedAggrNoAckReportHeader) Verify() (err error) {
	// verify original request
	if err = sh.AggrNoAckReportHeader.Request.Verify(); err != nil {
		return
	}
	// verify original reports
	for _, r := range sh.Reports {
		if err = r.Verify(); err != nil {
			return
		}
	}
	// verify hash
	if err = verifyHash(&sh.AggrNoAckReportHeader, &sh.HeaderHash); err != nil {
		return
	}
	// verify signature
	if !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

func (r *AggrNoAckReport) toPB() *types.SignedAggrNoQueryAckReport {
	return &types.SignedAggrNoQueryAckReport{
		Header: r.Header.toPB(),
	}
}

func (r *AggrNoAckReport) fromPB(pbr *types.SignedAggrNoQueryAckReport) (err error) {
	if pbr == nil {
		return
	}
	return r.Header.fromPB(pbr.GetHeader())
}

func (r *AggrNoAckReport) marshal() ([]byte, error) {
	return pb.Marshal(r.toPB())
}

func (r *AggrNoAckReport) unmarshal(buffer []byte) (err error) {
	pbr := new(types.SignedAggrNoQueryAckReport)
	if err = pb.Unmarshal(buffer, pbr); err != nil {
		return
	}
	return r.fromPB(pbr)
}

// Verify the whole aggregation no ack report.
func (r *AggrNoAckReport) Verify() (err error) {
	return r.Header.Verify()
}
