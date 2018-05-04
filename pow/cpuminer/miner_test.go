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

package cpuminer

import (
	"math/big"
	"testing"

	"time"

	"github.com/thunderdb/ThunderDB/crypto/hash"
)

func TestCPUMiner_HashBlock(t *testing.T) {
	miner := NewCPUMiner(make(chan struct{}))
	nonceCh := make(chan Nonce)
	stop := make(chan struct{})
	diffWanted := 20
	data := []byte{
		0x79, 0xa6, 0x1a, 0xdb, 0xc6, 0xe5, 0xa2, 0xe1,
		0x39, 0xd2, 0x71, 0x3a, 0x54, 0x6e, 0xc7, 0xc8,
		0x75, 0x63, 0x2e, 0x75, 0xf1, 0xdf, 0x9c, 0x3f,
		0xa6, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	block := MiningBlock{
		Data:      data,
		NonceChan: nonceCh,
		Stop:      stop,
	}
	var (
		err error
	)
	go func() {
		err = miner.CalculateBlockNonce(block, *big.NewInt(0), diffWanted)
	}()
	nonceFromCh := <-nonceCh

	hash := hash.DoubleHashH(append(data, nonceFromCh.Nonce.Bytes()...))
	if err != nil || nonceFromCh.Difficulty < diffWanted || hash.Difficulty() < diffWanted {
		t.Errorf("CalculateBlockNonce got %v, difficulty %d, nonce %s",
			err, nonceFromCh.Difficulty, nonceFromCh.Nonce.String())
	}
	t.Logf("Difficulty: %d, Hash: %s", nonceFromCh.Difficulty, hash.String())
}

func TestCPUMiner_HashBlock_stop(t *testing.T) {
	minerQuit := make(chan struct{})
	miner := NewCPUMiner(minerQuit)
	nonceCh := make(chan Nonce)
	stop := make(chan struct{})
	diffWanted := 256
	data := []byte{
		0x79, 0xa6,
	}
	block := MiningBlock{
		Data:      data,
		NonceChan: nonceCh,
		Stop:      stop,
	}
	var (
		err error
	)
	go func() {
		err = miner.CalculateBlockNonce(block, *big.NewInt(0), diffWanted)
	}()
	// stop miner
	time.Sleep(2 * time.Second)
	block.Stop <- struct{}{}
	//miner.quit <- struct{}{}

	nonceFromCh := <-block.NonceChan

	//hasha := miner.HashBlock(data, nonceFromCh.Nonce)
	hasha := hash.DoubleHashH(append(data, nonceFromCh.Nonce.Bytes()...))
	if nonceFromCh.Difficulty < 1 || hasha.Difficulty() != nonceFromCh.Difficulty {
		t.Errorf("CalculateBlockNonce got %v, difficulty %d, nonce %s, hash %s",
			err, nonceFromCh.Difficulty, nonceFromCh.Nonce.String(), hasha.String())
	}
	t.Logf("Difficulty: %d, Hash: %s", nonceFromCh.Difficulty, hasha.String())
}

func TestCPUMiner_HashBlock_quit(t *testing.T) {
	minerQuit := make(chan struct{})
	miner := NewCPUMiner(minerQuit)
	nonceCh := make(chan Nonce)
	stop := make(chan struct{})
	diffWanted := 256
	data := []byte{
		0x79, 0xa6,
	}
	block := MiningBlock{
		Data:      data,
		NonceChan: nonceCh,
		Stop:      stop,
	}
	var (
		err error
	)
	go func() {
		err = miner.CalculateBlockNonce(block, *big.NewInt(0), diffWanted)
	}()
	// stop miner
	time.Sleep(2 * time.Second)
	//block.Stop <- struct{}{}
	miner.quit <- struct{}{}

	nonceFromCh := <-block.NonceChan

	//hasha := miner.HashBlock(data, nonceFromCh.Nonce)
	hasha := hash.DoubleHashH(append(data, nonceFromCh.Nonce.Bytes()...))
	if nonceFromCh.Difficulty < 1 || hasha.Difficulty() != nonceFromCh.Difficulty {
		t.Errorf("CalculateBlockNonce got %v, difficulty %d, nonce %s, hash %s",
			err, nonceFromCh.Difficulty, nonceFromCh.Nonce.String(), hasha.String())
	}
	t.Logf("Difficulty: %d, Hash: %s", nonceFromCh.Difficulty, hasha.String())
}
