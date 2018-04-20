// Copyright (C) 2012 Numerotron Inc.
// Use of this source code is governed by an MIT-style license
// that can be found in the LICENSE file.

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
	"sort"
	"strconv"
	"sync"
	"hash/fnv"
)

type NodeKeys []NodeKey

// Len returns the length of the uints array.
func (x NodeKeys) Len() int { return len(x) }

// Less returns true if node i is less than node j.
func (x NodeKeys) Less(i, j int) bool { return x[i] < x[j] }

// Swap exchanges nodes i and j.
func (x NodeKeys) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

// ErrEmptyCircle is the error returned when trying to get an node when nothing has been added to hash.
var ErrEmptyCircle = errors.New("empty circle")

type NodeId 	string
type NodeKey	uint64

type Node struct {
	Name      string
	Port      uint16
	Protocol  string
	Id        NodeId
	PublicKey string
}

// Consistent holds the information about the members of the consistent hash circle.
type Consistent struct {
	circle           map[NodeKey]Node
	members          map[NodeId]Node
	sortedHashes     NodeKeys
	NumberOfReplicas int
	count            int64
	scratch          [64]byte
	sync.RWMutex
}

// New creates a new Consistent object with a default setting of 20 replicas for each entry.
//
// To change the number of replicas, set NumberOfReplicas before adding entries.
func New() *Consistent {
	c := new(Consistent)
	c.NumberOfReplicas = 20
	c.circle = make(map[NodeKey]Node)
	c.members = make(map[NodeId]Node)
	return c
}

// nodeKey generates a string key for an node with an index.
func (c *Consistent) nodeKey(node_id NodeId, idx int) string {
	// return node + "|" + strconv.Itoa(idx)
	return strconv.Itoa(idx) + string(node_id)
}

// Add inserts a string node in the consistent hash.
func (c *Consistent) Add(node Node) {
	c.Lock()
	defer c.Unlock()
	c.add(node)
}

// need c.Lock() before calling
func (c *Consistent) add(node Node) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		c.circle[c.hashKey(c.nodeKey(node.Id, i))] = node
	}
	c.members[node.Id] = node
	c.updateSortedHashes()
	c.count++
}

// Remove removes an node from the hash.
func (c *Consistent) Remove(node Node) {
	c.Lock()
	defer c.Unlock()
	c.remove(node.Id)
}

// need c.Lock() before calling
func (c *Consistent) remove(node_id NodeId) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		delete(c.circle, c.hashKey(c.nodeKey(node_id, i)))
	}
	delete(c.members, node_id)
	c.updateSortedHashes()
	c.count--
}

// Set sets all the nodes in the hash.  If there are existing nodes not
// present in nodes, they will be removed.
func (c *Consistent) Set(nodes []Node) {
	c.Lock()
	defer c.Unlock()
	for k := range c.members {
		found := false
		for _, v := range nodes {
			if k == v.Id {
				found = true
				break
			}
		}
		if !found {
			c.remove(k)
		}
	}
	for _, v := range nodes {
		_, exists := c.members[v.Id]
		if exists {
			continue
		}
		c.add(v)
	}
}

func (c *Consistent) Members() []Node {
	c.RLock()
	defer c.RUnlock()
	var m []Node
	for _, v := range c.members {
		m = append(m, v)
	}
	return m
}

// Get returns an node close to where name hashes to in the circle.
func (c *Consistent) Get(name string) (Node, error) {
	c.RLock()
	defer c.RUnlock()
	if len(c.circle) == 0 {
		return Node{}, ErrEmptyCircle
	}
	key := c.hashKey(name)
	i := c.search(key)
	return c.circle[c.sortedHashes[i]], nil
}

func (c *Consistent) search(key NodeKey) (i int) {
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
func (c *Consistent) GetTwo(name string) (Node, Node, error) {
	c.RLock()
	defer c.RUnlock()
	if len(c.circle) == 0 {
		return Node{}, Node{}, ErrEmptyCircle
	}
	key := c.hashKey(name)
	i := c.search(key)
	a := c.circle[c.sortedHashes[i]]

	if c.count == 1 {
		return a, Node{}, nil
	}

	start := i
	var b Node
	for i = start + 1; i != start; i++ {
		if i >= len(c.sortedHashes) {
			i = 0
		}
		b = c.circle[c.sortedHashes[i]]
		if b != a {
			break
		}
	}
	return a, b, nil
}

// GetN returns the N closest distinct nodes to the name input in the circle.
func (c *Consistent) GetN(name string, n int) ([]Node, error) {
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
		res   = make([]Node, 0, n)
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

func (c *Consistent) hashKey(key string) NodeKey {
	h := fnv.New64a()
	if len(key) < 64 {
		var scratch [64]byte
		copy(scratch[:], key)
		h.Write(scratch[:len(key)])
	}
	h.Write([]byte(key))
	return NodeKey(h.Sum64())
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

func sliceContainsMember(set []Node, member Node) bool {
	for _, m := range set {
		if m.Id == member.Id {
			return true
		}
	}
	return false
}
