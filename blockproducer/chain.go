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
	"bytes"
	"encoding/binary"
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
	enc, err := pubKey.MarshalBinary()
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
		db: db,
		bi: newBlockIndex(),
		ti: newTxIndex(),
		rt: newRuntime(cfg, accountAddress),
		st: &state{},
		cl: rpc.NewCaller(),
	}

	chain.pushGenesisBlock(cfg.genesis)

	chain.rt.server.RegisterService(MainChainRPCName, chain)

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
	enc, err := pubKey.MarshalBinary()
	if err != nil {
		return nil, err
	}
	accountAddress = proto.AccountAddress(hash.THashH(enc[:]))

	chain = &Chain{
		db: db,
		bi: newBlockIndex(),
		ti: newTxIndex(),
		rt: newRuntime(cfg, accountAddress),
		st: &state{},
		cl: rpc.NewCaller(),
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
			reader := bytes.NewReader(k)
			err = utils.ReadElements(reader, binary.BigEndian, &databaseID)
			if err != nil {
				return err
			}

			var sequenceID uint64
			reader = bytes.NewReader(v)
			utils.ReadElements(reader, binary.BigEndian, &sequenceID)
			if err != nil {
				return err
			}

			chain.ti.updateLastTxBilling(&databaseID, sequenceID)
			return nil
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	chain.rt.server.RegisterService(MainChainRPCName, chain)

	return chain, nil
}

// checkTxBilling has two steps: 1. Hash 2. Signature 3. existed tx 4. SequenceID
func (c *Chain) checkTxBilling(tb *types.TxBilling) error {
	enc, err := tb.TxContent.MarshalBinary()
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

	enc, err := b.SignedHeader.Header.MarshalBinary()
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
	if err != err {
		return err
	}

	err = c.db.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket(metaBucket[:]).Put(metaStateKey, encState)
		if err != nil {
			return err
		}
		err = tx.Bucket(metaBucket[:]).Bucket(metaBlockIndexBucket).Put(node.indexKey(), encBlock)
		if err != nil {
			return err
		}
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
			buffer := bytes.NewBuffer(nil)
			err := utils.WriteElements(buffer, binary.BigEndian, tb.GetDatabaseID())
			if err != nil {
				return err
			}
			databaseID := buffer.Bytes()

			buffer.Reset()
			err = utils.WriteElements(buffer, binary.BigEndian, tb.GetSequenceID())
			if err != nil {
				return err
			}
			sequenceID := buffer.Bytes()
			err = meta.Bucket(metaLastTxBillingIndexBucket).Put(databaseID, sequenceID)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	c.ti.addTxBilling(tb)
	if tb.SignedBlock != nil {
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
		b.TxBillings[i].SignedBlock = &b.SignedHeader.BlockHash
	}

	err = c.pushBlock(b)

	return err
}

func (c *Chain) produceTxBilling(br *types.BillingRequest) (*types.BillingResponse, error) {
	err := c.checkBillingRequest(br)
	if err != nil {
		return nil, err
	}

	// update stable coin's balance
	// TODO(lambda): because there is no token distribution,
	// we only increase miners' balance but not decrease customer's balance
	accounts := make([]*types.Account, len(br.Header.GasAmounts))
	err = c.db.View(func(tx *bolt.Tx) error {
		accountBucket := tx.Bucket(metaBucket[:]).Bucket(metaAccountIndexBucket)
		for i, addrAndGas := range br.Header.GasAmounts {
			enc := accountBucket.Get(addrAndGas.AccountAddress[:])
			if len(enc) == 0 {
				accounts[i] = &types.Account{
					Address:            addrAndGas.AccountAddress,
					StableCoinBalance:  uint64(addrAndGas.GasAmount * gasprice),
					ThunderCoinBalance: 0,
					SQLChains:          []proto.DatabaseID{br.Header.DatabaseID},
					Roles:              []byte{types.Miner},
					Rating:             0.0,
				}
			} else {
				accounts[i].StableCoinBalance = accounts[i].StableCoinBalance + uint64(addrAndGas.GasAmount*gasprice)
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
			var dec types.Account
			err = dec.UnmarshalBinary(enc)
			if err != nil {
				return err
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
			enc, err := account.MarshalBinary()
			if err != nil {
				return err
			}
			accountBucket.Put(account.Address[:], enc)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	enc, err := br.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// generate response
	privKey, err := kms.GetLocalPrivateKey()
	if err != nil {
		return nil, err
	}
	h := hash.THashH(enc)
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
	enc, err := br.MarshalBinary()
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

func (c *Chain) fetchBlockByHeight(h uint64) (*types.Block, error) {
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

// Stop stops the main process of the sql-chain.
// func (c *Chain) Stop() (err error) {
// 	// Stop main process
// 	log.WithFields(log.Fields{"peer": c.rt.getPeerInfoString()}).Debug("Stopping chain")
// 	c.rt.stop()
// 	log.WithFields(log.Fields{"peer": c.rt.getPeerInfoString()}).Debug("Chain service stopped")
// 	// Close database file
// 	err = c.db.Close()
// 	log.WithFields(log.Fields{"peer": c.rt.getPeerInfoString()}).Debug("Chain database closed")
// 	return
// }
