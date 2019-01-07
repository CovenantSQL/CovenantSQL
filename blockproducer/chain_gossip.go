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

package blockproducer

import (
	"context"
	"sync"
	"sync/atomic"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func (c *Chain) nonblockingBroadcastBlock(block *types.BPBlock) {
	for _, info := range c.getRemoteBPInfos() {
		func(remote *blockProducerInfo) {
			c.goFuncWithTimeout(func(ctx context.Context) {
				var (
					req = &types.AdviseNewBlockReq{
						Envelope: proto.Envelope{
							// TODO(lambda): Add fields.
						},
						Block: block,
					}
					err = c.caller.CallNodeWithContext(
						ctx, remote.nodeID, route.MCCAdviseNewBlock.String(), req, nil)
				)
				log.WithFields(log.Fields{
					"local":       c.getLocalBPInfo(),
					"remote":      remote,
					"block_time":  block.Timestamp(),
					"block_hash":  block.BlockHash().Short(4),
					"parent_hash": block.ParentHash().Short(4),
				}).WithError(err).Debug("broadcast new block to other peers")
			}, c.period)
		}(info)
	}
}

func (c *Chain) nonblockingBroadcastTx(ttl uint32, tx pi.Transaction) {
	for _, info := range c.getRemoteBPInfos() {
		func(remote *blockProducerInfo) {
			c.goFuncWithTimeout(func(ctx context.Context) {
				var (
					req = &types.AddTxReq{
						Envelope: proto.Envelope{
							// TODO(lambda): Add fields.
						},
						TTL: ttl,
						Tx:  tx,
					}
					err = c.caller.CallNodeWithContext(
						ctx, remote.nodeID, route.MCCAddTx.String(), req, nil)
				)
				log.WithFields(log.Fields{
					"local":   c.getLocalBPInfo(),
					"remote":  remote,
					"hash":    tx.Hash().Short(4),
					"address": tx.GetAccountAddress(),
					"type":    tx.GetTransactionType(),
				}).WithError(err).Debug("broadcast transaction to other peers")
			}, c.tick)
		}(info)
	}
}

func (c *Chain) blockingFetchBlock(ctx context.Context, h uint32) (unreachable uint32) {
	var (
		cld, ccl = context.WithTimeout(ctx, c.tick)
		wg       = &sync.WaitGroup{}
	)
	defer func() {
		wg.Wait()
		ccl()
	}()
	for _, info := range c.getRemoteBPInfos() {
		wg.Add(1)
		go func(remote *blockProducerInfo) {
			defer wg.Done()
			var (
				err error
				req = &types.FetchBlockReq{
					Envelope: proto.Envelope{
						// TODO(lambda): Add fields.
					},
					Height: h,
				}
				resp = &types.FetchBlockResp{}
			)
			var le = log.WithFields(log.Fields{
				"local":  c.getLocalBPInfo(),
				"remote": remote,
				"height": h,
			})
			if err = c.caller.CallNodeWithContext(
				cld, remote.nodeID, route.MCCFetchBlock.String(), req, resp,
			); err != nil {
				le.WithError(err).Warn("failed to fetch block")
				atomic.AddUint32(&unreachable, 1)
				return
			}
			if resp.Block == nil {
				le.Debug("fetch block request reply: no such block")
				return
			}
			// Push new block from other peers
			le.WithFields(log.Fields{
				"parent": resp.Block.ParentHash().Short(4),
				"hash":   resp.Block.BlockHash().Short(4),
			}).Debug("fetch block request reply: found block")
			select {
			case c.pendingBlocks <- resp.Block:
			case <-cld.Done():
				log.WithError(cld.Err()).Warn("add pending block aborted")
			}
		}(info)
	}
	return
}
