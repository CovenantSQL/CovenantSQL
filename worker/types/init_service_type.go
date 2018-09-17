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
	"bytes"
	"encoding/binary"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/proto"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

//go:generate hsp

// InitService defines worker service init request.
type InitService struct {
	proto.Envelope
}

// ResourceMeta defines single database resource meta.
type ResourceMeta struct {
	Node          uint16 // reserved node count
	Space         uint64 // reserved storage space in bytes
	Memory        uint64 // reserved memory in bytes
	LoadAvgPerCPU uint64 // max loadAvg15 per CPU
	EncryptionKey string `hspack:"-"` // encryption key for database instance
}

// ServiceInstance defines single instance to be initialized.
type ServiceInstance struct {
	DatabaseID   proto.DatabaseID
	Peers        *kayak.Peers
	ResourceMeta ResourceMeta
	GenesisBlock *ct.Block
}

// InitServiceResponseHeader defines worker service init response header.
type InitServiceResponseHeader struct {
	Instances []ServiceInstance
}

// SignedInitServiceResponseHeader defines signed worker service init response header.
type SignedInitServiceResponseHeader struct {
	InitServiceResponseHeader
	HeaderHash hash.Hash
	Signee     *asymmetric.PublicKey
	Signature  *asymmetric.Signature
}

// InitServiceResponse defines worker service init response.
type InitServiceResponse struct {
	Header SignedInitServiceResponseHeader
}

// Serialize structure to bytes.
func (m *ResourceMeta) Serialize() []byte {
	if m == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, m.Node)
	binary.Write(buf, binary.LittleEndian, m.Space)
	binary.Write(buf, binary.LittleEndian, m.Memory)

	return buf.Bytes()
}

// Serialize structure to bytes.
func (i *ServiceInstance) Serialize() []byte {
	if i == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	buf.WriteString(string(i.DatabaseID))
	buf.Write(i.Peers.Serialize())
	buf.Write(i.ResourceMeta.Serialize())
	if i.GenesisBlock != nil {
		genesisBlock, _ := utils.EncodeMsgPack(i.GenesisBlock)
		buf.Write(genesisBlock.Bytes())
	} else {
		buf.Write([]byte{'\000'})
	}

	return buf.Bytes()
}

// Serialize structure to bytes.
func (h *InitServiceResponseHeader) Serialize() []byte {
	if h == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint64(len(h.Instances)))
	for _, instance := range h.Instances {
		buf.Write(instance.Serialize())
	}

	return buf.Bytes()
}

// Serialize structure to bytes.
func (sh *SignedInitServiceResponseHeader) Serialize() []byte {
	if sh == nil {
		return []byte{'\000'}
	}

	buf := new(bytes.Buffer)

	buf.Write(sh.InitServiceResponseHeader.Serialize())
	buf.Write(sh.HeaderHash[:])
	if sh.Signee != nil {
		buf.Write(sh.Signee.Serialize())
	} else {
		buf.WriteRune('\000')
	}
	if sh.Signature != nil {
		buf.Write(sh.Signature.Serialize())
	} else {
		buf.WriteRune('\000')
	}

	return buf.Bytes()
}

// Verify checks hash and signature in init service response header.
func (sh *SignedInitServiceResponseHeader) Verify() (err error) {
	// verify hash
	if err = verifyHash(&sh.InitServiceResponseHeader, &sh.HeaderHash); err != nil {
		return
	}
	// verify sign
	if sh.Signee == nil || sh.Signature == nil || !sh.Signature.Verify(sh.HeaderHash[:], sh.Signee) {
		return ErrSignVerification
	}
	return
}

// Sign the request.
func (sh *SignedInitServiceResponseHeader) Sign(signer *asymmetric.PrivateKey) (err error) {
	// build hash
	buildHash(&sh.InitServiceResponseHeader, &sh.HeaderHash)

	// sign
	sh.Signature, err = signer.Sign(sh.HeaderHash[:])

	return
}

// Serialize structure to bytes.
func (rs *InitServiceResponse) Serialize() []byte {
	if rs == nil {
		return []byte{'\000'}
	}

	return rs.Header.Serialize()
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
