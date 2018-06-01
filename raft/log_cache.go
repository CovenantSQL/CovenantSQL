package raft

import (
	"fmt"
	"sync"
)

// LogCache wraps any LogStore implementation to provide an
// in-memory ring buffer. This is used to cache access to
// the recently written entries. For implementations that do not
// cache themselves, this can provide a substantial boost by
// avoiding disk I/O on recent entries.
type LogCache struct {
	store LogStore

	cache []*Log
	l     sync.RWMutex
}

// NewLogCache is used to create a new LogCache with the
// given capacity and backend store.
func NewLogCache(capacity int, store LogStore) (*LogCache, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("capacity must be positive")
	}
	c := &LogCache{
		store: store,
		cache: make([]*Log, capacity),
	}
	return c, nil
}

// GetLog gets a log entry at a given index.
func (c *LogCache) GetLog(idx uint64, log *Log) error {
	// Check the buffer for an entry
	c.l.RLock()
	cached := c.cache[idx%uint64(len(c.cache))]
	c.l.RUnlock()

	// Check if entry is valid
	if cached != nil && cached.Index == idx {
		*log = *cached
		return nil
	}

	// Forward request on cache miss
	return c.store.GetLog(idx, log)
}

// StoreLog stores a log entry.
func (c *LogCache) StoreLog(log *Log) error {
	return c.StoreLogs([]*Log{log})
}

// StoreLogs stores multiple log entries.
func (c *LogCache) StoreLogs(logs []*Log) error {
	// Insert the logs into the ring buffer
	c.l.Lock()
	for _, l := range logs {
		c.cache[l.Index%uint64(len(c.cache))] = l
	}
	c.l.Unlock()

	return c.store.StoreLogs(logs)
}

// FirstIndex returns the first index written. 0 for no entries.
func (c *LogCache) FirstIndex() (uint64, error) {
	return c.store.FirstIndex()
}

// LastIndex returns the last index written. 0 for no entries.
func (c *LogCache) LastIndex() (uint64, error) {
	return c.store.LastIndex()
}

// DeleteRange deletes a range of log entries. The range is inclusive.
func (c *LogCache) DeleteRange(min, max uint64) error {
	// Invalidate the cache on deletes
	c.l.Lock()
	c.cache = make([]*Log, len(c.cache))
	c.l.Unlock()

	return c.store.DeleteRange(min, max)
}
