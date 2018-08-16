/*
 * Copyright 2018 The ThunderDB Authors.
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

	"gitlab.com/thunderdb/ThunderDB/utils"

	"gitlab.com/thunderdb/ThunderDB/merkle"

	"gitlab.com/thunderdb/ThunderDB/rpc"

	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/proto"

	"github.com/coreos/bbolt"
	"gitlab.com/thunderdb/ThunderDB/blockproducer/types"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

var (
	metaBucket                   = [4]byte{0x0, 0x0, 0x0, 0x0}
	metaStateKey                 = []byte("thunderdb-state")
	metaBlockIndexBucket         = []byte("thunderdb-block-index-bucket")
	metaTxBillingIndexBucket     = []byte("thunderdb-tx-billing-index-bucket")
	metaLastTxBillingIndexBucket = []byte("thunderdb-last-tx-billing-index-bucket")
	metaAccountIndexBucket       = []byte("thunderdb-account-index-bucket")

	gasprice uint32 = 1

	accountAddress proto.AccountAddress
)

// Chain defines the main chain
type Chain struct {
	db *bolt.DB
	bi *blockIndex
	ti *txIndex
	rt *rt
	st *state
	cl *rpc.Caller

	blocksFromSelf    chan *types.Block
	blocksFromRPC     chan *types.Block
	txBillingFromSelf chan *types.TxBilling
	txBillingFromPRC  chan *types.TxBilling
	stopCh            chan struct{}
}

// NewChain creates a new blockchain
func NewChain(cfg *config) (*Chain, error) {
	// open db file
	db, err := bolt.Open(cfg.dataFile, 0600, nil)
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

		_, err = bucket.CreateBucketIfNotExists(metaTxBillingIndexBucket)
		if err != nil {
			return
		}

		_, err = bucket.CreateBucketIfNotExists(metaLastTxBillingIndexBucket)
		if err != nil {
			return
		}

		_, err = bucket.CreateBucketIfNotExists(metaAccountIndexBucket)
		return
	})
	if err != nil {
		return nil, err
	}

	// create chain
	chain := &Chain{
		db:                db,
		bi:                newBlockIndex(),
		ti:                newTxIndex(),
		rt:                newRuntime(cfg, accountAddress),
		st:                &state{},
		cl:                rpc.NewCaller(),
		blocksFromSelf:    make(chan *types.Block),
		blocksFromRPC:     make(chan *types.Block),
		txBillingFromSelf: make(chan *types.TxBilling),
		txBillingFromPRC:  make(chan *types.TxBilling),
		stopCh:            make(chan struct{}),
	}

	chain.pushGenesisBlock(cfg.genesis)

	return chain, nil
}

// LoadChain rebuilds the chain from db
func LoadChain(cfg *config) (chain *Chain, err error) {
	// open db file
	db, err := bolt.Open(cfg.dataFile, 0600, nil)
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
		db:                db,
		bi:                newBlockIndex(),
		ti:                newTxIndex(),
		rt:                newRuntime(cfg, accountAddress),
		st:                &state{},
		cl:                rpc.NewCaller(),
		blocksFromSelf:    make(chan *types.Block),
		blocksFromRPC:     make(chan *types.Block),
		txBillingFromSelf: make(chan *types.TxBilling),
		txBillingFromPRC:  make(chan *types.TxBilling),
		stopCh:            make(chan struct{}),
	}

	err = chain.db.View(func(tx *bolt.Tx) error {
		meta := tx.Bucket(metaBucket[:])
		err = chain.st.deserialize(meta.Get(metaStateKey))

		if err != nil {
			return err
		}

		var last *blockNode
		var index int32
		blocks := meta.Bucket(metaBlockIndexBucket)
		nodes := make([]blockNode, blocks.Stats().KeyN)

		err = blocks.ForEach(func(k, v []byte) (err error) {
			block := &types.Block{}

			if err = block.Deserialize(v); err != nil {
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
		})
		if err != nil {
			return err
		}

		txbillings := meta.Bucket(metaTxBillingIndexBucket)
		err = txbillings.ForEach(func(k, v []byte) error {
			txbilling := types.TxBilling{}
			err = txbilling.Deserialize(v)
			if err != nil {
				return err
			}
			chain.ti.addTxBilling(&txbilling)
			return err
		})

		lastTxBillings := meta.Bucket(metaLastTxBillingIndexBucket)
		err = lastTxBillings.ForEach(func(k, v []byte) error {
			var databaseID proto.DatabaseID
			err = utils.DecodeMsgPack(k, &databaseID)
			if err != nil {
				return err
			}

			var sequenceID uint32
			err = utils.DecodeMsgPack(v, &sequenceID)
			if err != nil {
				return err
			}

			chain.ti.updateLastTxBilling(&databaseID, sequenceID)
			return nil
		})

		return err
	})
	if err != nil {
		return nil, err
	}

	return chain, nil
}

// checkTxBilling has two steps: 1. Hash 2. Signature 3. existed tx 4. SequenceID
func (c *Chain) checkTxBilling(tb *types.TxBilling) error {
	enc, err := tb.TxContent.MarshalHash()
	if err != nil {
		return err
	}
	h := hash.THashH(enc)
	if !tb.TxHash.IsEqual(&h) {
		return ErrInvalidHash
	}

	err = tb.Verify(&h)
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

// checkBlock has following steps: 1. check parent block 2. checkTx 2. merkle tree 3. Hash 4. Signature
func (c *Chain) checkBlock(b *types.Block) error {
	// TODO(lambda): process block fork
	if !b.SignedHeader.ParentHash.IsEqual(&c.st.Head) {
		return ErrParentNotMatch
	}
	hashes := make([]*hash.Hash, len(b.TxBillings))
	for i := range b.TxBillings {
		err := c.checkTxBilling(b.TxBillings[i])
		if err != nil {
			return err
		}
		hashes[i] = b.TxBillings[i].TxHash
	}

	rootHash := merkle.NewMerkle(hashes).GetRoot()
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
	node := newBlockNode(b, c.st.node)
	state := state{
		node:   node,
		Head:   node.hash,
		Height: node.height,
	}

	encBlock, err := b.Serialize()
	if err != nil {
		return err
	}

	encState, err := c.st.serialize()
	if err != nil {
		return err
	}

	err = c.db.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket(metaBucket[:]).Put(metaStateKey, encState)
		if err != nil {
			return err
		}
		err = tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Put(node.indexKey(), encBlock)
		return err
	})
	if err != nil {
		return err
	}
	c.st = &state
	c.bi.addBlock(node)
	return nil

}

func (c *Chain) pushGenesisBlock(b *types.Block) error {
	err := c.pushBlockWithoutCheck(b)
	return err
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
				ParentHash: c.st.Head,
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
		if s.ID != c.rt.nodeID {
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
				}
			}(s.ID)
		}
	}

	return err
}

func (c *Chain) produceTxBilling(br *types.BillingRequest) (*types.BillingResponse, error) {
	// TODO(lambda): simplify the function
	err := c.checkBillingRequest(br)
	if err != nil {
		return nil, err
	}

	// update stable coin's balance
	// TODO(lambda): because there is no token distribution,
	// we only increase miners' balance but not decrease customer's balance
	accountNumber := len(br.Header.GasAmounts)
	receivers := make([]*proto.AccountAddress, accountNumber)
	fees := make([]uint64, accountNumber)
	rewards := make([]uint64, accountNumber)
	accounts := make([]*types.Account, accountNumber)

	err = c.db.View(func(tx *bolt.Tx) error {
		accountBucket := tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
		for i, addrAndGas := range br.Header.GasAmounts {
			receivers[i] = &addrAndGas.AccountAddress
			fees[i] = addrAndGas.GasAmount * uint64(gasprice)
			rewards[i] = 0

			enc := accountBucket.Get(addrAndGas.AccountAddress[:])
			if enc == nil {
				accounts[i] = &types.Account{
					Address:            addrAndGas.AccountAddress,
					StableCoinBalance:  addrAndGas.GasAmount * uint64(gasprice),
					ThunderCoinBalance: 0,
					SQLChains:          []proto.DatabaseID{br.Header.DatabaseID},
					Roles:              []byte{types.Miner},
					Rating:             0.0,
				}
			} else {
				var dec types.Account
				err = utils.DecodeMsgPack(enc, &dec)
				if err != nil {
					return err
				}
				accounts[i].StableCoinBalance = dec.StableCoinBalance + addrAndGas.GasAmount*uint64(gasprice)
				included := false
				for j := range accounts[i].SQLChains {
					if accounts[i].SQLChains[j] == br.Header.DatabaseID {
						included = true
						break
					}
				}
				if !included {
					accounts[i].SQLChains = append(accounts[i].SQLChains, br.Header.DatabaseID)
					accounts[i].Roles = append(accounts[i].Roles, types.Miner)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// update accounts
	err = c.db.Update(func(tx *bolt.Tx) error {
		accountBucket := tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
		for _, account := range accounts {
			enc, err := utils.EncodeMsgPack(account)
			if err != nil {
				return err
			}
			accountBucket.Put(account.Address[:], enc.Bytes())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	enc, err := br.MarshalHash()
	if err != nil {
		return nil, err
	}
	h := hash.THashH(enc)

	// generate response
	privKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		return nil, err
	}
	sign, err := privKey.Sign(h[:])
	if err != nil {
		return nil, err
	}
	resp := &types.BillingResponse{
		AccountAddress: accountAddress,
		RequestHash:    h,
		Signee:         privKey.PubKey(),
		Signature:      sign,
	}

	// generate and push the txbilling
	// 1. generate txbilling
	var seqID uint32
	var tc *types.TxContent
	var tb *types.TxBilling
	err = c.db.View(func(tx *bolt.Tx) error {
		meta := tx.Bucket(metaBucket[:])
		metaLastTB := meta.Bucket(metaLastTxBillingIndexBucket)

		// generate unique seqID
		encDatabaseID, err := utils.EncodeMsgPack(br.Header.DatabaseID)
		if err != nil {
			return err
		}
		oldSeqIDRaw := metaLastTB.Get(encDatabaseID.Bytes())
		if oldSeqIDRaw != nil {
			var oldSeqID uint32
			err = utils.DecodeMsgPack(oldSeqIDRaw, oldSeqID)
			if err != nil {
				return err
			}
			seqID = oldSeqID + 1
		} else {
			seqID = 0
		}

		// generate txbilling
		tc = types.NewTxContent(seqID, br, receivers, fees, rewards, resp)
		tb = types.NewTxBilling(tc, types.TxTypeBilling, &c.rt.accountAddress)
		tb.PackAndSignTx(privKey)

		return nil
	})
	if err != nil {
		return nil, err
	}
	log.Debugf("response is %s", resp.RequestHash)
	// 2. push tx
	c.txBillingFromSelf <- tb
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
		if s.ID != c.rt.nodeID {
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
// 3. miners' signatures
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
	node := c.st.node.ancestor(h)
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
	defer c.rt.setNextTurn()

	if !c.rt.isMyTurn() {
		return
	}

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

		if c.rt.getNextTurn() >= height {
			break
		}

		for c.rt.getNextTurn() <= height {
			// TODO(lambda): fetch blocks and txes.
			c.rt.setNextTurn()
		}
	}

	return nil
}

// Start starts the chain by step:
// 1. sync the chain
// 2. goroutine for getting blocks
// 3. goroutine for getting txes
func (c *Chain) Start() error {
	err := c.sync()
	if err != nil {
		return err
	}

	c.rt.wg.Add(1)
	go c.processBlocks()
	c.rt.wg.Add(1)
	go c.processTxBillings()
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

func (c *Chain) processTxBillings() {
	defer c.rt.wg.Done()
	for {
		select {
		case tb := <-c.txBillingFromSelf:
			err := c.pushTxBillingWithoutCheck(tb)
			if err != nil {
				log.WithFields(log.Fields{
					"peer":        c.rt.getPeerInfoString(),
					"next_turn":   c.rt.getNextTurn(),
					"head_height": c.st.Height,
					"head_block":  c.st.Head.String(),
					"tx_hash":     tb.TxHash,
				}).Debugf("Failed to push self-producing tx billing with error: %v", err)
			}
		case tb := <-c.txBillingFromPRC:
			err := c.pushTxBilling(tb)
			if err != nil {
				log.WithFields(log.Fields{
					"peer":        c.rt.getPeerInfoString(),
					"next_turn":   c.rt.getNextTurn(),
					"head_height": c.st.Height,
					"head_block":  c.st.Head.String(),
					"tx_hash":     tb.TxHash,
				}).Debugf("Failed to push rpc tx billing with error: %v", err)
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
					"head_height": c.st.Height,
					"head_block":  c.st.Head.String(),
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
	if h := c.rt.getNextTurn() - 1; c.st.Height < h {
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
			if s.ID != c.rt.nodeID {
				err = c.cl.CallNode(s.ID, method, req, resp)
				if err != nil || resp.Block == nil {
					log.WithFields(log.Fields{
						"peer":        c.rt.getPeerInfoString(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s.ID),
						"curr_turn":   c.rt.getNextTurn(),
						"head_height": c.st.Height,
						"head_block":  c.st.Head.String(),
					}).WithError(err).Debug(
						"Failed to fetch block from peer")
				} else {
					c.blocksFromRPC <- resp.Block
					log.WithFields(log.Fields{
						"peer":        c.rt.getPeerInfoString(),
						"remote":      fmt.Sprintf("[%d/%d] %s", i, len(peers.Servers), s.ID),
						"curr_turn":   c.rt.getNextTurn(),
						"head_height": c.st.Height,
						"head_block":  c.st.Head.String(),
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
				"head_height": c.st.Height,
				"head_block":  c.st.Head.String(),
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
