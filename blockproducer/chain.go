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
	"fmt"
	"sync"
	"time"

	pi "github.com/CovenantSQL/CovenantSQL/blockproducer/interfaces"
	"github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/merkle"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/coreos/bbolt"
)

var (
	metaBucket                          = [4]byte{0x0, 0x0, 0x0, 0x0}
	metaStateKey                        = []byte("covenantsql-state")
	metaBlockIndexBucket                = []byte("covenantsql-block-index-bucket")
	metaTransactionBucket               = []byte("covenantsql-tx-index-bucket")
	metaTxBillingIndexBucket            = []byte("covenantsql-tx-billing-index-bucket")
	metaLastTxBillingIndexBucket        = []byte("covenantsql-last-tx-billing-index-bucket")
	metaAccountIndexBucket              = []byte("covenantsql-account-index-bucket")
	metaSQLChainIndexBucket             = []byte("covenantsql-sqlchain-index-bucket")
	gasprice                     uint32 = 1
	accountAddress               proto.AccountAddress
)

// Chain defines the main chain.
type Chain struct {
	db *bolt.DB
	ms *metaState
	bi *blockIndex
	ti *txIndex
	rt *rt
	cl *rpc.Caller

	blocksFromSelf chan *types.Block
	blocksFromRPC  chan *types.Block
	pendingTxs     chan pi.Transaction
	stopCh         chan struct{}
}

// NewChain creates a new blockchain.
func NewChain(cfg *Config) (*Chain, error) {
	// open db file
	db, err := bolt.Open(cfg.DataFile, 0600, nil)
	if err != nil {
		return nil, err
	}

	// get accountAddress
	pubKey, err := kms.GetLocalPublicKey()
	if err != nil {
		return nil, err
	}
	enc, err := pubKey.MarshalHash()
	if err != nil {
		return nil, err
	}
	accountAddress = proto.AccountAddress(hash.THashH(enc[:]))

	// create bucket for meta data
	err = db.Update(func(tx *bolt.Tx) (err error) {
		bucket, err := tx.CreateBucketIfNotExists(metaBucket[:])

		if err != nil {
			return
		}

		_, err = bucket.CreateBucketIfNotExists(metaBlockIndexBucket)
		if err != nil {
			return
		}

		txbk, err := bucket.CreateBucketIfNotExists(metaTransactionBucket)
		if err != nil {
			return
		}
		for i := pi.TransactionType(0); i < pi.TransactionTypeNumber; i++ {
			if _, err = txbk.CreateBucketIfNotExists(i.Bytes()); err != nil {
				return
			}
		}

		_, err = bucket.CreateBucketIfNotExists(metaTxBillingIndexBucket)
		if err != nil {
			return
		}

		_, err = bucket.CreateBucketIfNotExists(metaLastTxBillingIndexBucket)
		if err != nil {
			return
		}

		_, err = bucket.CreateBucketIfNotExists(metaAccountIndexBucket)
		if err != nil {
			return
		}

		_, err = bucket.CreateBucketIfNotExists(metaSQLChainIndexBucket)
		return
	})
	if err != nil {
		return nil, err
	}

	// create chain
	chain := &Chain{
		db:             db,
		ms:             newMetaState(),
		bi:             newBlockIndex(),
		ti:             newTxIndex(),
		rt:             newRuntime(cfg, accountAddress),
		cl:             rpc.NewCaller(),
		blocksFromSelf: make(chan *types.Block),
		blocksFromRPC:  make(chan *types.Block),
		pendingTxs:     make(chan pi.Transaction),
		stopCh:         make(chan struct{}),
	}

	log.Debugf("pushing genesis block: %v", cfg.Genesis)

	if err = chain.pushGenesisBlock(cfg.Genesis); err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"index":     chain.rt.index,
		"bp_number": chain.rt.bpNum,
		"period":    chain.rt.period.String(),
		"tick":      chain.rt.tick.String(),
		"head":      chain.rt.getHead().getHeader().String(),
		"height":    chain.rt.getHead().getHeight(),
	}).Debug("current chain state")

	return chain, nil
}

// LoadChain rebuilds the chain from db.
func LoadChain(cfg *Config) (chain *Chain, err error) {
	// open db file
	db, err := bolt.Open(cfg.DataFile, 0600, nil)
	if err != nil {
		return nil, err
	}

	// get accountAddress
	pubKey, err := kms.GetLocalPublicKey()
	if err != nil {
		return nil, err
	}
	enc, err := pubKey.MarshalHash()
	if err != nil {
		return nil, err
	}
	accountAddress = proto.AccountAddress(hash.THashH(enc[:]))

	chain = &Chain{
		db:             db,
		ms:             newMetaState(),
		bi:             newBlockIndex(),
		ti:             newTxIndex(),
		rt:             newRuntime(cfg, accountAddress),
		cl:             rpc.NewCaller(),
		blocksFromSelf: make(chan *types.Block),
		blocksFromRPC:  make(chan *types.Block),
		pendingTxs:     make(chan pi.Transaction),
		stopCh:         make(chan struct{}),
	}

	err = chain.db.View(func(tx *bolt.Tx) (err error) {
		meta := tx.Bucket(metaBucket[:])
		state := &State{}
		if err = state.deserialize(meta.Get(metaStateKey)); err != nil {
			return
		}
		chain.rt.setHead(state)

		var last *blockNode
		var index int32
		blocks := meta.Bucket(metaBlockIndexBucket)
		nodes := make([]blockNode, blocks.Stats().KeyN)

		if err = blocks.ForEach(func(k, v []byte) (err error) {
			block := &types.Block{}

			if err = block.Deserialize(v); err != nil {
				log.Errorf("loadeing block: %v", err)
				return err
			}

			parent := (*blockNode)(nil)

			if last == nil {
				// TODO(lambda): check genesis block
			} else if block.SignedHeader.ParentHash.IsEqual(&last.hash) {
				if err = block.SignedHeader.Verify(); err != nil {
					return err
				}

				parent = last
			} else {
				parent = chain.bi.lookupBlock(block.SignedHeader.BlockHash)

				if parent == nil {
					return ErrParentNotFound
				}
			}

			nodes[index].initBlockNode(block, parent)
			last = &nodes[index]
			index++
			return err
		}); err != nil {
			return err
		}

		txbillings := meta.Bucket(metaTxBillingIndexBucket)
		if err = txbillings.ForEach(func(k, v []byte) (err error) {
			txbilling := types.TxBilling{}
			err = txbilling.Deserialize(v)
			if err != nil {
				return
			}
			chain.ti.addTxBilling(&txbilling)
			return
		}); err != nil {
			return
		}

		lastTxBillings := meta.Bucket(metaLastTxBillingIndexBucket)
		if err = lastTxBillings.ForEach(func(k, v []byte) (err error) {
			var databaseID proto.DatabaseID
			err = utils.DecodeMsgPack(k, &databaseID)
			if err != nil {
				return
			}

			var sequenceID uint32
			err = utils.DecodeMsgPack(v, &sequenceID)
			if err != nil {
				return
			}

			chain.ti.updateLastTxBilling(&databaseID, sequenceID)
			return
		}); err != nil {
			return
		}

		// Reload state
		if err = chain.ms.reloadProcedure()(tx); err != nil {
			return
		}

		return
	})
	if err != nil {
		return nil, err
	}

	return chain, nil
}

// checkTxBilling has two steps: 1. Hash 2. Signature 3. existed tx 4. SequenceID.
func (c *Chain) checkTxBilling(tb *types.TxBilling) (err error) {
	err = tb.Verify()
	if err != nil {
		return err
	}

	if val := c.ti.getTxBilling(tb.TxHash); val == nil {
		err = c.db.View(func(tx *bolt.Tx) error {
			meta := tx.Bucket(metaBucket[:])
			dec := meta.Bucket(metaTxBillingIndexBucket).Get(tb.TxHash[:])
			if len(dec) != 0 {
				decTx := &types.TxBilling{}
				err = decTx.Deserialize(dec)
				if err != nil {
					return err
				}

				if decTx != nil && (!decTx.SignedBlock.IsEqual(tb.SignedBlock)) {
					return ErrExistedTx
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		if val.SignedBlock != nil && (!val.SignedBlock.IsEqual(tb.SignedBlock)) {
			return ErrExistedTx
		}
	}

	// check sequence ID to avoid double rewards and fees
	databaseID := tb.GetDatabaseID()
	sequenceID, err := c.ti.lastSequenceID(databaseID)
	if err == nil {
		if sequenceID >= tb.GetSequenceID() {
			return ErrSmallerSequenceID
		}
	}

	return nil
}

// checkBlock has following steps: 1. check parent block 2. checkTx 2. merkle tree 3. Hash 4. Signature.
func (c *Chain) checkBlock(b *types.Block) (err error) {
	// TODO(lambda): process block fork
	if !b.SignedHeader.ParentHash.IsEqual(c.rt.getHead().getHeader()) {
		log.WithFields(log.Fields{
			"head":            c.rt.getHead().getHeader().String(),
			"height":          c.rt.getHead().getHeight(),
			"received_parent": b.SignedHeader.ParentHash,
		}).Debug("invalid parent")
		return ErrParentNotMatch
	}

	// TODO(leventeliu): merge transactions checking.
	for i := range b.TxBillings {
		if err = c.checkTxBilling(b.TxBillings[i]); err != nil {
			return err
		}
	}

	rootHash := merkle.NewMerkle(b.GetTxHashes()).GetRoot()
	if !b.SignedHeader.MerkleRoot.IsEqual(rootHash) {
		return ErrInvalidMerkleTreeRoot
	}

	enc, err := b.SignedHeader.Header.MarshalHash()
	if err != nil {
		return err
	}
	h := hash.THashH(enc)
	if !b.SignedHeader.BlockHash.IsEqual(&h) {
		return ErrInvalidHash
	}

	return nil
}

func (c *Chain) pushBlockWithoutCheck(b *types.Block) error {
	h := c.rt.getHeightFromTime(b.Timestamp())
	node := newBlockNode(h, b, c.rt.getHead().getNode())
	state := &State{
		Node:   node,
		Head:   node.hash,
		Height: node.height,
	}

	encBlock, err := b.Serialize()
	if err != nil {
		return err
	}

	encState, err := c.rt.getHead().serialize()
	if err != nil {
		return err
	}

	err = c.db.Update(func(tx *bolt.Tx) (err error) {
		err = tx.Bucket(metaBucket[:]).Put(metaStateKey, encState)
		if err != nil {
			return err
		}
		err = tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Put(node.indexKey(), encBlock)
		if err != nil {
			return err
		}
		for _, v := range b.Transactions {
			if err = c.ms.applyTransactionProcedure(v)(tx); err != nil {
				return err
			}
		}
		err = c.ms.partialCommitProcedure(b.Transactions)(tx)
		return
	})
	if err != nil {
		return err
	}
	c.rt.setHead(state)
	c.bi.addBlock(node)
	return nil
}

func (c *Chain) pushGenesisBlock(b *types.Block) (err error) {
	err = c.pushBlockWithoutCheck(b)
	if err != nil {
		log.Errorf("push genesis block failed: %v", err)
	}
	return
}

func (c *Chain) pushBlock(b *types.Block) error {
	err := c.checkBlock(b)
	if err != nil {
		return err
	}

	err = c.pushBlockWithoutCheck(b)
	if err != nil {
		return err
	}

	for i := range b.TxBillings {
		err = c.pushTxBillingWithoutCheck(b.TxBillings[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Chain) pushTxBillingWithoutCheck(tb *types.TxBilling) error {
	encTx, err := tb.Serialize()
	if err != nil {
		return err
	}

	err = c.db.Update(func(tx *bolt.Tx) error {
		meta := tx.Bucket(metaBucket[:])
		err = meta.Bucket(metaTxBillingIndexBucket).Put(tb.TxHash[:], encTx)
		if err != nil {
			return err
		}

		// if the tx is packed in some block, its nonce should be stored to ensure nonce is monotone increasing
		if tb.SignedBlock != nil {
			databaseID, err := utils.EncodeMsgPack(tb.GetDatabaseID())
			if err != nil {
				return err
			}

			sequenceID, err := utils.EncodeMsgPack(tb.GetSequenceID())
			if err != nil {
				return err
			}
			err = meta.Bucket(metaLastTxBillingIndexBucket).Put(databaseID.Bytes(), sequenceID.Bytes())
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	c.ti.addTxBilling(tb)
	if tb.IsSigned() {
		c.ti.updateLastTxBilling(tb.GetDatabaseID(), tb.GetSequenceID())
	}
	return nil
}

func (c *Chain) pushTxBilling(tb *types.TxBilling) error {
	err := c.checkTxBilling(tb)
	if err != nil {
		return err
	}

	err = c.pushTxBillingWithoutCheck(tb)
	return err
}

func (c *Chain) produceBlock(now time.Time) error {
	priv, err := kms.GetLocalPrivateKey()
	if err != nil {
		return err
	}

	b := &types.Block{
		SignedHeader: types.SignedHeader{
			Header: types.Header{
				Version:    blockVersion,
				Producer:   c.rt.accountAddress,
				ParentHash: *c.rt.getHead().getHeader(),
				Timestamp:  now,
			},
		},
		TxBillings: c.ti.fetchUnpackedTxBillings(),
	}

	err = b.PackAndSignBlock(priv)
	if err != nil {
		return err
	}

	for i := range b.TxBillings {
		b.TxBillings[i].SetSignedBlock(&b.SignedHeader.BlockHash)
	}

	log.Debugf("generate new block: %v", b)

	err = c.pushBlockWithoutCheck(b)
	if err != nil {
		return err
	}

	blockReq := &AdviseNewBlockReq{
		Envelope: proto.Envelope{
			// TODO(lambda): Add fields.
		},
		Block: b,
	}
	method := fmt.Sprintf("%s.%s", MainChainRPCName, "AdviseNewBlock")
	peers := c.rt.getPeers()
	wg := &sync.WaitGroup{}
	for _, s := range peers.Servers {
		if !s.ID.IsEqual(&c.rt.nodeID) {
			wg.Add(1)
			go func(id proto.NodeID) {
				defer wg.Done()
				blockResp := &AdviseNewBlockResp{}
				if err = c.cl.CallNode(id, method, blockReq, blockResp); err != nil {
					log.WithFields(log.Fields{
						"peer":       c.rt.getPeerInfoString(),
						"curr_turn":  c.rt.getNextTurn(),
						"now_time":   time.Now().UTC().Format(time.RFC3339Nano),
						"block_hash": b.SignedHeader.BlockHash,
					}).WithError(err).Error(
						"Failed to advise new block")
				} else {
					log.Debugf("Success to advising #%d height block to %s", c.rt.getHead().getHeight(), id)
				}
			}(s.ID)
		}
	}

	return err
}

func (c *Chain) produceTxBilling(br *types.BillingRequest) (_ *types.BillingResponse, err error) {
	// TODO(lambda): simplify the function
	if err = c.checkBillingRequest(br); err != nil {
		return
	}

	// update stable coin's balance
	// TODO(lambda): because there is no token distribution,
	// we only increase miners' balance but not decrease customer's balance
	var (
		enc           []byte
		accountNumber = len(br.Header.GasAmounts)
		receivers     = make([]*proto.AccountAddress, accountNumber)
		fees          = make([]uint64, accountNumber)
		rewards       = make([]uint64, accountNumber)
	)

	for i, addrAndGas := range br.Header.GasAmounts {
		receivers[i] = &addrAndGas.AccountAddress
		fees[i] = addrAndGas.GasAmount * uint64(gasprice)
		rewards[i] = 0
	}

	if enc, err = br.MarshalHash(); err != nil {
		return
	}
	h := hash.THashH(enc)

	// generate response
	privKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		return
	}
	sign, err := privKey.Sign(h[:])
	if err != nil {
		return
	}
	resp := &types.BillingResponse{
		AccountAddress: accountAddress,
		RequestHash:    h,
		Signee:         privKey.PubKey(),
		Signature:      sign,
	}

	// generate and push the txbilling
	// 1. generate txbilling
	var nc pi.AccountNonce
	if nc, err = c.ms.nextNonce(accountAddress); err != nil {
		return
	}
	var (
		tc = types.NewTxContent(uint32(nc), br, receivers, fees, rewards, resp)
		tb = types.NewTxBilling(tc, types.TxTypeBilling, &c.rt.accountAddress)
	)
	if err = tb.Sign(privKey); err != nil {
		return
	}
	log.Debugf("response is %s", resp.RequestHash)

	// 2. push tx
	c.pendingTxs <- tb
	tbReq := &AdviseTxBillingReq{
		Envelope: proto.Envelope{
			// TODO(lambda): Add fields.
		},
		TxBilling: tb,
	}
	method := fmt.Sprintf("%s.%s", MainChainRPCName, "AdviseTxBilling")
	peers := c.rt.getPeers()
	wg := &sync.WaitGroup{}
	for _, s := range peers.Servers {
		if !s.ID.IsEqual(&c.rt.nodeID) {
			wg.Add(1)
			go func(id proto.NodeID) {
				defer wg.Done()
				tbResp := &AdviseTxBillingResp{}
				if err = c.cl.CallNode(id, method, tbReq, tbResp); err != nil {
					log.WithFields(log.Fields{
						"peer":      c.rt.getPeerInfoString(),
						"curr_turn": c.rt.getNextTurn(),
						"now_time":  time.Now().UTC().Format(time.RFC3339Nano),
						"tx_hash":   tb.TxHash,
					}).WithError(err).Error(
						"Failed to advise new block")
				}
			}(s.ID)
		}
	}
	wg.Wait()

	return resp, nil
}

// checkBillingRequest checks followings by order:
// 1. period of sqlchain;
// 2. request's hash
// 3. miners' signatures.
func (c *Chain) checkBillingRequest(br *types.BillingRequest) error {
	// period of sqlchain;
	// TODO(lambda): get and check period and miner list of specific sqlchain

	// request's hash
	enc, err := br.Header.MarshalHash()
	if err != nil {
		return err
	}
	h := hash.THashH(enc[:])
	if !h.IsEqual(&br.RequestHash) {
		return ErrInvalidHash
	}

	// miners' signatures
	sLen := len(br.Signees)
	if sLen != len(br.Signatures) {
		return ErrInvalidBillingRequest
	}
	for i := range br.Signees {
		if !br.Signatures[i].Verify(h[:], br.Signees[i]) {
			return ErrSignVerification
		}
	}

	return nil
}

func (c *Chain) fetchBlockByHeight(h uint32) (*types.Block, error) {
	node := c.rt.getHead().getNode().ancestor(h)
	if node == nil {
		return nil, ErrNoSuchBlock
	}

	b := &types.Block{}
	k := node.indexKey()

	err := c.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Get(k)
		return b.Deserialize(v)
	})
	if err != nil {
		return nil, err
	}

	return b, nil
}

// runCurrentTurn does the check and runs block producing if its my turn.
func (c *Chain) runCurrentTurn(now time.Time) {
	log.WithFields(log.Fields{
		"next_turn":  c.rt.getNextTurn(),
		"bp_number":  c.rt.bpNum,
		"node_index": c.rt.index,
	}).Info("check turns")
	defer c.rt.setNextTurn()

	if !c.rt.isMyTurn() {
		return
	}

	log.Infof("produce a new block with height %d", c.rt.getNextTurn())
	if err := c.produceBlock(now); err != nil {
		log.WithField("now", now.Format(time.RFC3339Nano)).WithError(err).Errorln(
			"Failed to produce block")
	}
}

// sync synchronizes blocks and queries from the other peers.
func (c *Chain) sync() error {
	log.WithFields(log.Fields{
		"peer": c.rt.getPeerInfoString(),
	}).Debug("Synchronizing chain state")

	for {
		now := c.rt.now()
		height := c.rt.getHeightFromTime(now)

		log.Infof("current height is %d, next turn is %d", height, c.rt.getNextTurn())
		if c.rt.getNextTurn() >= height {
			log.Infof("return with height %d, next turn is %d", height, c.rt.getNextTurn())
			break
		}

		for c.rt.getNextTurn() <= height {
			// TODO(lambda): fetch blocks and txes.
			c.rt.setNextTurn()
			// TODO(lambda): remove it after implementing fetch
			c.rt.getHead().increaseHeightByOne()
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

	c.rt.wg.Add(1)
	go c.processBlocks()
	c.rt.wg.Add(1)
	go c.processTxs()
	c.rt.wg.Add(1)
	go c.mainCycle()
	c.rt.startService(c)

	return nil
}

func (c *Chain) processBlocks() {
	rsCh := make(chan struct{})
	rsWG := &sync.WaitGroup{}
	returnStash := func(stash []*types.Block) {
		defer rsWG.Done()
		for _, block := range stash {
			select {
			case c.blocksFromRPC <- block:
			case <-rsCh:
				return
			}
		}
	}

	defer func() {
		close(rsCh)
		rsWG.Wait()
		c.rt.wg.Done()
	}()

	var stash []*types.Block
	for {
		select {
		case block := <-c.blocksFromSelf:
			h := c.rt.getHeightFromTime(block.Timestamp())
			if h == c.rt.getNextTurn()-1 {
				err := c.pushBlockWithoutCheck(block)
				if err != nil {
					log.Error(err)
				}
			}
		case block := <-c.blocksFromRPC:
			if h := c.rt.getHeightFromTime(block.Timestamp()); h > c.rt.getNextTurn()-1 {
				// Stash newer blocks for later check
				if stash == nil {
					stash = make([]*types.Block, 0)
				}
				stash = append(stash, block)
			} else {
				// Process block
				if h < c.rt.getNextTurn()-1 {
					// TODO(lambda): check and add to fork list.
				} else {
					err := c.pushBlock(block)
					if err != nil {
						log.Error(err)
					}
				}

				// Return all stashed blocks to pending channel
				if stash != nil {
					rsWG.Add(1)
					go returnStash(stash)
					stash = nil
				}
			}
		case <-c.stopCh:
			return
		}
	}
}

func (c *Chain) processTx(tx pi.Transaction) (err error) {
	return c.db.Update(c.ms.applyTransactionProcedure(tx))
}

func (c *Chain) processTxs() {
	defer c.rt.wg.Done()
	for {
		select {
		case tx := <-c.pendingTxs:
			if err := c.processTx(tx); err != nil {
				log.WithFields(log.Fields{
					"peer":        c.rt.getPeerInfoString(),
					"next_turn":   c.rt.getNextTurn(),
					"head_height": c.rt.getHead().getHeight(),
					"head_block":  c.rt.getHead().getHeader().String(),
					"transaction": tx.GetHash().String(),
				}).Debugf("Failed to push tx with error: %v", err)
			}
		case <-c.stopCh:
			return
		}
	}
}

func (c *Chain) mainCycle() {
	defer func() {
		c.rt.wg.Done()
		// Signal worker goroutines to stop
		close(c.stopCh)
	}()

	for {
		select {
		case <-c.rt.stopCh:
			return
		default:
			c.syncHead()

			if t, d := c.rt.nextTick(); d > 0 {
				log.WithFields(log.Fields{
					"peer":        c.rt.getPeerInfoString(),
					"next_turn":   c.rt.getNextTurn(),
					"head_height": c.rt.getHead().getHeight(),
					"head_block":  c.rt.getHead().getHeader().String(),
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
		"index":     c.rt.index,
		"next_turn": c.rt.getNextTurn(),
		"height":    c.rt.getHead().getHeight(),
		"count":     c.rt.getHead().getNode().count,
	}).Debugf("sync header")
	if h := c.rt.getNextTurn() - 1; c.rt.getHead().getHeight() < h {
		var err error
		req := &FetchBlockReq{
			Envelope: proto.Envelope{
				// TODO(lambda): Add fields.
			},
			Height: h,
		}
		resp := &FetchBlockResp{}
		method := fmt.Sprintf("%s.%s", MainChainRPCName, "FetchBlock")
		peers := c.rt.getPeers()
		succ := false

		for i, s := range peers.Servers {
			if !s.ID.IsEqual(&c.rt.nodeID) {
				err = c.cl.CallNode(s.ID, method, req, resp)
				if err != nil || resp.Block == nil {
					log.WithFields(log.Fields{
						"peer":        c.rt.getPeerInfoString(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s.ID),
						"curr_turn":   c.rt.getNextTurn(),
						"head_height": c.rt.getHead().getHeight(),
						"head_block":  c.rt.getHead().getHeader().String(),
					}).WithError(err).Debug(
						"Failed to fetch block from peer")
				} else {
					c.blocksFromRPC <- resp.Block
					log.WithFields(log.Fields{
						"peer":        c.rt.getPeerInfoString(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s.ID),
						"curr_turn":   c.rt.getNextTurn(),
						"head_height": c.rt.getHead().getHeight(),
						"head_block":  c.rt.getHead().getHeader().String(),
					}).Debug(
						"Fetch block from remote peer successfully")
					succ = true
					break
				}
			}
		}

		if !succ {
			log.WithFields(log.Fields{
				"peer":        c.rt.getPeerInfoString(),
				"curr_turn":   c.rt.getNextTurn(),
				"head_height": c.rt.getHead().getHeight(),
				"head_block":  c.rt.getHead().getHeader().String(),
			}).Debug(
				"Cannot get block from any peer")
		}
	}
}

// Stop stops the main process of the sql-chain.
func (c *Chain) Stop() (err error) {
	// Stop main process
	log.WithFields(log.Fields{"peer": c.rt.getPeerInfoString()}).Debug("Stopping chain")
	c.rt.stop()
	log.WithFields(log.Fields{"peer": c.rt.getPeerInfoString()}).Debug("Chain service stopped")
	// Close database file
	err = c.db.Close()
	log.WithFields(log.Fields{"peer": c.rt.getPeerInfoString()}).Debug("Chain database closed")
	return
}
