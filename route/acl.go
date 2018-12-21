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

package route

import (
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

/*
NOTE:
   	1. Node holds its own Private Key, Public Key is Registered to BP
	1. NodeID = Hash(NodePublicKey + Nonce)
	1. Every node holds BP's Public Key and gets other node's Public Key from BP
    1. PubKey is verified by ETLS with ECDH
	1. For more about ETLS: https://github.com/CovenantSQL/research/wiki/ETLS(Enhanced-Transport-Layer-Security)

ACLs:
	Client -> BP, Request for Allocating DB:
  		ACL: Open to world, add difficulty verification
   		Checking pledge should be done in the Server RPC implementation

   	BP -> Miner, Request for Creating DB:
  		ACL: Open to BP

   	Miner -> BP, Metric.UploadMetrics():
   		ACL: Open to Registered Miner

	Miner -> Miner, Kayak.Call():
		ACL: Open to Miner Leader.

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
	// KayakCall is used by BP for data consistency
	KayakCall
	// DBSQuery is used by client to read/write database
	DBSQuery
	// DBSAck is used by client to send acknowledge to the query response
	DBSAck
	// DBSDeploy is used by BP to create/drop/update database
	DBSDeploy
	// DBCCall is used by Miner for data consistency
	DBCCall
	// BPDBCreateDatabase is used by client to create database
	BPDBCreateDatabase
	// BPDBDropDatabase is used by client to drop database
	BPDBDropDatabase
	// BPDBGetDatabase is used by client to get database meta
	BPDBGetDatabase
	// BPDBGetNodeDatabases is used by miner to node residential databases
	BPDBGetNodeDatabases
	// SQLCAdviseNewBlock is used by sqlchain to advise new block between adjacent node
	SQLCAdviseNewBlock
	// SQLCAdviseBinLog is usd by sqlchain to advise binlog between adjacent node
	SQLCAdviseBinLog
	// SQLCAdviseAckedQuery is used by sqlchain to advice response query between adjacent node
	SQLCAdviseAckedQuery
	// SQLCFetchBlock is used by sqlchain to fetch block from adjacent nodes
	SQLCFetchBlock
	// SQLCSignBilling is used by sqlchain to response billing signature for periodic billing request
	SQLCSignBilling
	// SQLCLaunchBilling is used by blockproducer to trigger the billing process in sqlchain
	SQLCLaunchBilling
	// SQLCSubscribeTransactions is used by sqlchain to handle observer subscription request
	SQLCSubscribeTransactions
	// SQLCCancelSubscription is used by sqlchain to handle observer subscription cancellation request
	SQLCCancelSubscription
	// OBSAdviseNewBlock is used by sqlchain to push new block to observers
	OBSAdviseNewBlock
	// MCCAdviseNewBlock is used by block producer to push block to adjacent nodes
	MCCAdviseNewBlock
	// MCCAdviseTxBilling is used by block producer to push billing transaction to adjacent nodes
	MCCAdviseTxBilling
	// MCCAdviseBillingRequest is used by block producer to push billing request to adjacent nodes
	MCCAdviseBillingRequest
	// MCCFetchBlock is used by nodes to fetch block from block producer
	MCCFetchBlock
	// MCCFetchBlockByCount is used by nodes to fetch block from block producer by block count since genesis
	MCCFetchBlockByCount
	// MCCFetchTxBilling is used by nodes to fetch billing transaction from block producer
	MCCFetchTxBilling
	// MCCNextAccountNonce is used by block producer main chain to allocate next nonce for transactions
	MCCNextAccountNonce
	// MCCAddTx is used by block producer main chain to upload transaction
	MCCAddTx
	// MCCQueryAccountStableBalance is used by block producer to provide account stable coin balance
	MCCQueryAccountStableBalance
	// MCCQueryAccountCovenantBalance is used by block producer to provide account covenant coin balance
	MCCQueryAccountCovenantBalance

	// DHTRPCName defines the block producer dh-rpc service name
	DHTRPCName = "DHT"
	// BlockProducerRPCName defines main chain rpc name
	BlockProducerRPCName = "MCC"
	// SQLChainRPCName defines the sql chain rpc name
	SQLChainRPCName = "SQLC"
	// DBRPCName defines the sql chain db service rpc name
	DBRPCName = "DBS"
	// BPDBRPCName defines the block producer db service rpc name
	BPDBRPCName = "BPDB"
	// ObserverRPCName defines the observer node service rpc name
	ObserverRPCName = "OBS"
)

// String returns the RemoteFunc string.
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
	case DBSQuery:
		return "DBS.Query"
	case DBSAck:
		return "DBS.Ack"
	case DBSDeploy:
		return "DBS.Deploy"
	case DBCCall:
		return "DBC.Call"
	case BPDBCreateDatabase:
		return "BPDB.CreateDatabase"
	case BPDBDropDatabase:
		return "BPDB.DropDatabase"
	case BPDBGetDatabase:
		return "BPDB.GetDatabase"
	case BPDBGetNodeDatabases:
		return "BPDB.GetNodeDatabases"
	case SQLCAdviseNewBlock:
		return "SQLC.AdviseNewBlock"
	case SQLCAdviseBinLog:
		return "SQLC.AdviseBinLog"
	case SQLCAdviseAckedQuery:
		return "SQLC.AdviseAckedQuery"
	case SQLCFetchBlock:
		return "SQLC.FetchBlock"
	case SQLCSignBilling:
		return "SQLC.SignBilling"
	case SQLCLaunchBilling:
		return "SQLC.LaunchBilling"
	case SQLCSubscribeTransactions:
		return "SQLC.SubscribeTransactions"
	case SQLCCancelSubscription:
		return "SQLC.CancelSubscription"
	case OBSAdviseNewBlock:
		return "OBS.AdviseNewBlock"
	case MCCAdviseNewBlock:
		return "MCC.AdviseNewBlock"
	case MCCAdviseTxBilling:
		return "MCC.AdviseTxBilling"
	case MCCAdviseBillingRequest:
		return "MCC.AdviseBillingRequest"
	case MCCFetchBlock:
		return "MCC.FetchBlock"
	case MCCFetchBlockByCount:
		return "MCC.FetchBlockByCount"
	case MCCFetchTxBilling:
		return "MCC.FetchTxBilling"
	case MCCNextAccountNonce:
		return "MCC.NextAccountNonce"
	case MCCAddTx:
		return "MCC.AddTx"
	case MCCQueryAccountStableBalance:
		return "MCC.QueryAccountStableBalance"
	case MCCQueryAccountCovenantBalance:
		return "MCC.QueryAccountCovenantBalance"
	}
	return "Unknown"
}

// IsPermitted returns if the node is permitted to call the RPC func.
func IsPermitted(callerEnvelope *proto.Envelope, funcName RemoteFunc) (ok bool) {
	callerETLSNodeID := callerEnvelope.GetNodeID()
	// strict anonymous ETLS only used for Ping
	// the envelope node id is set at NodeAwareServerCodec and CryptoListener.CHandler
	// if callerETLSNodeID == nil here indicates that ETLS is not used, just ignore it
	if callerETLSNodeID != nil {
		if callerETLSNodeID.IsEqual(&kms.AnonymousRawNodeID.Hash) {
			if funcName != DHTPing {
				log.WithField("field", funcName).Warning("anonymous ETLS connection can not used")
				return false
			}
		}
	}

	if !IsBPNodeID(callerETLSNodeID) {
		// non BP
		switch funcName {
		// DHT related
		case DHTPing, DHTFindNode, DHTFindNeighbor, MetricUploadMetrics:
			return true
			// Kayak related
		case KayakCall:
			return false
			// DBSDeploy
		case DBSDeploy:
			return false
		default:
			// calling Unspecified RPC is forbidden
			return false
		}
	}

	// BP can call any RPC
	return true
}
