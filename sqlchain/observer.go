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
	"sync"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

/*
Observer implements interface like a sqlchain including AdviseNewBlock/AdviseAckedQuery.
Request/Response entity from sqlchain api is re-used for simplicity.

type Observer interface {
	AdviseNewBlock(*MuxAdviseNewBlockReq, *MuxAdviseNewBlockResp) error
	AdviseAckedQuery(*MuxAdviseAckedQueryReq, *MuxAdviseAckedQueryResp) error
}

The observer could call DBS.GetRequest to fetch original request entity from the DBMS service.
The whole observation of block producing and write query execution would be as follows.
AdviseAckedQuery -> AdviseNewBlock -> GetRequest.
*/

// observerReplicator defines observer replication state.
type observerReplicator struct {
	nodeID    proto.NodeID
	height    int32
	triggerCh chan struct{}
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
	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}
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

	if r.height == ct.ReplicateFromNewest {
		log.Warningf("observer %v set to read from the newest block, which is in height %v", r.nodeID, curHeight)
		r.height = curHeight
	} else if r.height > curHeight+1 {
		log.Warningf("observer %v subscribes block height %v, which is not produced yet", r.nodeID, r.height)
		log.Warningf("reset observer %v height to %v", r.nodeID, curHeight+1)
		r.height = curHeight + 1
	} else if r.height == curHeight+1 {
		// wait for next block
		log.Infof("no more blocks for observer %v to read", r.nodeID)
		return
	}

	log.Debugf("try replicating block %v for observer %v", r.height, r.nodeID)

	// replicate one record
	var block *ct.Block
	if block, err = r.c.FetchBlock(r.height); err != nil {
		// fetch block failed
		log.Warningf("fetch block with height %v failed: %v", r.height, err)
		return
	} else if block == nil {
		log.Debugf("no block of height %v for observer %v", r.height, r.nodeID)

		// black hole in chain?
		// find last available block
		log.Debug("start block height hole detection")

		var lastBlock, nextBlock *ct.Block
		var lastHeight, nextHeight int32

		for h := r.height - 1; h >= 0; h-- {
			if lastBlock, err = r.c.FetchBlock(h); err == nil && lastBlock != nil {
				lastHeight = h
				log.Debugf("found last available block %v with height %v",
					lastBlock.BlockHash().String(), lastHeight)
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
					log.Warningf("inconsistency detected during hole detection, "+
						"last block height: %v, hash: %v, next block height: %v, hash: %v, parent hash: %v",
						lastBlock.BlockHash().String(), lastHeight,
						nextBlock.BlockHash().String(), h, nextBlock.ParentHash().String())

					return
				}

				nextHeight = h
				log.Debugf("found next available block %v with height %v",
					nextBlock.BlockHash().String(), nextHeight)
				break
			}
		}

		if nextBlock == nil {
			// could not find next available block, try next time
			log.Debug("could not found next available block during hole detection")
			return
		}

		// successfully found a hole in chain
		log.Debugf("found a hole in chain, started with block: %v, height: %v to block: %v, height: %v, skipped %v blocks",
			lastBlock.BlockHash().String(), lastHeight, nextBlock.BlockHash().String(), nextHeight, nextHeight-lastHeight-1)

		r.height = nextHeight
		block = nextBlock

		log.Debugf("finish block height hole detection, skipping to block: %v, height: %v",
			block.BlockHash().String(), r.height)
	}

	// fetch acks in block
	for _, h := range block.Queries {
		var ack *wt.SignedAckHeader
		if ack, err = r.c.queryOrSyncAckedQuery(r.height, h, block.Producer()); err != nil || ack == nil {
			log.Warningf("fetch ack %v in block height %v failed: %v", h, r.height, err)
			continue
		}

		// send advise to this block
		req := &MuxAdviseAckedQueryReq{
			Envelope:   proto.Envelope{},
			DatabaseID: r.c.rt.databaseID,
			AdviseAckedQueryReq: AdviseAckedQueryReq{
				Query: ack,
			},
		}
		resp := &MuxAdviseAckedQueryResp{}
		err = r.c.cl.CallNode(r.nodeID, route.OBSAdviseAckedQuery.String(), req, resp)
		if err != nil {
			log.Warningf("send ack advise for block height %v to observer %v failed: %v",
				r.height, r.nodeID, err)
			return
		}
	}

	// send block
	req := &MuxAdviseNewBlockReq{
		Envelope:   proto.Envelope{},
		DatabaseID: r.c.rt.databaseID,
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
		log.Warningf("send block height %v advise to observer %v failed: %v", r.height, r.nodeID, err)
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
func (r *observerReplicator) run() {
	defer r.c.replWg.Done()

	for {
		select {
		case <-r.triggerCh:
			// replication
			r.replicate()
		case <-r.c.stopCh:
			r.stop()
			return
		case <-r.stopCh:
			return
		}
	}
}
