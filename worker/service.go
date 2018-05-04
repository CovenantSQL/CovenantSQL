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
	"net"
	"sync"

	"github.com/thunderdb/ThunderDB/proto"
	"github.com/thunderdb/ThunderDB/rpc"
)

// InstanceMap is a database storage instance map
type InstanceMap struct {
	sync.Map
}

// Load is to load rqlite store from storage instance map
func (d *InstanceMap) Load(k string) (s *Store, ok bool) {
	vs, ok := d.Map.Load(k)

	if !ok {
		return
	}

	s, ok = vs.(*Store)
	return
}

// Store is to store rqlite instance with specified key
func (d *InstanceMap) Store(k string, s *Store) {
	d.Map.Store(k, s)
}

// Delete is to delete a rqlite instance from instance map
func (d *InstanceMap) Delete(k string) {
	d.Map.Delete(k)
}

// Service is a database worker side RPC implementation
type Service struct {
	listenAddr net.Addr
	server     *rpc.Server
	config     *ServiceConfig
	storage    InstanceMap
}

// ServiceConfig is a database worker config
type ServiceConfig struct {
	StorageDir string
	NodeID     proto.NodeID
}

// NewService will return a new Service
func NewService(listenAddr net.Addr, server *rpc.Server, config *ServiceConfig) *Service {
	return &Service{
		listenAddr: listenAddr,
		server:     server,
		config:     config,
	}
}

// bootstrap Service instance
// 1. send local WorkerMeta to super nodes
// 2. fetch database topology from super nodes
// 3. init database storage for issued databases
// 4. form database replication chains with peer nodes
func (worker *Service) bootstrap() {
	// init database
}

// PrepareQuery is used to process prepare query from client nodes
func (worker *Service) PrepareQuery(req *proto.PrepareQueryReq, resp *proto.PrepareQueryResp) (err error) {
	return
}

// CommitQuery is used to process real query from client nodes
func (worker *Service) CommitQuery(req *proto.CommitQueryReq, resp *proto.CommitQueryResp) (err error) {
	return
}

// RollbackQuery is used to process confirm query from client nodes
func (worker *Service) RollbackQuery(req *proto.RollbackQueryReq, resp *proto.RollbackQueryResp) (err error) {
	return
}

// PrepareServiceUpdate is used to process service update from block producer
func (worker *Service) PrepareServiceUpdate(req *proto.PrepareServiceUpdateReq, resp *proto.PrepareServiceUpdateResp) {
	return
}

// CommitServiceUpdate is used to process service update from block producer
func (worker *Service) CommitServiceUpdate(req *proto.CommitServiceUpdateReq, resp *proto.CommitServiceUpdateResp) {
	return
}

// RollbackServiceUpdate is used to process service update from block producer
func (worker *Service) RollbackServiceUpdate(req *proto.RollbackServiceUpdateReq, resp *proto.RollbackServiceUpdateResp) {
	return
}

// GetStatus is used to get server status from block producer
func (worker *Service) GetStatus(req *proto.GetStatusReq, resp *proto.GetStatusResp) {
	return
}

// Checkpoint is used to get database checkpoint for other peers in database quorum
func (worker *Service) Checkpoint(req *proto.CheckpointReq, resp *proto.CheckpointResp) {
	return
}

// Subscribe is used to subscribe database updates for peers in database quorum or client
func (worker *Service) Subscribe(req *proto.SubscribeReq, resp *proto.SubscribeResp) {
	return
}

// UpdateSubscription is used to cancel existing subscription
func (worker *Service) UpdateSubscription(req *proto.UpdateSubscriptionReq, resp *proto.UpdateSubscriptionResp) {
	return
}
