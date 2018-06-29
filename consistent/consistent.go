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

// Package consistent provides a consistent hashing function.
//
// Consistent hashing is often used to distribute requests to a changing set of servers.  For example,
// say you have some cache servers cacheA, cacheB, and cacheC.  You want to decide which cache server
// to use to look up information on a user.
//
// You could use a typical hash table and hash the user id
// to one of cacheA, cacheB, or cacheC.  But with a typical hash table, if you add or remove a server,
// almost all keys will get remapped to different results, which basically could bring your service
// to a grinding halt while the caches get rebuilt.
//
// With a consistent hash, adding or removing a server drastically reduces the number of keys that
// get remapped.
//
// Read more about consistent hashing on wikipedia:  http://en.wikipedia.org/wiki/Consistent_hashing
//
package consistent

import (
	"errors"
	"hash/fnv"
	"sort"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
)

// NodeKeys is NodeKey array
type NodeKeys []proto.NodeKey

// Len returns the length of the uints array.
func (x NodeKeys) Len() int { return len(x) }

// Less returns true if node i is less than node j.
func (x NodeKeys) Less(i, j int) bool { return x[i] < x[j] }

// Swap exchanges nodes i and j.
func (x NodeKeys) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

// ErrEmptyCircle is the error returned when trying to get an node when nothing has been added to hash.
var ErrEmptyCircle = errors.New("empty circle")

// Consistent holds the information about the members of the consistent hash circle.
type Consistent struct {
	// TODO(auxten): do not store node info on circle, just put node id as value
	// and put node info into kms
	circle map[proto.NodeKey]proto.Node
	//members          map[proto.NodeID]proto.Node
	sortedHashes     NodeKeys
	NumberOfReplicas int
	count            int64
	persist          Persistence
	sync.RWMutex
}

// InitConsistent creates a new Consistent object with a default setting of 20 replicas for each entry.
//
// To change the number of replicas, set NumberOfReplicas before adding entries.
func InitConsistent(storePath string, persistImpl Persistence, initBP bool) (c *Consistent, err error) {
	var BPNode *proto.Node
	if initBP {
		// Load BlockProducer public key, set it in public key store
		// as all kms.BP stuff is initialized on kms init()
		BPNode = &proto.Node{
			ID:        kms.BPNodeID,
			Addr:      "",
			PublicKey: kms.BPPublicKey,
			Nonce:     kms.BPNonce,
		}
	}

	// Create new public key store
	//err = kms.InitPublicKeyStore(storePath, BPNode)
	//if err != nil {
	//	log.Errorf("init public keystore failed: %s", err)
	//	return
	//}
	//IDs, err := kms.GetAllNodeID()
	//if err != nil {
	//	log.Errorf("get all node id failed: %s", err)
	//	return
	//}
	c = &Consistent{
		//TODO(auxten): reduce NumberOfReplicas
		NumberOfReplicas: 20,
		circle:           make(map[proto.NodeKey]proto.Node),
		persist:          persistImpl,
	}
	c.persist.Init(storePath, BPNode)
	nodes, err := c.persist.GetAllNodeInfo()
	if err != nil {
		log.Errorf("get all node id failed: %s", err)
		return
	}
	log.Debugf("c.persist.GetAllNodeInfo: %v", nodes)
	for _, n := range nodes[:] {
		c.add(n)
	}

	return
}

// nodeKey generates a string key for an node with an index.
func (c *Consistent) nodeKey(nodeID proto.NodeID, idx int) string {
	// return node + "|" + strconv.Itoa(idx)
	return strconv.Itoa(idx) + string(nodeID)
}

// Add inserts a string node in the consistent hash.
func (c *Consistent) Add(node proto.Node) (err error) {
	c.Lock()
	defer c.Unlock()
	return c.add(node)
}

// need c.Lock() before calling
func (c *Consistent) add(node proto.Node) (err error) {
	err = c.persist.SetNode(&node)
	if err != nil {
		log.Errorf("set node info failed: %s", err)
		return
	}

	return c.AddCache(node)
}

// AddCache only adds c.circle skips persist
func (c *Consistent) AddCache(node proto.Node) (err error) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		c.circle[c.hashKey(c.nodeKey(node.ID, i))] = node
	}
	c.updateSortedHashes()
	c.count++
	return
}

// Remove removes an node from the hash.
func (c *Consistent) Remove(node proto.NodeID) (err error) {
	c.Lock()
	defer c.Unlock()
	return c.remove(node)
}

// need c.Lock() before calling
func (c *Consistent) remove(nodeID proto.NodeID) (err error) {
	err = c.persist.DelNode(nodeID)
	if err != nil {
		log.Errorf("del node failed: %s", err)
		return
	}

	for i := 0; i < c.NumberOfReplicas; i++ {
		delete(c.circle, c.hashKey(c.nodeKey(nodeID, i)))
	}
	c.updateSortedHashes()
	c.count--

	return
}

// Set sets all the nodes in the hash.  If there are existing nodes not
// present in nodes, they will be removed.
func (c *Consistent) Set(nodes []proto.Node) (err error) {
	c.Lock()
	defer c.Unlock()
	err = c.persist.Reset()
	if err != nil {
		log.Errorf("reset bucket failed: %s", err)
		return
	}

	c.circle = make(map[proto.NodeKey]proto.Node)
	c.count = 0
	c.sortedHashes = NodeKeys{}

	for _, v := range nodes {
		c.add(v)
	}

	return
}

// Get returns an node close to where name hashes to in the circle.
func (c *Consistent) Get(name string) (proto.Node, error) {
	c.RLock()
	defer c.RUnlock()
	if len(c.circle) == 0 {
		return proto.Node{}, ErrEmptyCircle
	}
	key := c.hashKey(name)
	i := c.search(key)
	return c.circle[c.sortedHashes[i]], nil
}

func (c *Consistent) search(key proto.NodeKey) (i int) {
	f := func(x int) bool {
		return c.sortedHashes[x] > key
	}
	i = sort.Search(len(c.sortedHashes), f)
	if i >= len(c.sortedHashes) {
		i = 0
	}
	return
}

// GetTwo returns the two closest distinct nodes to the name input in the circle.
func (c *Consistent) GetTwo(name string) (proto.Node, proto.Node, error) {
	c.RLock()
	defer c.RUnlock()
	if len(c.circle) == 0 {
		return proto.Node{}, proto.Node{}, ErrEmptyCircle
	}
	key := c.hashKey(name)
	i := c.search(key)
	a := c.circle[c.sortedHashes[i]]

	if c.count == 1 {
		return a, proto.Node{}, nil
	}

	start := i
	var b proto.Node
	for i = start + 1; i != start; i++ {
		if i >= len(c.sortedHashes) {
			i = 0
		}
		b = c.circle[c.sortedHashes[i]]
		if b.ID != a.ID {
			break
		}
	}
	return a, b, nil
}

// GetN returns the N closest distinct nodes to the name input in the circle.
func (c *Consistent) GetN(name string, n int) ([]proto.Node, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.circle) == 0 {
		return nil, ErrEmptyCircle
	}

	if c.count < int64(n) {
		n = int(c.count)
	}

	var (
		key   = c.hashKey(name)
		i     = c.search(key)
		start = i
		res   = make([]proto.Node, 0, n)
		elem  = c.circle[c.sortedHashes[i]]
	)

	res = append(res, elem)

	if len(res) == n {
		return res, nil
	}

	for i = start + 1; i != start; i++ {
		if i >= len(c.sortedHashes) {
			i = 0
		}
		elem = c.circle[c.sortedHashes[i]]
		if !sliceContainsMember(res, elem) {
			res = append(res, elem)
		}
		if len(res) == n {
			break
		}
	}

	return res, nil
}

func (c *Consistent) hashKey(key string) proto.NodeKey {
	h := fnv.New64a()
	if len(key) < 64 {
		var scratch [64]byte
		copy(scratch[:], key)
		h.Write(scratch[:len(key)])
	}
	h.Write([]byte(key))
	return proto.NodeKey(h.Sum64())
}

func (c *Consistent) updateSortedHashes() {
	hashes := c.sortedHashes[:0]
	//reallocate if we're holding on to too much (1/4th)
	if cap(c.sortedHashes)/(c.NumberOfReplicas*4) > len(c.circle) {
		hashes = nil
	}
	for k := range c.circle {
		hashes = append(hashes, k)
	}
	sort.Sort(hashes)
	c.sortedHashes = hashes
}

func sliceContainsMember(set []proto.Node, member proto.Node) bool {
	for _, m := range set {
		if m.ID == member.ID {
			return true
		}
	}
	return false
}
