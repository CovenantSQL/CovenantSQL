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
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/kayak"
)

// CreateDatabaseRequest defines client create database rpc request entity.
type CreateDatabaseRequest struct {
	proto.Envelope
	ResourceMeta DBResourceMeta
}

// CreateDatabaseResponse defines client create database rpc response entity.
type CreateDatabaseResponse struct {
	proto.Envelope
	DatabaseID proto.DatabaseID
	Peers      *kayak.Peers
}

// DropDatabaseRequest defines client drop database rpc request entity.
type DropDatabaseRequest struct {
	proto.Envelope
	DatabaseID proto.DatabaseID
}

// DropDatabaseResponse defines client drop database rpc response entity.
type DropDatabaseResponse struct {
	proto.Envelope
}

// GetDatabaseRequest defines client get database rpc request entity.
type GetDatabaseRequest struct {
	proto.Envelope
	DatabaseID proto.DatabaseID
}

// GetDatabaseResponse defines client get database rpc response entity.
type GetDatabaseResponse struct {
	proto.Envelope
	Peers *kayak.Peers
}

// GetNodeDatabasesRequest defines miner get node databases rpc request entity.
type GetNodeDatabasesRequest struct {
	proto.Envelope
}

// GetNodeDatabasesResponse defines miner get node databases rpc response entity.
type GetNodeDatabasesResponse struct {
	proto.Envelope
	DBS map[proto.DatabaseID]*kayak.Peers
}
