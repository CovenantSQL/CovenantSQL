/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
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
				block.NonceChan <- bestNonce
				return
			}
			if currentDifficulty > bestNonce.Difficulty {
				bestNonce.Difficulty = currentDifficulty
				bestNonce.Nonce.Set(&i)
			}
		}
	}
}
