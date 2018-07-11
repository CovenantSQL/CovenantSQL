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

	"github.com/prometheus/common/log"
	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils"
)

// ResolveCache is the map of proto.RawNodeID to node address
type ResolveCache map[proto.RawNodeID]string

const BPDomain = "_bp._tcp.gridb.io."

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
	cache       ResolveCache
	bpAddresses []string // only write on booting
	sync.RWMutex
}

// InitResolver return a new resolver
func InitResolver() {
	resolverOnce.Do(func() {
		resolver = &Resolver{
			cache:       make(ResolveCache),
			bpAddresses: initBPAddrs(),
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

// SetResolveCache init Resolver.cache by a new map
func SetResolveCache(initCache ResolveCache) {
	resolver.Lock()
	defer resolver.Unlock()
	resolver.cache = initCache
}

// GetNodeAddrCache get node addr by node id, if cache missed try RPC
func GetNodeAddrCache(id *proto.RawNodeID) (addr string, err error) {
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

// SetNodeAddrCache set node id and addr
func SetNodeAddrCache(id *proto.RawNodeID, addr string) (err error) {
	if id == nil {
		return ErrNilNodeID
	}
	resolver.Lock()
	defer resolver.Unlock()
	resolver.cache[*id] = addr
	return
}

func GetBPAddrs() (BPAddrs []string) {
	return resolver.bpAddresses
}

// initBPAddrs return BlockProducer addresses array
func initBPAddrs() (BPAddrs []string) {
	if conf.GConf.KnownNodes != nil {
		for _, n := range (*conf.GConf.KnownNodes)[:] {
			if n.Role == conf.Leader || n.Role == conf.Follower {
				BPAddrs = append(BPAddrs, n.Addr)
			}
		}
	}

	dc := NewDNSClient()
	addrs, err := dc.GetBPAddresses(BPDomain)
	if err == nil {
		for _, addr := range addrs {
			BPAddrs = append(BPAddrs, addr)
		}
	} else {
		log.Errorf("getting BP addr from DNS failed: %s", err)
	}

	BPAddrs = utils.RemoveDuplicatesUnordered(BPAddrs)
	return
}
