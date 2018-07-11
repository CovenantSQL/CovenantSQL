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
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

// NodeIDAddressMap is the map of proto.RawNodeID to node address
type NodeIDAddressMap map[proto.RawNodeID]string

// BPDomain is the default BP domain list
const BPDomain = "_bp._tcp.gridb.io."

var (
	// resolver holds the singleton instance
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
	cache NodeIDAddressMap
	sync.RWMutex
}

// initResolver returns a new resolver
func initResolver() {
	resolverOnce.Do(func() {
		resolver = &Resolver{
			cache: make(NodeIDAddressMap),
		}
		initBPFromDNSSeed()
	})
	return
}

// IsBPNodeID returns if it is Block Producer node id
func IsBPNodeID(id *proto.RawNodeID) bool {
	if id == nil {
		return false
	}
	return id.IsEqual(&kms.BP.RawNodeID.Hash)
}

// setResolveCache initializes Resolver.cache by a new map
func setResolveCache(initCache NodeIDAddressMap) {
	initResolver()
	resolver.Lock()
	defer resolver.Unlock()
	resolver.cache = initCache
}

// GetNodeAddrCache gets node addr by node id, if cache missed try RPC
func GetNodeAddrCache(id *proto.RawNodeID) (addr string, err error) {
	initResolver()
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

// SetNodeAddrCache sets node id and addr
func SetNodeAddrCache(id *proto.RawNodeID, addr string) (err error) {
	initResolver()
	if id == nil {
		return ErrNilNodeID
	}
	resolver.Lock()
	defer resolver.Unlock()
	resolver.cache[*id] = addr
	return
}

// initBPFromDNSSeed initializes BlockProducer info from DNS Seed
func initBPFromDNSSeed() {
	dc := NewDNSClient()
	addrs, err := dc.GetBPIDAddrMap(BPDomain)
	if err == nil {
		for id, addr := range addrs {
			SetNodeAddrCache(&id, addr)
		}
	} else {
		log.Errorf("getting BP addr from DNS failed: %s", err)
	}

	return
}
