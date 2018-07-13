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

package types

import (
	"gitlab.com/thunderdb/ThunderDB/proto"
)

// UpdateType defines service update type.
type UpdateType int32

const (
	// CreateDB indicates create database operation.
	CreateDB UpdateType = iota
	// UpdateDB indicates database peers update operation.
	UpdateDB
	// DropDB indicates drop database operation.
	DropDB
)

// UpdateService defines service update type.
type UpdateService struct {
	proto.Envelope
	Op       UpdateType
	Instance ServiceInstance
}

// UpdateServiceResponse defines empty response entity.
type UpdateServiceResponse struct{}
