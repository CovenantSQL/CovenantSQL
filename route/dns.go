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
	"sync"

	"errors"

	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

//TODO(auxten): this whole file need to be implemented

// ResolveCache is the map of proto.RawNodeID to node address
type ResolveCache map[proto.RawNodeID]string

var (
	// resolver hold the singleton instance
	resolver     *Resolver
	resolverOnce sync.Once
)

var (
	// ErrUnknownNodeID indicates we got unknown node id
	ErrUnknownNodeID = errors.New("unknown node id")

	// ErrNilNodeID indicates we got nil node id
	ErrNilNodeID = errors.New("nil node id")
)

// Resolver does NodeID translation
type Resolver struct {
	cache ResolveCache
	sync.RWMutex
}

// InitResolver return a new resolver
func InitResolver() {
	resolverOnce.Do(func() {
		resolver = &Resolver{
			cache: make(ResolveCache),
		}
	})
	return
}

// IsBPNodeID return if it is Block Producer node id
func IsBPNodeID(id *proto.RawNodeID) bool {
	if id == nil {
		return false
	}
	return id.IsEqual(&kms.BP.RawNodeID.Hash)
}

// InitResolveCache init Resolver.cache by a new map
func InitResolveCache(initCache ResolveCache) {
	resolver.Lock()
	defer resolver.Unlock()
	resolver.cache = initCache
}

// GetNodeAddr get node addr by node id
func GetNodeAddr(id *proto.RawNodeID) (addr string, err error) {
	//TODO(auxten): implement that
	if id == nil {
		return "", ErrNilNodeID
	}
	resolver.RLock()
	defer resolver.RUnlock()
	addr, ok := resolver.cache[*id]
	if !ok {
		return "", ErrUnknownNodeID
	}
	return
}

// SetNodeAddr set node id and addr
func SetNodeAddr(id *proto.RawNodeID, addr string) (err error) {
	//TODO(auxten): implement that
	if id == nil {
		return ErrNilNodeID
	}
	resolver.Lock()
	defer resolver.Unlock()
	resolver.cache[*id] = addr
	return
}

// GetBPAddr return BlockProducer addresses array
func GetBPAddr() []string {
	//TODO(auxten): implement that
	return []string{"127.0.0.1:2120", "127.0.0.1:2120"}
}
