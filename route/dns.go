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

	"encoding/hex"

	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

// NodeIDAddressMap is the map of proto.RawNodeID to node address
type NodeIDAddressMap map[proto.RawNodeID]string

// BPDomain is the default BP domain list
const BPDomain = "_bp._tcp.gridb.io."

var (
	// resolver holds the singleton instance
	resolver *Resolver
	// Once is exported just for unit test
	Once sync.Once
)

var (
	// ErrUnknownNodeID indicates we got unknown node id
	ErrUnknownNodeID = errors.New("unknown node id")

	// ErrNilNodeID indicates we got nil node id
	ErrNilNodeID = errors.New("nil node id")
)

// Resolver does NodeID translation
type Resolver struct {
	cache     NodeIDAddressMap
	bpNodeIDs NodeIDAddressMap
	sync.RWMutex
}

// initResolver returns a new resolver
func initResolver() {
	Once.Do(func() {
		resolver = &Resolver{
			cache:     make(NodeIDAddressMap),
			bpNodeIDs: make(NodeIDAddressMap),
		}
		initBPNodeIDs()
	})
	return
}

// IsBPNodeID returns if it is Block Producer node id
func IsBPNodeID(id *proto.RawNodeID) bool {
	initResolver()
	if id == nil {
		return false
	}
	_, ok := resolver.bpNodeIDs[*id]
	return ok
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

// setNodeAddrCache sets node id and addr
func setNodeAddrCache(id *proto.RawNodeID, addr string) (err error) {
	if id == nil {
		return ErrNilNodeID
	}
	resolver.Lock()
	defer resolver.Unlock()
	resolver.cache[*id] = addr
	return
}

// SetNodeAddrCache sets node id and addr
func SetNodeAddrCache(id *proto.RawNodeID, addr string) (err error) {
	initResolver()
	return setNodeAddrCache(id, addr)
}

// initBPNodeIDs initializes BlockProducer route and map from config file and DNS Seed
func initBPNodeIDs() (bpNodeIDs NodeIDAddressMap) {
	// clear address map before init
	resolver.bpNodeIDs = make(NodeIDAddressMap)
	bpNodeIDs = resolver.bpNodeIDs

	if conf.GConf.KnownNodes != nil {
		for _, n := range (*conf.GConf.KnownNodes)[:] {
			if n.Role == proto.Leader || n.Role == proto.Follower {
				rawID := n.ID.ToRawNodeID()
				if rawID != nil {
					setNodeAddrCache(rawID, n.Addr)
					resolver.bpNodeIDs[*rawID] = n.Addr
				}
			}
		}

		return
	}

	dc := NewDNSClient()
	addrs, err := dc.GetBPIDAddrMap(BPDomain)
	if err == nil {
		for id, addr := range addrs {
			setNodeAddrCache(&id, addr)
			resolver.bpNodeIDs[id] = addr
		}
	} else {
		log.Errorf("getting BP addr from DNS failed: %s", err)
	}

	return resolver.bpNodeIDs
}

// GetBPs return the known BP node id list
func GetBPs() (BPAddrs []proto.NodeID) {
	BPAddrs = make([]proto.NodeID, 0, len(resolver.bpNodeIDs))
	for id := range resolver.bpNodeIDs {
		BPAddrs = append(BPAddrs, proto.NodeID(id.String()))
	}
	return
}

// InitKMS inits nasty stuff, only for testing
func InitKMS(PubKeyStoreFile string) {
	if conf.GConf.KnownNodes != nil {
		for i, n := range (*conf.GConf.KnownNodes)[:] {
			if n.Role == proto.Leader || n.Role == proto.Follower {
				//HACK(auxten): put PublicKey to yaml
				(*conf.GConf.KnownNodes)[i].PublicKey = kms.BP.PublicKey
				log.Debugf("node: %s, pubkey: %x", n.ID, kms.BP.PublicKey.Serialize())
			}
			if n.Role == proto.Client {
				var publicKeyBytes []byte
				var clientPublicKey *asymmetric.PublicKey
				//FIXME(auxten) remove fixed client pub key
				//02ec784ca599f21ef93fe7abdc68d78817ab6c9b31f2324d15ea174d9da498b4c4
				publicKeyBytes, err := hex.DecodeString("02ec784ca599f21ef93fe7abdc68d78817ab6c9b31f2324d15ea174d9da498b4c4")
				if err != nil {
					log.Errorf("hex decode clientPublicKey error: %s", err)
					continue
				}
				clientPublicKey, err = asymmetric.ParsePubKey(publicKeyBytes)
				if err != nil {
					log.Errorf("parse clientPublicKey error: %s", err)
					continue
				}
				(*conf.GConf.KnownNodes)[i].PublicKey = clientPublicKey
			}
		}
	}
	kms.InitPublicKeyStore(PubKeyStoreFile, nil)
	if conf.GConf.KnownNodes != nil {
		for _, n := range (*conf.GConf.KnownNodes)[:] {
			rawNodeID := n.ID.ToRawNodeID()

			log.Debugf("set node addr: %v, %v", rawNodeID, n.Addr)
			SetNodeAddrCache(rawNodeID, n.Addr)
			node := &proto.Node{
				ID:        n.ID,
				Addr:      n.Addr,
				PublicKey: n.PublicKey,
				Nonce:     n.Nonce,
				Role:      n.Role,
			}
			log.Debugf("known node to set: %v", node)
			err := kms.SetNode(node)
			if err != nil {
				log.Errorf("set node failed: %v\n %s", node, err)
			}
			if n.ID == conf.GConf.ThisNodeID {
				kms.SetLocalNodeIDNonce(rawNodeID.CloneBytes(), &n.Nonce)
			}
		}
	}
	log.Debugf("AllNodes:\n %v\n", conf.GConf.KnownNodes)
}
