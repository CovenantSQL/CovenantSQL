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

package route

import (
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

/*
NOTE:
   	1. Node holds its own Private Key, Public Key is Registered to BP
	1. NodeID = Hash(NodePublicKey + Nonce)
	1. Every node holds BP's Public Key and gets other node's Public Key from BP
    1. PubKey is verified by ETLS with ECDH
	1. For more about ETLS: https://github.com/thunderdb/research/wiki/ETLS(Enhanced-Transport-Layer-Security)

ACLs:
	Client -> BP, Request for Allocating DB:
  		ACL: Open to world, add difficulty verification
   		Checking pledge should be done in the Server RPC implementation

   	BP -> Miner, Request for Creating DB:
  		ACL: Open to BP

   	Miner -> BP, Metric.UploadMetrics():
   		ACL: Open to Registered Miner

   	BP -> BP, Exchange NodeInfo, Kayak.Call():
  		ACL: Open to BP

   	Client -> Miner, SQL Query:
   		ACL: Open to Registered Client

   	* -> BP, DHT.Ping():
  		ACL: Open to world, add difficulty verification

   	* -> BP, DHT.FindNode(), DHT.FindNeighbor():
  		ACL: Open to world
*/

// RemoteFunc defines the RPC Call name
type RemoteFunc int

const (
	// DHTPing is for node info register to BP
	DHTPing RemoteFunc = iota
	// DHTFindNeighbor finds consistent hash neighbors
	DHTFindNeighbor
	// DHTFindNode gets node info
	DHTFindNode
	// MetricUploadMetrics uploads node metrics
	MetricUploadMetrics
	// KayakCall is used by BP, Miner for data consistency
	KayakCall
)

// String returns the RemoteFunc string
func (s RemoteFunc) String() string {
	switch s {
	case DHTPing:
		return "DHT.Ping"
	case DHTFindNeighbor:
		return "DHT.FindNeighbor"
	case DHTFindNode:
		return "DHT.FindNode"
	case MetricUploadMetrics:
		return "Metric.UploadMetrics"
	case KayakCall:
		return "Kayak.Call"
	}
	return "Unknown"
}

// IsPermitted returns if the node is permitted to call the RPC func
func IsPermitted(callerEnvelope *proto.Envelope, funcName RemoteFunc) (ok bool) {
	//FIXME(auxten,xq262144,leventeliu,lambda) collect all RPC call server side
	// implementations, add Envelope to the Request struct and call this func
	// to filter permission

	callerETLSNodeID := callerEnvelope.GetNodeID()
	// strict anonymous ETLS only used for Ping
	// the envelope node id is set at NodeAwareServerCodec and CryptoListener.CHandler
	// if callerETLSNodeID == nil here indicates that ETLS is not used, just ignore it
	if callerETLSNodeID != nil {
		if callerETLSNodeID.IsEqual(&kms.AnonymousRawNodeID.Hash) {
			if funcName != DHTPing {
				log.Warnf("anonymous ETLS connection can not used by %s", funcName)
				return false
			}
		}
	}

	if !IsBPNodeID(callerETLSNodeID) {
		// non BP
		switch funcName {
		case DHTPing, DHTFindNode, DHTFindNeighbor, MetricUploadMetrics:
			return true
		case KayakCall:
			return false
		default:
			// calling Unspecified RPC is forbidden
			return false
		}
	}

	// BP can call any RPC
	return true
}
