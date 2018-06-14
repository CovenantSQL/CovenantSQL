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

package asymmetric

import (
	"time"

	ec "github.com/btcsuite/btcd/btcec"
	log "github.com/sirupsen/logrus"
	mine "github.com/thunderdb/ThunderDB/pow/cpuminer"
)

// GenSecp256k1KeyPair generate Secp256k1(used by Bitcoin) key pair
func GenSecp256k1KeyPair() (
	privateKey *ec.PrivateKey,
	publicKey *ec.PublicKey,
	err error) {

	privateKey, err = ec.NewPrivateKey(ec.S256())
	if err != nil {
		log.Errorf("private key generation error: %s", err)
		return nil, nil, err
	}
	publicKey = privateKey.PubKey()
	return
}

// GetPubKeyNonce will make his best effort to find a difficult enough
// nonce.
func GetPubKeyNonce(
	publicKey *ec.PublicKey,
	difficulty int,
	timeThreshold time.Duration,
	quit chan struct{}) (nonce mine.NonceInfo) {

	miner := mine.NewCPUMiner(quit)
	nonceCh := make(chan mine.NonceInfo)
	// if miner finished his work before timeThreshold
	// make sure writing to the Stop chan non-blocking.
	stop := make(chan struct{}, 1)
	block := mine.MiningBlock{
		Data:      publicKey.SerializeCompressed(),
		NonceChan: nonceCh,
		Stop:      stop,
	}

	go miner.ComputeBlockNonce(block, mine.Uint256{}, difficulty)

	time.Sleep(timeThreshold)
	// stop miner
	block.Stop <- struct{}{}

	return <-block.NonceChan
}
