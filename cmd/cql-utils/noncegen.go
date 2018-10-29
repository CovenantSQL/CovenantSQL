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

package main

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	mine "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func runNonce() {
	var publicKey *asymmetric.PublicKey

	if publicKeyHex != "" {
		publicKeyBytes, err := hex.DecodeString(publicKeyHex)
		if err != nil {
			log.WithError(err).Fatal("error converting hex")
		}
		publicKey, err = asymmetric.ParsePubKey(publicKeyBytes)
		if err != nil {
			log.WithError(err).Fatal("error converting public key")
		}
	} else if privateKeyFile != "" {
		masterKey, err := readMasterKey()
		if err != nil {
			log.WithError(err).Error("read master key failed")
			os.Exit(1)
		}
		privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(masterKey))
		if err != nil {
			log.WithError(err).Fatal("load private key file failed")
		}
		publicKey = privateKey.PubKey()
	} else {
		log.Fatalln("can neither convert public key nor load private key")
	}

	noncegen(publicKey)
}

func noncegen(publicKey *asymmetric.PublicKey) *mine.NonceInfo {
	publicKeyBytes := publicKey.Serialize()

	cpuCount := runtime.NumCPU()
	log.Infof("cpu: %#v\n", cpuCount)
	stopCh := make(chan struct{})
	nonceCh := make(chan mine.NonceInfo)

	rand.Seed(time.Now().UnixNano())
	step := 256 / cpuCount
	for i := 0; i < cpuCount; i++ {
		go func(i int) {
			startBit := i * step
			position := startBit / 64
			shift := uint(startBit % 64)
			log.Infof("position: %#v, shift: %#v, i: %#v", position, shift, i)
			var start mine.Uint256
			if position == 0 {
				start = mine.Uint256{A: uint64(1<<shift) + uint64(rand.Uint32())}
			} else if position == 1 {
				start = mine.Uint256{B: uint64(1<<shift) + uint64(rand.Uint32())}
			} else if position == 2 {
				start = mine.Uint256{C: uint64(1<<shift) + uint64(rand.Uint32())}
			} else if position == 3 {
				start = mine.Uint256{D: uint64(1<<shift) + uint64(rand.Uint32())}
			}

			for j := start; ; j.Inc() {
				select {
				case <-stopCh:
					break
				default:
					currentHash := mine.HashBlock(publicKeyBytes, j)
					currentDifficulty := currentHash.Difficulty()
					if currentDifficulty >= difficulty {
						nonce := mine.NonceInfo{
							Nonce:      j,
							Difficulty: currentDifficulty,
							Hash:       currentHash,
						}
						nonceCh <- nonce
					}
				}
			}
		}(i)
	}

	nonce := <-nonceCh
	close(stopCh)

	// verify result
	if !kms.IsIDPubNonceValid(&proto.RawNodeID{Hash: nonce.Hash}, &nonce.Nonce, publicKey) {
		log.WithFields(log.Fields{
			"nonce": nonce,
			"id":    nonce.Hash.String(),
		}).Fatal("invalid nonce")
	}

	// print result
	fmt.Printf("nonce: %v\n", nonce)
	fmt.Printf("node id: %v\n", nonce.Hash.String())

	return &nonce
}
