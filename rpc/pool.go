/*
 * Copyright 2018-2019 The CovenantSQL Authors.
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

package rpc

import (
	"net/rpc"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

type pooledClient struct {
	*rpc.Client

	// Fields for pooling
	sync.Mutex
	freelist *freelist
	lastErr  error
}

func (c *pooledClient) SetLastErr(err error) {
	c.Lock()
	defer c.Unlock()
	if err != nil {
		c.lastErr = err
	}
}

// Call overwrites the Close method of client.
func (c *pooledClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	err := c.Client.Call(serviceMethod, args, reply)
	if err != nil { // TODO(leventeliu): check recoverable errors
		c.SetLastErr(err)
	}
	return err
}

// TODO(leventeliu): call with Client.Go still needs to explicitly set last error.

// Close overwrites the Close method of client.
func (c *pooledClient) Close() error {
	c.Lock()
	if c.freelist != nil && c.lastErr == nil {
		err := c.freelist.put(c.Client) // unwrap
		c.freelist = nil
		c.Unlock()
		return err
	}
	c.Unlock()
	return c.Client.Close()
}

type freelist struct {
	sync.RWMutex
	target proto.NodeID
	freeCh chan *rpc.Client
}

func (l *freelist) close() error {
	l.Lock()
	defer l.Unlock()
	close(l.freeCh)
	var errmsgs []string
	for s := range l.freeCh {
		if err := s.Close(); err != nil {
			errmsgs = append(errmsgs, err.Error())
		}
	}
	l.freeCh = nil // set to nil explicitly to force close in freelist.put
	if len(errmsgs) > 0 {
		return errors.Wrapf(errors.New(strings.Join(errmsgs, ", ")), "close free list %s", l.target)
	}
	return nil
}

func (l *freelist) put(cli *rpc.Client) (err error) {
	l.RLock() // note that this is read op on the freelist aspect
	defer l.RUnlock()
	select {
	case l.freeCh <- cli:
	default:
		err = cli.Close()
	}
	return
}

func (l *freelist) getFree() (cli *rpc.Client, ok bool) {
	l.RLock()
	defer l.RUnlock()
	select {
	case cli, ok = <-l.freeCh:
	default:
	}
	return
}

func (l *freelist) get() (cli Client, err error) {
	var (
		raw *rpc.Client
		ok  bool
	)
	if raw, ok = l.getFree(); !ok {
		if raw, err = l.newClient(); err != nil {
			return
		}
	}
	return &pooledClient{
		Client:   raw, // wrap
		freelist: l,
	}, nil
}

// len returns physical connection count.
func (l *freelist) len() int {
	l.RLock()
	defer l.RUnlock()
	return len(l.freeCh)
}

func (l *freelist) newClient() (*rpc.Client, error) {
	conn, err := Dial(l.target)
	if err != nil {
		return nil, errors.Wrap(err, "dialing new connection failed")
	}
	return NewClient(conn), err
}

// ClientPool is the struct type of connection pool.
type ClientPool struct {
	nodeFreeLists sync.Map // proto.NodeID -> freelist
}

func (p *ClientPool) loadFreeList(id proto.NodeID) (list *freelist, ok bool) {
	var v interface{}
	if v, ok = p.nodeFreeLists.Load(id); ok {
		list = v.(*freelist)
		return
	}
	v, ok = p.nodeFreeLists.LoadOrStore(id, &freelist{
		target: id,
		freeCh: make(chan *rpc.Client, conf.MaxRPCPoolPhysicalConnection),
	})
	list = v.(*freelist)
	return
}

// Get returns existing freelist to the node, if not exist try best to create one.
func (p *ClientPool) Get(id proto.NodeID) (cli Client, err error) {
	list, _ := p.loadFreeList(id)
	return list.get()
}

// GetEx returns a client with an one-off connection if it's anonymous,
// otherwise returns existing freelist with Get.
func (p *ClientPool) GetEx(id proto.NodeID, isAnonymous bool) (cli Client, err error) {
	if isAnonymous {
		conn, err := DialEx(id, true)
		if err != nil {
			return nil, err
		}
		return NewClient(conn), nil
	}
	return p.Get(id)
}

// Remove the node freelist in the pool.
func (p *ClientPool) Remove(id proto.NodeID) {
	v, ok := p.nodeFreeLists.Load(id)
	if ok {
		_ = v.(*freelist).close()
		p.nodeFreeLists.Delete(id)
	}
	return
}

// Close closes all FreeLists in the pool.
func (p *ClientPool) Close() error {
	var errmsgs []string
	p.nodeFreeLists.Range(func(k, v interface{}) bool {
		if err := v.(*freelist).close(); err != nil {
			errmsgs = append(errmsgs, err.Error())
		}
		return true
	})
	if len(errmsgs) > 0 {
		return errors.Wrap(errors.New(strings.Join(errmsgs, ", ")), "close connection pool")
	}
	return nil
}

// Len returns the connection count in the pool.
func (p *ClientPool) Len() (total int) {
	p.nodeFreeLists.Range(func(k, v interface{}) bool {
		total += v.(*freelist).len()
		return true
	})
	return
}

var (
	defaultPool = &ClientPool{}
)
