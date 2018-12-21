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

package sqlchain

import (
	"context"
	"sync"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

/*
Observer implements method AdviseNewBlock to receive blocks from sqlchain node.
Request/Response entity from sqlchain api is re-used for simplicity.

type Observer interface {
	AdviseNewBlock(*MuxAdviseNewBlockReq, *MuxAdviseNewBlockResp) error
}
*/

// observerReplicator defines observer replication state.
type observerReplicator struct {
	nodeID    proto.NodeID
	height    int32
	triggerCh chan struct{}
	stopOnce  sync.Once
	stopCh    chan struct{}
	replLock  sync.Mutex
	c         *Chain
}

// newObserverReplicator creates new observer.
func newObserverReplicator(nodeID proto.NodeID, startHeight int32, c *Chain) *observerReplicator {
	return &observerReplicator{
		nodeID:    nodeID,
		height:    startHeight,
		triggerCh: make(chan struct{}, 1),
		stopCh:    make(chan struct{}, 1),
		c:         c,
	}
}

func (r *observerReplicator) setNewHeight(newHeight int32) {
	r.replLock.Lock()
	defer r.replLock.Unlock()
	r.height = newHeight
}

func (r *observerReplicator) stop() {
	r.stopOnce.Do(func() {
		select {
		case <-r.stopCh:
		default:
			close(r.stopCh)
		}
	})
}

func (r *observerReplicator) replicate() {
	r.replLock.Lock()
	defer r.replLock.Unlock()

	var err error

	defer func() {
		if err != nil {
			// TODO(xq262144), add backoff logic to prevent sqlchain node from flooding the observer
		}
	}()

	curHeight := r.c.rt.getHead().Height

	if r.height == types.ReplicateFromNewest {
		log.WithFields(log.Fields{
			"node":   r.nodeID,
			"height": curHeight,
		}).Warning("observer being set to read from the newest block")
		r.height = curHeight
	} else if r.height > curHeight+1 {
		log.WithFields(log.Fields{
			"node":   r.nodeID,
			"height": r.height,
		}).Warning("observer subscribes to height not yet produced")
		log.WithFields(log.Fields{
			"node":   r.nodeID,
			"height": curHeight + 1,
		}).Warning("reset observer to height")
		r.height = curHeight + 1
	} else if r.height == curHeight+1 {
		// wait for next block
		log.WithField("node", r.nodeID).Info("no more blocks for observer to read")
		return
	}

	log.WithFields(log.Fields{
		"node":   r.nodeID,
		"height": r.height,
	}).Debug("try replicating block for observer")

	// replicate one record
	var block *types.Block
	if block, err = r.c.FetchBlock(r.height); err != nil {
		// fetch block failed
		log.WithField("height", r.height).WithError(err).Warning("fetch block with height failed")
		return
	} else if block == nil {
		log.WithFields(log.Fields{
			"node":   r.nodeID,
			"height": r.height,
		}).Debug("no block of height for observer")

		// black hole in chain?
		// find last available block
		log.Debug("start block height hole detection")

		var lastBlock, nextBlock *types.Block
		var lastHeight, nextHeight int32

		for h := r.height - 1; h >= 0; h-- {
			if lastBlock, err = r.c.FetchBlock(h); err == nil && lastBlock != nil {
				lastHeight = h
				log.WithFields(log.Fields{
					"block":  lastBlock.BlockHash().String(),
					"height": lastHeight,
				}).Debug("found last available block of height")
				break
			}
		}

		if lastBlock == nil {
			// could not find last available block, this should be a fatal issue
			log.Warning("could not found last available block during hole detection")
			return
		}

		// find next available block
		for h := r.height + 1; h <= curHeight; h++ {
			if nextBlock, err = r.c.FetchBlock(h); err == nil && nextBlock != nil {
				if !nextBlock.ParentHash().IsEqual(lastBlock.BlockHash()) {
					// inconsistency
					log.WithFields(log.Fields{
						"lastHeight":       lastHeight,
						"lastHash":         lastBlock.BlockHash().String(),
						"nextHeight":       h,
						"nextHash":         nextBlock.BlockHash().String(),
						"actualParentHash": nextBlock.ParentHash().String(),
					}).Warning("inconsistency detected during hole detection")

					return
				}

				nextHeight = h
				log.WithFields(log.Fields{
					"block":  nextBlock.BlockHash().String(),
					"height": nextHeight,
				}).Debug("found next available block of height")
				break
			}
		}

		if nextBlock == nil {
			// could not find next available block, try next time
			log.Debug("could not found next available block during hole detection")
			return
		}

		// successfully found a hole in chain
		log.WithFields(log.Fields{
			"fromBlock":  lastBlock.BlockHash().String(),
			"fromHeight": lastHeight,
			"toBlock":    nextBlock.BlockHash().String(),
			"toHeight":   nextHeight,
			"skipped":    nextHeight - lastHeight - 1,
		}).Debug("found a hole in chain, skipping")

		r.height = nextHeight
		block = nextBlock

		log.WithFields(log.Fields{
			"block":  block.BlockHash().String(),
			"height": r.height,
		}).Debug("finish block height hole detection, skipping")
	}

	// send block
	req := &MuxAdviseNewBlockReq{
		Envelope:   proto.Envelope{},
		DatabaseID: r.c.databaseID,
		AdviseNewBlockReq: AdviseNewBlockReq{
			Block: block,
			Count: func() int32 {
				if nd := r.c.bi.lookupNode(block.BlockHash()); nd != nil {
					return nd.count
				}
				if pn := r.c.bi.lookupNode(block.ParentHash()); pn != nil {
					return pn.count + 1
				}
				return -1
			}(),
		},
	}
	resp := &MuxAdviseNewBlockResp{}
	err = r.c.cl.CallNode(r.nodeID, route.OBSAdviseNewBlock.String(), req, resp)
	if err != nil {
		log.WithFields(log.Fields{
			"node":   r.nodeID,
			"height": r.height,
		}).WithError(err).Warning("send block advise to observer failed")
		return
	}

	// advance to next height
	r.height++

	if r.height <= r.c.rt.getHead().Height {
		// send ticks to myself
		r.tick()
	}
}

func (r *observerReplicator) tick() {
	select {
	case r.triggerCh <- struct{}{}:
	default:
	}
}
func (r *observerReplicator) run(ctx context.Context) {
	for {
		select {
		case <-r.triggerCh:
			// replication
			r.replicate()
		case <-ctx.Done():
			r.stop()
			return
		case <-r.stopCh:
			return
		}
	}
}
