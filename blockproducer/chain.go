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
	"fmt"
	"os"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/chainbus"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/merkle"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	"github.com/pkg/errors"
)

var (
	metaBucket                     = [4]byte{0x0, 0x0, 0x0, 0x0}
	metaStateKey                   = []byte("covenantsql-state")
	metaBlockIndexBucket           = []byte("covenantsql-block-index-bucket")
	metaTransactionBucket          = []byte("covenantsql-tx-index-bucket")
	metaAccountIndexBucket         = []byte("covenantsql-account-index-bucket")
	metaSQLChainIndexBucket        = []byte("covenantsql-sqlchain-index-bucket")
	metaProviderIndexBucket        = []byte("covenantsql-provider-index-bucket")
	gasPrice                uint32 = 1
	accountAddress          proto.AccountAddress
	txEvent                 = "/BP/Tx"
)

// Chain defines the main chain.
type Chain struct {
	ctx context.Context
	rt  *runtime
	st  xi.Storage
	cl  *rpc.Caller
	bs  chainbus.Bus

	pendingBlocks chan *types.BPBlock
	pendingTxs    chan pi.Transaction
}

// NewChain creates a new blockchain.
func NewChain(cfg *Config) (c *Chain, err error) {
	return NewChainWithContext(context.Background(), cfg)
}

// NewChainWithContext creates a new blockchain with context.
func NewChainWithContext(ctx context.Context, cfg *Config) (c *Chain, err error) {
	var (
		existed bool
		ierr    error

		st        xi.Storage
		irre      *blockNode
		heads     []*blockNode
		immutable *metaState
		txPool    map[hash.Hash]pi.Transaction

		addr   proto.AccountAddress
		pubKey *asymmetric.PublicKey

		inst   *Chain
		rt     *runtime
		bus    = chainbus.New()
		caller = rpc.NewCaller()
	)

	if fi, err := os.Stat(cfg.DataFile); err == nil && fi.Mode().IsRegular() {
		existed = true
	}

	// Open storage
	if st, ierr = openStorage(fmt.Sprintf("file:%s", cfg.DataFile)); ierr != nil {
		err = errors.Wrap(ierr, "failed to open storage")
		return
	}

	// Storage genesis
	if !existed {
		// TODO(leventeliu): reuse rt.switchBranch to construct initial state.
		var init = newMetaState()
		for _, v := range cfg.Genesis.Transactions {
			if ierr = init.apply(v); ierr != nil {
				err = errors.Wrap(ierr, "failed to initialize immutable state")
				return
			}
		}
		var sps []storageProcedure
		sps = append(sps, addBlock(0, cfg.Genesis))
		for k, v := range init.dirty.accounts {
			if v != nil {
				sps = append(sps, updateAccount(&v.Account))
			} else {
				sps = append(sps, deleteAccount(k))
			}
		}
		for k, v := range init.dirty.databases {
			if v != nil {
				sps = append(sps, updateShardChain(&v.SQLChainProfile))
			} else {
				sps = append(sps, deleteShardChain(k))
			}
		}
		for k, v := range init.dirty.provider {
			if v != nil {
				sps = append(sps, updateProvider(&v.ProviderProfile))
			} else {
				sps = append(sps, deleteProvider(k))
			}
		}
		sps = append(sps, updateIrreversible(cfg.Genesis.SignedHeader.BlockHash))
		if ierr = store(st, sps, nil); ierr != nil {
			err = errors.Wrap(ierr, "failed to initialize storage")
			return
		}
	}

	// Load and create runtime
	if irre, heads, immutable, txPool, ierr = loadDatabase(st); ierr != nil {
		err = errors.Wrap(ierr, "failed to load data from storage")
		return
	}
	rt = newRuntime(ctx, cfg, addr, irre, heads, immutable, txPool)

	// get accountAddress
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	if addr, err = crypto.PubKeyHash(pubKey); err != nil {
		return
	}

	// create chain
	inst = &Chain{
		ctx: ctx,
		rt:  rt,
		st:  st,
		cl:  caller,
		bs:  bus,

		pendingBlocks: make(chan *types.BPBlock),
		pendingTxs:    make(chan pi.Transaction),
	}

	log.WithFields(log.Fields{
		"index":     inst.rt.locSvIndex,
		"bp_number": inst.rt.serversNum,
		"period":    inst.rt.period.String(),
		"tick":      inst.rt.tick.String(),
		"height":    inst.rt.head().height,
	}).Debug("current chain state")

	// sub chain events
	inst.bs.Subscribe(txEvent, inst.addTx)

	c = inst
	return
}

// checkBlock has following steps: 1. check parent block 2. checkTx 2. merkle tree 3. Hash 4. Signature.
func (c *Chain) checkBlock(b *types.BPBlock) (err error) {
	rootHash := merkle.NewMerkle(b.GetTxHashes()).GetRoot()
	if !b.SignedHeader.MerkleRoot.IsEqual(rootHash) {
		return ErrInvalidMerkleTreeRoot
	}

	enc, err := b.SignedHeader.BPHeader.MarshalHash()
	if err != nil {
		return err
	}
	h := hash.THashH(enc)
	if !b.BlockHash().IsEqual(&h) {
		return ErrInvalidHash
	}

	return nil
}

func (c *Chain) pushBlockWithoutCheck(b *types.BPBlock) (err error) {
	if err = c.rt.applyBlock(c.st, b); err != nil {
		return err
	}
	return err
}

func (c *Chain) pushBlock(b *types.BPBlock) error {
	err := c.checkBlock(b)
	if err != nil {
		err = errors.Wrap(err, "check block failed")
		return err
	}

	err = c.pushBlockWithoutCheck(b)
	if err != nil {
		return err
	}

	return nil
}

func (c *Chain) produceBlock(now time.Time) (err error) {
	var (
		priv *asymmetric.PrivateKey
		b    *types.BPBlock
	)

	if priv, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	if b, err = c.rt.produceBlock(c.st, now, priv); err != nil {
		return
	}
	log.WithField("block", b).Debug("produced new block")

	var peers = c.rt.getPeers()
	for _, s := range peers.Servers {
		if !s.IsEqual(&c.rt.nodeID) {
			// Bind NodeID to subroutine
			func(id proto.NodeID) {
				c.rt.goFunc(func(ctx context.Context) {
					var (
						blockReq = &AdviseNewBlockReq{
							Envelope: proto.Envelope{
								// TODO(lambda): Add fields.
							},
							Block: b,
						}
						blockResp = &AdviseNewBlockResp{}
					)
					if err := c.cl.CallNodeWithContext(
						ctx, id, route.MCCAdviseNewBlock.String(), blockReq, blockResp,
					); err != nil {
						log.WithFields(log.Fields{
							"peer":       c.rt.peerInfo(),
							"now_time":   time.Now().UTC().Format(time.RFC3339Nano),
							"block_hash": b.BlockHash(),
						}).WithError(err).Error("failed to advise new block")
					} else {
						log.WithFields(log.Fields{
							"node": id,
						}).Debug("success advising block")
					}
				})
			}(s)
		}
	}

	return err
}

func (c *Chain) fetchBlock(h hash.Hash) (b *types.BPBlock, err error) {
	var (
		enc []byte
		out = &types.BPBlock{}
	)
	if err = c.st.Reader().QueryRow(
		`SELECT "encoded" FROM "blocks" WHERE "hash"=?`, h.String(),
	).Scan(&enc); err != nil {
		return
	}
	if err = utils.DecodeMsgPack(enc, out); err != nil {
		return
	}
	b = out
	return
}

func (c *Chain) fetchBlockByHeight(h uint32) (b *types.BPBlock, count uint32, err error) {
	var node = c.rt.head().ancestor(h)
	if node == nil {
		err = ErrNoSuchBlock
		return
	} else if node.block != nil {
		b = node.block
		count = node.count
		return
	}
	// Not cached, read from database
	if b, err = c.fetchBlock(node.hash); err != nil {
		return
	}
	count = node.count
	return
}

func (c *Chain) fetchBlockByCount(count uint32) (b *types.BPBlock, height uint32, err error) {
	var node = c.rt.head().ancestorByCount(count)
	if node == nil {
		err = ErrNoSuchBlock
		return
	} else if node.block != nil {
		b = node.block
		height = node.height
		return
	}
	// Not cached, read from database
	if b, err = c.fetchBlock(node.hash); err != nil {
		return
	}
	height = node.height
	return
}

// runCurrentTurn does the check and runs block producing if its my turn.
func (c *Chain) runCurrentTurn(now time.Time) {
	log.WithFields(log.Fields{
		"next_turn":  c.rt.getNextTurn(),
		"bp_number":  c.rt.serversNum,
		"node_index": c.rt.locSvIndex,
	}).Info("check turns")
	defer c.rt.setNextTurn()

	if !c.rt.isMyTurn() {
		return
	}

	log.WithField("height", c.rt.getNextTurn()).Info("producing a new block")
	if err := c.produceBlock(now); err != nil {
		log.WithField("now", now.Format(time.RFC3339Nano)).WithError(err).Errorln(
			"failed to produce block")
	}
}

// sync synchronizes blocks and queries from the other peers.
func (c *Chain) sync() error {
	log.WithFields(log.Fields{
		"peer": c.rt.peerInfo(),
	}).Debug("synchronizing chain state")

	// sync executes firstly alone, so it's ok to sync without locking runtime
	for {
		now := c.rt.now()
		height := c.rt.height(now)

		log.WithFields(log.Fields{
			"height":   height,
			"nextTurn": c.rt.nextTurn,
		}).Info("try sync heights")
		if c.rt.nextTurn >= height {
			log.WithFields(log.Fields{
				"height":   height,
				"nextTurn": c.rt.nextTurn,
			}).Info("return heights")
			break
		}

		for c.rt.nextTurn <= height {
			// TODO(lambda): fetch blocks and txes.
			c.rt.nextTurn++
		}
	}

	return nil
}

// Start starts the chain by step:
// 1. sync the chain
// 2. goroutine for getting blocks
// 3. goroutine for getting txes.
func (c *Chain) Start() error {
	err := c.sync()
	if err != nil {
		return err
	}

	c.rt.goFunc(c.processBlocks)
	c.rt.goFunc(c.processTxs)
	c.rt.goFunc(c.mainCycle)
	c.rt.startService(c)

	return nil
}

func (c *Chain) processBlocks(ctx context.Context) {
	for {
		select {
		case block := <-c.pendingBlocks:
			err := c.pushBlock(block)
			if err != nil {
				log.WithFields(log.Fields{
					"block_hash":        block.BlockHash(),
					"block_parent_hash": block.ParentHash(),
					"block_timestamp":   block.Timestamp(),
				}).Debug(err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (c *Chain) addTx(tx pi.Transaction) {
	c.pendingTxs <- tx
}

func (c *Chain) processTx(tx pi.Transaction) {
	if err := tx.Verify(); err != nil {
		log.WithError(err).Error("Failed to verify transaction")
		return
	}
	if err := c.rt.addTx(c.st, tx); err != nil {
		log.WithError(err).Error("Failed to add transaction")
	}
}

func (c *Chain) processTxs(ctx context.Context) {
	for {
		select {
		case tx := <-c.pendingTxs:
			c.processTx(tx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *Chain) mainCycle(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			c.syncHead()
			if t, d := c.rt.nextTick(); d > 0 {
				log.WithFields(log.Fields{
					"peer":        c.rt.peerInfo(),
					"next_turn":   c.rt.getNextTurn(),
					"head_height": c.rt.head().height,
					"head_block":  c.rt.head().hash.String(),
					"now_time":    t.Format(time.RFC3339Nano),
					"duration":    d,
				}).Debug("Main cycle")
				time.Sleep(d)
			} else {
				c.runCurrentTurn(t)
			}
		}
	}
}

func (c *Chain) syncHead() {
	// Try to fetch if the the block of the current turn is not advised yet
	log.WithFields(log.Fields{
		"index":     c.rt.locSvIndex,
		"next_turn": c.rt.getNextTurn(),
		"height":    c.rt.head().height,
	}).Debug("sync header")
	if h := c.rt.getNextTurn() - 1; c.rt.head().height < h {
		log.Debugf("sync header with height %d", h)
		var err error
		req := &FetchBlockReq{
			Envelope: proto.Envelope{
				// TODO(lambda): Add fields.
			},
			Height: h,
		}
		resp := &FetchBlockResp{}
		peers := c.rt.getPeers()
		succ := false

		for i, s := range peers.Servers {
			if !s.IsEqual(&c.rt.nodeID) {
				err = c.cl.CallNode(s, route.MCCFetchBlock.String(), req, resp)
				if err != nil || resp.Block == nil {
					log.WithFields(log.Fields{
						"peer":        c.rt.peerInfo(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s),
						"curr_turn":   c.rt.getNextTurn(),
						"head_height": c.rt.head().height,
						"head_block":  c.rt.head().hash.String(),
					}).WithError(err).Debug("Failed to fetch block from peer")
				} else {
					c.pendingBlocks <- resp.Block
					log.WithFields(log.Fields{
						"peer":        c.rt.peerInfo(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s),
						"curr_turn":   c.rt.getNextTurn(),
						"head_height": c.rt.head().height,
						"head_block":  c.rt.head().hash.String(),
					}).Debug("Fetch block from remote peer successfully")
					succ = true
					break
				}
			}
		}

		if !succ {
			log.WithFields(log.Fields{
				"peer":        c.rt.peerInfo(),
				"curr_turn":   c.rt.getNextTurn(),
				"head_height": c.rt.head().height,
				"head_block":  c.rt.head().hash.String(),
			}).Debug(
				"Cannot get block from any peer")
		}
	}
}

// Stop stops the main process of the sql-chain.
func (c *Chain) Stop() (err error) {
	// Stop main process
	log.WithFields(log.Fields{"peer": c.rt.peerInfo()}).Debug("Stopping chain")
	c.rt.stop()
	log.WithFields(log.Fields{"peer": c.rt.peerInfo()}).Debug("Chain service stopped")
	c.st.Close()
	log.WithFields(log.Fields{"peer": c.rt.peerInfo()}).Debug("Chain database closed")
	close(c.pendingBlocks)
	close(c.pendingTxs)
	return
}
