/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
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

	"github.com/thunderdb/ThunderDB/proto"
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
	circle           map[proto.NodeKey]proto.Node
	members          map[proto.NodeID]proto.Node
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
	c.circle = make(map[proto.NodeKey]proto.Node)
	c.members = make(map[proto.NodeID]proto.Node)
	return c
}

// nodeKey generates a string key for an node with an index.
func (c *Consistent) nodeKey(nodeID proto.NodeID, idx int) string {
	// return node + "|" + strconv.Itoa(idx)
	return strconv.Itoa(idx) + string(nodeID)
}

// Add inserts a string node in the consistent hash.
func (c *Consistent) Add(node proto.Node) {
	c.Lock()
	defer c.Unlock()
	c.add(node)
}

// need c.Lock() before calling
func (c *Consistent) add(node proto.Node) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		c.circle[c.hashKey(c.nodeKey(node.ID, i))] = node
	}
	c.members[node.ID] = node
	c.updateSortedHashes()
	c.count++
}

// Remove removes an node from the hash.
func (c *Consistent) Remove(node proto.Node) {
	c.Lock()
	defer c.Unlock()
	c.remove(node.ID)
}

// need c.Lock() before calling
func (c *Consistent) remove(nodeID proto.NodeID) {
	for i := 0; i < c.NumberOfReplicas; i++ {
		delete(c.circle, c.hashKey(c.nodeKey(nodeID, i)))
	}
	delete(c.members, nodeID)
	c.updateSortedHashes()
	c.count--
}

// Set sets all the nodes in the hash.  If there are existing nodes not
// present in nodes, they will be removed.
func (c *Consistent) Set(nodes []proto.Node) {
	c.Lock()
	defer c.Unlock()
	for k := range c.members {
		found := false
		for _, v := range nodes {
			if k == v.ID {
				found = true
				break
			}
		}
		if !found {
			c.remove(k)
		}
	}
	for _, v := range nodes {
		_, exists := c.members[v.ID]
		if exists {
			continue
		}
		c.add(v)
	}
}

// Members get all Node inserted by Set/Add
func (c *Consistent) Members() []proto.Node {
	c.RLock()
	defer c.RUnlock()
	var m []proto.Node
	for _, v := range c.members {
		m = append(m, v)
	}
	return m
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
		if b != a {
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
