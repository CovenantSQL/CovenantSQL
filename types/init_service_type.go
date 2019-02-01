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
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

//go:generate hsp

// InitService defines worker service init request.
type InitService struct {
	proto.Envelope
}

// ResourceMeta defines single database resource meta.
type ResourceMeta struct {
	TargetMiners           []proto.AccountAddress // designated miners
	Node                   uint16                 // reserved node count
	Space                  uint64                 // reserved storage space in bytes
	Memory                 uint64                 // reserved memory in bytes
	LoadAvgPerCPU          float64                // max loadAvg15 per CPU
	EncryptionKey          string                 // encryption key for database instance
	UseEventualConsistency bool                   // use eventual consistency replication if enabled
	ConsistencyLevel       float64                // customized strong consistency level
	IsolationLevel         int                    // customized isolation level
}

// ServiceInstance defines single instance to be initialized.
type ServiceInstance struct {
	DatabaseID   proto.DatabaseID
	Peers        *proto.Peers
	ResourceMeta ResourceMeta
	GenesisBlock *Block
}

// InitServiceResponseHeader defines worker service init response header.
type InitServiceResponseHeader struct {
	Instances []ServiceInstance
}

// SignedInitServiceResponseHeader defines signed worker service init response header.
type SignedInitServiceResponseHeader struct {
	InitServiceResponseHeader
	verifier.DefaultHashSignVerifierImpl
}

// InitServiceResponse defines worker service init response.
type InitServiceResponse struct {
	Header SignedInitServiceResponseHeader
}

// Verify checks hash and signature in init service response header.
func (sh *SignedInitServiceResponseHeader) Verify() (err error) {
	return sh.DefaultHashSignVerifierImpl.Verify(&sh.InitServiceResponseHeader)
}

// Sign the request.
func (sh *SignedInitServiceResponseHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	return sh.DefaultHashSignVerifierImpl.Sign(&sh.InitServiceResponseHeader, signer)
}

// Verify checks hash and signature in init service response header.
func (rs *InitServiceResponse) Verify() error {
	return rs.Header.Verify()
}

// Sign the request.
func (rs *InitServiceResponse) Sign(signer *asymmetric.PrivateKey) (err error) {
	// sign
	return rs.Header.Sign(signer)
}
