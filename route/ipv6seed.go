/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package route

import (
	"fmt"

	"github.com/CovenantSQL/beacon/ipv6"
	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

const (
	// ID is node id
	ID = "id."
	// PUBKEY is public key
	PUBKEY = "pub."
	// NONCE is nonce
	NONCE = "n."
	// ADDR is address
	ADDR = "addr."
)

// IPv6SeedClient is IPv6 DNS seed client
type IPv6SeedClient struct{}

// GetBPFromDNSSeed gets BP info from the IPv6 domain
func (isc *IPv6SeedClient) GetBPFromDNSSeed(BPDomain string) (BPNodes IDNodeMap, err error) {
	// Public key
	pubKeyBuf := make([]byte, asymmetric.PublicKeyBytesLen)
	pubKeyBuf[0] = asymmetric.PublicKeyFormatHeader
	var pubBuf, nonceBuf, addrBuf []byte
	if pubBuf, err = ipv6.FromDomain(PUBKEY + BPDomain); err != nil {
		return
	}
	if len(pubBuf) != asymmetric.PublicKeyBytesLen-1 {
		return nil, errors.Errorf("error public key bytes len: %d", len(pubBuf))
	}
	copy(pubKeyBuf[1:], pubBuf)
	var pubKey asymmetric.PublicKey
	err = pubKey.UnmarshalBinary(pubKeyBuf)
	if err != nil {
		return
	}

	// Nonce
	if nonceBuf, err = ipv6.FromDomain(NONCE + BPDomain); err != nil {
		return
	}
	nonce, err := cpuminer.Uint256FromBytes(nonceBuf)
	if err != nil {
		return
	}

	// Addr
	addrBuf, err = ipv6.FromDomain(ADDR + BPDomain)
	if err != nil {
		return
	}
	addrBytes, err := crypto.RemovePKCSPadding(addrBuf)
	if err != nil {
		return
	}

	// NodeID
	nodeIDBuf, err := ipv6.FromDomain(ID + BPDomain)
	if err != nil {
		return
	}
	var nodeID proto.RawNodeID
	err = nodeID.SetBytes(nodeIDBuf)
	if err != nil {
		return
	}

	BPNodes = make(IDNodeMap)
	BPNodes[nodeID] = proto.Node{
		ID:        nodeID.ToNodeID(),
		Addr:      string(addrBytes),
		PublicKey: &pubKey,
		Nonce:     *nonce,
	}

	return
}

// GenBPIPv6 generates the IPv6 addrs contain BP info
func (isc *IPv6SeedClient) GenBPIPv6(node *proto.Node, domain string) (out string, err error) {
	// NodeID
	nodeIDIps, err := ipv6.ToIPv6(node.ID.ToRawNodeID().AsBytes())
	if err != nil {
		return "", err
	}
	for i, ip := range nodeIDIps {
		out += fmt.Sprintf("%02d.%s%s	1	IN	AAAA	%s\n", i, ID, domain, ip)
	}

	// Public key, with leading 1 byte type trimmed
	//  see: asymmetric.PublicKeyFormatHeader
	pubKeyIps, err := ipv6.ToIPv6(node.PublicKey.Serialize()[1:])
	if err != nil {
		return "", err
	}
	for i, ip := range pubKeyIps {
		out += fmt.Sprintf("%02d.%s%s	1	IN	AAAA	%s\n", i, PUBKEY, domain, ip)
	}

	// Nonce
	nonceIps, err := ipv6.ToIPv6(node.Nonce.Bytes())
	if err != nil {
		return "", err
	}
	for i, ip := range nonceIps {
		out += fmt.Sprintf("%02d.%s%s	1	IN	AAAA	%s\n", i, NONCE, domain, ip)
	}

	// Addr
	addrIps, err := ipv6.ToIPv6(crypto.AddPKCSPadding([]byte(node.Addr)))
	if err != nil {
		return "", err
	}
	for i, ip := range addrIps {
		out += fmt.Sprintf("%02d.%s%s	1	IN	AAAA	%s\n", i, ADDR, domain, ip)
	}

	return
}
