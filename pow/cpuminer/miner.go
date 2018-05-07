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

// Package cpuminer implements CPU based PoW functions
package cpuminer

import (
	"errors"
	"math/big"

	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/crypto/hash"
)

// Nonce contains nonce and the difficulty to the block
type Nonce struct {
	Nonce      big.Int
	Difficulty int
	Hash       hash.Hash
}

// MiningBlock contains Data tobe mined
type MiningBlock struct {
	Data []byte
	// NonceChan is used to notify the got nonce
	NonceChan chan Nonce
	// Stop chan is used to stop mining and return the max difficult nonce
	Stop chan struct{}
}

// CPUMiner provides concurrency-safe PoW worker group to solve hash puzzle
// Inspired by:
// 	"S/Kademlia: A Practicable Approach Towards Secure Key-Based Routing"
// 	- Section 4.1. Secure nodeId assignment.
// 	- Figure 3. Static (left) and dynamic (right) crypto puzzles for nodeId
// 		generation
type CPUMiner struct {
	quit chan struct{}
}

// NewCPUMiner init a new CPU miner
func NewCPUMiner(quit chan struct{}) *CPUMiner {
	return &CPUMiner{quit: quit}
}

// HashBlock calculate the hash of MiningBlock
func (miner *CPUMiner) HashBlock(data []byte, nonce big.Int) hash.Hash {
	return hash.DoubleHashH(append(data, nonce.Bytes()...))
}

// CalculateBlockNonce find nonce make HashBlock() match the MiningBlock Difficulty from the startNonce
// if interrupted or stopped highest difficulty nonce will be sent to the NonceCh
//  todo: make calculation parallel
func (miner *CPUMiner) CalculateBlockNonce(
	block MiningBlock,
	startNonce big.Int,
	difficulty int,
) (err error) {

	var (
		bestNonce Nonce
	)
	for i := startNonce; ; i.Add(&i, big.NewInt(1)) {
		select {
		case <-block.Stop:
			log.Info("Stop mining job")
			block.NonceChan <- bestNonce
			return errors.New("mining job stopped")
		case <-miner.quit:
			log.Info("Stop mining worker")
			block.NonceChan <- bestNonce
			return errors.New("miner interrupted")
		default:
			currentHash := miner.HashBlock(block.Data, i)
			currentDifficulty := currentHash.Difficulty()
			if currentDifficulty >= difficulty {
				bestNonce.Difficulty = currentDifficulty
				bestNonce.Nonce.Set(&i)
				bestNonce.Hash.SetBytes(currentHash[:])
				block.NonceChan <- bestNonce
				return
			}
			if currentDifficulty > bestNonce.Difficulty {
				bestNonce.Difficulty = currentDifficulty
				bestNonce.Nonce.Set(&i)
				bestNonce.Hash.SetBytes(currentHash[:])
			}
		}
	}
}
