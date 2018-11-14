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

package xenomint

import (
	"context"
	"sync"
	"time"

	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
	xt "github.com/CovenantSQL/CovenantSQL/xenomint/types"
)

const (
	inCommandBufferLength  = 100000
	outCommandBufferLength = 100000
)

type command interface{}

type applyRequest struct {
	request  *types.Request
	response *types.Response
}

type commitRequest struct {
	queries []*QueryTracker
	resp    *MuxFollowerCommitResponse
}

type blockNode struct {
	parent *blockNode
	// Cached block fields
	hash   hash.Hash
	count  int32
	height int32
	// Cached block object, may be nil
	block *xt.Block
}

// Chain defines the xenomint chain structure.
type Chain struct {
	state *State
	// Command queue
	in  chan command
	out chan command
	// Config fields
	start  time.Time
	period time.Duration
	// Runtime state
	stMu     sync.RWMutex
	index    int32
	total    int32
	head     *blockNode
	nextTurn int32
	pending  []*commitRequest
	// Utility fields
	caller *rpc.Caller
	// Cached fields
	priv *ca.PrivateKey
	// Goroutine control
	wg     *sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *Chain) nextTick() (t time.Time, d time.Duration) {
	c.stMu.RLock()
	defer c.stMu.RUnlock()
	t = time.Now()
	d = c.start.Add(time.Duration(c.nextTurn) * c.period).Sub(t)
	return
}

func (c *Chain) isMyTurn() (ret bool) {
	c.stMu.RLock()
	defer c.stMu.RUnlock()
	if c.total <= 0 {
		return
	}
	ret = (c.nextTurn%c.total == c.index)
	return
}

// NewChain returns new chain instance.
func NewChain(filename string) (c *Chain, err error) {
	var (
		strg   xi.Storage
		state  *State
		priv   *ca.PrivateKey
		ctx    context.Context
		cancel context.CancelFunc
	)
	// TODO(leventeliu): add multiple storage engine support.
	if strg, err = xs.NewSqlite(filename); err != nil {
		return
	}
	if state, err = NewState(strg); err != nil {
		return
	}
	if priv, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	ctx, cancel = context.WithCancel(context.Background())
	c = &Chain{
		state:  state,
		in:     make(chan command, inCommandBufferLength),
		out:    make(chan command, outCommandBufferLength),
		priv:   priv,
		wg:     &sync.WaitGroup{},
		ctx:    ctx,
		cancel: cancel,
	}
	return
}

// Query queries req from local chain state and returns the query results in resp.
func (c *Chain) Query(req *types.Request) (resp *types.Response, err error) {
	var ref *QueryTracker
	if ref, resp, err = c.state.Query(req); err != nil {
		return
	}
	if err = resp.Sign(c.priv); err != nil {
		return
	}
	ref.UpdateResp(resp)
	startWorker(c.ctx, c.wg, func(_ context.Context) {
		c.enqueueOut(&applyRequest{request: req, response: resp})
	})
	return
}

func startWorker(ctx context.Context, wg *sync.WaitGroup, f func(context.Context)) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		f(ctx)
	}()
}

func (c *Chain) stopWorkers() {
	c.cancel()
	c.wg.Wait()
}

func (c *Chain) enqueueIn(cmd command)  { c.in <- cmd }
func (c *Chain) enqueueOut(cmd command) { c.out <- cmd }

func (c *Chain) advanceTurn(ctx context.Context, t time.Time) {
	defer func() {
		c.stMu.Lock()
		defer c.stMu.Unlock()
		c.nextTurn++
	}()
	if !c.isMyTurn() {
		return
	}
	// Send commit request to leader peer
	var _, cancel = context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
}

func (c *Chain) produceBlock() {
}

func (c *Chain) main(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if c.head.height < c.nextTurn-1 {
				if c.isMyTurn() {
					// Try to produce next block
					if c.pending[len(c.pending)].resp.Height == c.nextTurn-1 {

					}
				} else {
					// Sync head of current turn
				}
			}
			//
			if t, d := c.nextTick(); d > 0 {
				time.Sleep(d)
			} else {
				// End current turn
				c.advanceTurn(c.ctx, t)
			}
		}
	}
}

func (c *Chain) processIn(ctx context.Context) {
	var err error
	for {
		select {
		case ci := <-c.in:
			switch cmd := ci.(type) {
			case *MuxApplyRequest:
				if err = c.state.Replay(cmd.Request, cmd.Response); err != nil {
					log.WithFields(log.Fields{
						"request":  cmd.Request,
						"response": cmd.Response,
					}).WithError(err).Errorf("Failed to apply request")
				}
			default:
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Chain) commitBlock() (err error) {
	if err = c.state.commit(c.out); err != nil {
		return
	}
	return
}

func (c *Chain) processOut(ctx context.Context) {
	for {
		select {
		case ci := <-c.out:
			switch ci.(type) {
			case *applyRequest:
			default:
			}
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops chain workers and RPC service.
func (c *Chain) Stop() (err error) {
	// Stop workers
	c.stopWorkers()
	// Close all opened resources
	return c.state.Close(true)
}

// Start starts chain workers and RPC service.
func (c *Chain) Start() (err error) {
	startWorker(c.ctx, c.wg, c.processIn)
	startWorker(c.ctx, c.wg, c.processOut)
	startWorker(c.ctx, c.wg, c.main)
	return
}
