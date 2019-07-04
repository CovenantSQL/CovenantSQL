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
	"errors"
	"fmt"
	"math/rand"
	"sync"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// NodeIDAddressMap is the map of proto.RawNodeID to node address.
type NodeIDAddressMap map[proto.RawNodeID]string

// IDNodeMap is the map of proto.RawNodeID to node.
type IDNodeMap map[proto.RawNodeID]proto.Node

var (
	// resolver holds the singleton instance
	resolver *Resolver
	// Once is exported just for unit test
	Once utils.Once
)

var (
	// ErrUnknownNodeID indicates we got unknown node id
	ErrUnknownNodeID = errors.New("unknown node id")

	// ErrNilNodeID indicates we got nil node id
	ErrNilNodeID = errors.New("nil node id")
)

// Resolver does NodeID translation.
type Resolver struct {
	cache     NodeIDAddressMap
	bpNodeIDs NodeIDAddressMap
	bpNodes   IDNodeMap
	sync.RWMutex
}

// initResolver returns a new resolver.
func initResolver() {
	Once.Do(func() {
		resolver = &Resolver{
			cache:     make(NodeIDAddressMap),
			bpNodeIDs: make(NodeIDAddressMap),
		}
		initBPNodeIDs()
	})
}

// IsBPNodeID returns if it is Block Producer node id.
func IsBPNodeID(id *proto.RawNodeID) bool {
	initResolver()
	if id == nil {
		return false
	}
	_, ok := resolver.bpNodeIDs[*id]
	return ok
}

// setResolveCache initializes Resolver.cache by a new map.
func setResolveCache(initCache NodeIDAddressMap) {
	initResolver()
	resolver.Lock()
	defer resolver.Unlock()
	resolver.cache = initCache
}

// GetNodeAddrCache gets node addr by node id, if cache missed try RPC.
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

// setNodeAddrCache sets node id and addr.
func setNodeAddrCache(id *proto.RawNodeID, addr string) (err error) {
	if id == nil {
		return ErrNilNodeID
	}
	resolver.Lock()
	defer resolver.Unlock()
	resolver.cache[*id] = addr
	return
}

// SetNodeAddrCache sets node id and addr.
func SetNodeAddrCache(id *proto.RawNodeID, addr string) (err error) {
	initResolver()
	return setNodeAddrCache(id, addr)
}

// initBPNodeIDs initializes BlockProducer route and map from config file and DNS Seed.
func initBPNodeIDs() (bpNodeIDs NodeIDAddressMap) {
	if conf.GConf == nil {
		log.Fatal("call conf.LoadConfig to init conf first")
	}

	// clear address map before init
	resolver.bpNodeIDs = make(NodeIDAddressMap)
	bpNodeIDs = resolver.bpNodeIDs

	var err error

	if conf.GConf.DNSSeed.Domain != "" {
		var bpIndex int
		dc := IPv6SeedClient{}
		bpIndex = rand.Intn(conf.GConf.DNSSeed.BPCount)
		bpDomain := fmt.Sprintf("bp%02d.%s", bpIndex, conf.GConf.DNSSeed.Domain)
		log.Infof("Geting bp address from dns: %v", bpDomain)
		resolver.bpNodes, err = dc.GetBPFromDNSSeed(bpDomain)
		if err != nil {
			log.WithField("seed", bpDomain).WithError(err).Error(
				"getting BP info from DNS failed")
			return
		}
	}

	if resolver.bpNodes == nil {
		resolver.bpNodes = make(IDNodeMap)
	}
	if conf.GConf.KnownNodes != nil {
		for _, n := range conf.GConf.KnownNodes {
			rawID := n.ID.ToRawNodeID()
			if rawID != nil {
				if n.Role == proto.Leader || n.Role == proto.Follower {
					resolver.bpNodes[*rawID] = n
				}
				setNodeAddrCache(rawID, n.Addr)
			}
		}
	}

	conf.GConf.SeedBPNodes = make([]proto.Node, 0, len(resolver.bpNodes))
	for _, n := range resolver.bpNodes {
		rawID := n.ID.ToRawNodeID()
		if rawID != nil {
			conf.GConf.SeedBPNodes = append(conf.GConf.SeedBPNodes, n)
			setNodeAddrCache(rawID, n.Addr)
			resolver.bpNodeIDs[*rawID] = n.Addr
		}
	}

	return resolver.bpNodeIDs
}

// GetBPs returns the known BP node id list.
func GetBPs() (bpAddrs []proto.NodeID) {
	bpAddrs = make([]proto.NodeID, 0, len(resolver.bpNodeIDs))
	for id := range resolver.bpNodeIDs {
		bpAddrs = append(bpAddrs, proto.NodeID(id.String()))
	}
	return
}

// InitKMS inits nasty stuff, only for testing.
func InitKMS(PubKeyStoreFile string) {
	initResolver()
	kms.InitPublicKeyStore(PubKeyStoreFile, nil)
	if conf.GConf.KnownNodes != nil {
		for _, n := range conf.GConf.KnownNodes {
			rawNodeID := n.ID.ToRawNodeID()

			log.WithFields(log.Fields{
				"node": rawNodeID.String(),
				"addr": n.Addr,
			}).Debug("set node addr")
			SetNodeAddrCache(rawNodeID, n.Addr)
			node := &proto.Node{
				ID:         n.ID,
				Addr:       n.Addr,
				DirectAddr: n.DirectAddr,
				PublicKey:  n.PublicKey,
				Nonce:      n.Nonce,
				Role:       n.Role,
			}
			log.WithField("node", node).Debug("known node to set")
			err := kms.SetNode(node)
			if err != nil {
				log.WithField("node", node).WithError(err).Error("set node failed")
			}
			if n.ID == conf.GConf.ThisNodeID {
				kms.SetLocalNodeIDNonce(rawNodeID.CloneBytes(), &n.Nonce)
			}
		}
	}
	log.Debugf("AllNodes:\n %#v\n", conf.GConf.KnownNodes)
}
