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

// Package cpuminer implements CPU based PoW functions
package cpuminer

import (
	"errors"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// NonceInfo contains nonce and the difficulty to the block
type NonceInfo struct {
	Nonce      Uint256
	Difficulty int
	Hash       hash.Hash // Hash can be used as raw NodeID
}

// MiningBlock contains Data tobe mined
type MiningBlock struct {
	Data []byte
	// NonceChan is used to notify the got nonce
	NonceChan chan NonceInfo
	// Stop chan is used to stop mining and return the max difficult nonce
	Stop chan struct{}
}

// CPUMiner provides concurrency-safe PoW worker group to solve hash puzzle
// Inspired by:
// 	"S/Kademlia: A Practicable Approach Towards Secure Key-Based Routing"
// 	- Section 4.1. Secure nodeID assignment.
// 	- Figure 3. Static (left) and dynamic (right) crypto puzzles for nodeID
// 		generation
type CPUMiner struct {
	quit chan struct{}
}

// NewCPUMiner init A new CPU miner
func NewCPUMiner(quit chan struct{}) *CPUMiner {
	return &CPUMiner{quit: quit}
}

// HashBlock calculate the hash of MiningBlock
func HashBlock(data []byte, nonce Uint256) hash.Hash {
	return hash.THashH(append(data, nonce.Bytes()...))
}

// ComputeBlockNonce find nonce make HashBlock() match the MiningBlock Difficulty from the startNonce
// if interrupted or stopped highest difficulty nonce will be sent to the NonceCh
//  HACK(auxten): make calculation parallel
func (miner *CPUMiner) ComputeBlockNonce(
	block MiningBlock,
	startNonce Uint256,
	difficulty int,
) (err error) {

	var (
		bestNonce NonceInfo
	)
	for i := startNonce; ; i.Inc() {
		select {
		case <-block.Stop:
			log.Info("stop block nonce job")
			block.NonceChan <- bestNonce
			return errors.New("mining job stopped")
		case <-miner.quit:
			log.Info("stop block nonce worker")
			block.NonceChan <- bestNonce
			return errors.New("miner interrupted")
		default:
			currentHash := HashBlock(block.Data, i)
			currentDifficulty := currentHash.Difficulty()
			if currentDifficulty >= difficulty {
				bestNonce.Difficulty = currentDifficulty
				bestNonce.Nonce = i
				bestNonce.Hash.SetBytes(currentHash[:])
				block.NonceChan <- bestNonce
				return
			}
			if currentDifficulty > bestNonce.Difficulty {
				bestNonce.Difficulty = currentDifficulty
				bestNonce.Nonce = i
				bestNonce.Hash.SetBytes(currentHash[:])
			}
		}
	}
}
