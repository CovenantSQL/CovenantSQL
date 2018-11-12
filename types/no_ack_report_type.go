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

package types

import (
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// NoAckReportHeader defines worker issued client no ack report.
type NoAckReportHeader struct {
	NodeID    proto.NodeID // reporter node id
	Timestamp time.Time    // time in UTC zone
	Response  SignedResponseHeader
}

// SignedNoAckReportHeader defines worker worker issued/signed client no ack report.
type SignedNoAckReportHeader struct {
	NoAckReportHeader
	Hash      hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// NoAckReport defines whole worker no client ack report.
type NoAckReport struct {
	proto.Envelope
	Header SignedNoAckReportHeader
}

// AggrNoAckReportHeader defines worker leader aggregated client no ack report.
type AggrNoAckReportHeader struct {
	NodeID    proto.NodeID              // aggregated report node id
	Timestamp time.Time                 // time in UTC zone
	Reports   []SignedNoAckReportHeader // no-ack reports
	Peers     *proto.Peers              // serving peers during report
}

// SignedAggrNoAckReportHeader defines worker leader aggregated/signed client no ack report.
type SignedAggrNoAckReportHeader struct {
	AggrNoAckReportHeader
	Hash      hash.Hash
	Signee    *asymmetric.PublicKey
	Signature *asymmetric.Signature
}

// AggrNoAckReport defines whole worker leader no client ack report.
type AggrNoAckReport struct {
	proto.Envelope
	Header SignedAggrNoAckReportHeader
}

// Verify checks hash and signature in signed no ack report header.
func (sh *SignedNoAckReportHeader) Verify() (err error) {
	// verify original response
	if err = sh.Response.Verify(); err != nil {
		return
	}
	// verify hash
	if err = verifyHash(&sh.NoAckReportHeader, &sh.Hash); err != nil {
		return
	}
	// validate signature
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.Hash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

// Sign the request.
func (sh *SignedNoAckReportHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// verify original response
	if err = sh.Response.Verify(); err != nil {
		return
	}

	// build hash
	if err = buildHash(&sh.NoAckReportHeader, &sh.Hash); err != nil {
		return
	}

	// sign
	sh.Signature, err = signer.Sign(sh.Hash[:])
	sh.Signee = signer.PubKey()

	return
}

// Verify checks hash and signature in whole no ack report.
func (r *NoAckReport) Verify() error {
	return r.Header.Verify()
}

// Sign the request.
func (r *NoAckReport) Sign(signer *asymmetric.PrivateKey) error {
	return r.Header.Sign(signer)
}

// Verify checks hash and signature in aggregated no ack report.
func (sh *SignedAggrNoAckReportHeader) Verify() (err error) {
	// verify original reports
	for _, r := range sh.Reports {
		if err = r.Verify(); err != nil {
			return
		}
	}
	// verify hash
	if err = verifyHash(&sh.AggrNoAckReportHeader, &sh.Hash); err != nil {
		return
	}
	// verify signature
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.Hash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

// Sign the request.
func (sh *SignedAggrNoAckReportHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	for _, r := range sh.Reports {
		if err = r.Verify(); err != nil {
			return
		}
	}

	// verify hash
	if err = buildHash(&sh.AggrNoAckReportHeader, &sh.Hash); err != nil {
		return
	}

	// verify signature
	sh.Signature, err = signer.Sign(sh.Hash[:])
	sh.Signee = signer.PubKey()

	return
}

// Verify the whole aggregation no ack report.
func (r *AggrNoAckReport) Verify() (err error) {
	return r.Header.Verify()
}

// Sign the request.
func (r *AggrNoAckReport) Sign(signer *asymmetric.PrivateKey) error {
	return r.Header.Sign(signer)
}
