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

package blockproducer

import (
	"math/rand"

	dto "github.com/prometheus/client_model/go"
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/consistent"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/metric"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

const (
	// MetricFreeMemoryBytes defines metric name for free memory in miner instance.
	MetricFreeMemoryBytes = "node_memory_free_bytes_total"
	// MetricFreeFSBytes defines metric name for free filesystem in miner instance.
	MetricFreeFSBytes = "node_filesystem_free_bytes_total"
)

// DBService defines block producer database service rpc endpoint.
type DBService struct {
	AllocationRounds int
	ServiceMap       DBServiceMap
	Consistent       *consistent.Consistent
	NodeMetrics      *metric.NodeMetricMap
}

// CreateDatabase defines block producer create database logic.
func (s *DBService) CreateDatabase(req *CreateDatabaseRequest, resp *CreateDatabaseResponse) (err error) {
	// validate authority
	// TODO(xq262144), call accounting features, top up deposit

	// create random DatabaseID
	var dbID proto.DatabaseID
	if dbID, err = s.generateDatabaseID(req.GetNodeID()); err != nil {
		return
	}

	// allocate nodes
	var peers *kayak.Peers
	peers, err = s.allocateNodes(0, dbID, req.ResourceMeta)

	_ = peers

	// call miner nodes to provide service
	// TODO(xq262144)

	// send response to client
	// TODO(xq262144)

	return
}

// DropDatabase defines block producer drop database logic.
func (s *DBService) DropDatabase(req *DropDatabaseRequest, resp *DropDatabaseResponse) (err error) {
	// call miner nodes to drop database
	// TODO(xq262144)

	// withdraw deposit from sqlchain
	// TODO(xq262144)

	// send response to client
	// TODO(xq262144)

	return
}

// GetDatabase defines block producer get database logic.
func (s *DBService) GetDatabase(req *GetDatabaseRequest, resp *GetDatabaseResponse) (err error) {
	// fetch from meta
	// TODO(xq262144)

	// send response to client
	// TODO(xq262144)

	return
}

// GetNodeDatabases defines block producer get node databases logic.
func (s *DBService) GetNodeDatabases(req *GetNodeDatabasesRequest, resp *GetNodeDatabasesResponse) (err error) {
	// fetch from meta
	// TODO(xq262144)

	// send response to client
	// TODO(xq262144)

	return
}

func (s *DBService) generateDatabaseID(reqNodeID *proto.RawNodeID) (dbID proto.DatabaseID, err error) {
	nonceCh := make(chan cpuminer.NonceInfo)
	quitCh := make(chan struct{})
	miner := cpuminer.NewCPUMiner(quitCh)
	go miner.ComputeBlockNonce(cpuminer.MiningBlock{
		Data:      reqNodeID.CloneBytes(),
		NonceChan: nonceCh,
		Stop:      nil,
	}, cpuminer.Uint256{A: 0, B: 0, C: 0, D: 0}, 4)

	defer close(nonceCh)
	defer close(quitCh)

	for nonce := range nonceCh {
		dbID = proto.DatabaseID(nonce.Hash.String())

		// check existence
		if _, err = s.ServiceMap.Get(dbID); err == ErrNoSuchDatabase {
			return
		}
	}

	return
}

func (s *DBService) allocateNodes(lastTerm uint64, dbID proto.DatabaseID, resourceMeta DBResourceMeta) (peers *kayak.Peers, err error) {
	curRange := int(resourceMeta.Node)
	excludeNodes := make(map[proto.NodeID]bool)
	allocated := make([]proto.NodeID, 0)

	for i := 0; i != s.AllocationRounds; i++ {
		var nodes []proto.Node

		// clear previous allocated
		allocated = allocated[:0]

		nodes, err = s.Consistent.GetNeighbors(string(dbID), curRange)

		// TODO(xq262144), brute force implementation to be optimized
		var nodeIDs []proto.NodeID

		for _, node := range nodes {
			if _, ok := excludeNodes[node.ID]; !ok {
				nodeIDs = append(nodeIDs, node.ID)
			}
		}

		if len(nodeIDs) < int(resourceMeta.Node) {
			continue
		}

		// check node resource status
		metrics := s.NodeMetrics.GetMetrics(nodeIDs)

		for nodeID, nodeMetric := range metrics {
			var metricValue uint64

			// get metric
			if metricValue, err = s.getMetric(nodeMetric, MetricFreeMemoryBytes); err != nil {
				// add to excludes
				excludeNodes[nodeID] = true
				continue
			}

			// TODO(xq262144), left reserved resources check is required
			// TODO(xq262144), filesystem check to be implemented

			if resourceMeta.Memory < metricValue {
				// can allocate
				allocated = append(allocated, nodeID)
			} else {
				excludeNodes[nodeID] = true
			}
		}

		if len(allocated) >= int(resourceMeta.Node) {
			allocated = allocated[:int(resourceMeta.Node)]

			// build peers
			return s.buildPeers(lastTerm+1, nodes, allocated)
		}

		curRange += int(resourceMeta.Node)
	}

	// allocation failed
	err = ErrDatabaseAllocation
	return
}

func (s *DBService) getMetric(metric metric.MetricMap, key string) (value uint64, err error) {
	var rawMetric *dto.MetricFamily
	var ok bool

	if rawMetric, ok = metric[key]; !ok || rawMetric == nil {
		err = ErrMetricNotCollected
		return
	}

	switch rawMetric.GetType() {
	case dto.MetricType_GAUGE:
		value = uint64(rawMetric.GetMetric()[0].GetGauge().GetValue())
	case dto.MetricType_COUNTER:
		value = uint64(rawMetric.GetMetric()[0].GetCounter().GetValue())
	default:
		err = ErrMetricNotCollected
		return
	}

	return
}

func (s *DBService) buildPeers(term uint64, nodes []proto.Node, allocated []proto.NodeID) (peers *kayak.Peers, err error) {
	// get local private key
	var pubKey *asymmetric.PublicKey
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	var privKey *asymmetric.PrivateKey
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	// get allocated node info
	allocatedMap := make(map[proto.NodeID]bool)

	for _, nodeID := range allocated {
		allocatedMap[nodeID] = true
	}

	allocatedNodes := make([]proto.Node, 0, len(allocated))

	for _, node := range nodes {
		allocatedNodes = append(allocatedNodes, node)
	}

	peers = &kayak.Peers{
		Term:    term,
		PubKey:  pubKey,
		Servers: make([]*kayak.Server, len(allocated)),
	}

	// TODO(xq262144), more practical leader selection, now random select node as leader
	// random choice leader
	leaderIdx := rand.Intn(len(allocated))

	for idx, node := range allocatedNodes {
		peers.Servers[idx] = &kayak.Server{
			Role:   conf.Follower,
			ID:     node.ID,
			PubKey: node.PublicKey,
		}

		if idx == leaderIdx {
			// set as leader
			peers.Servers[idx].Role = conf.Leader
			peers.Leader = peers.Servers[idx]
		}
	}

	// sign the peers structure
	err = peers.Sign(privKey)

	return
}
