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
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	mine "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	difficulty int
)

func init() {
	flag.IntVar(&difficulty, "difficulty", 24, "difficulty for miner to mine nodes and generating nonce")
}

func runMiner() {
	masterKey, err := readMasterKey()
	if err != nil {
		fmt.Printf("read master key failed: %v\n", err)
		os.Exit(1)
	}

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
		privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(masterKey))
		if err != nil {
			log.WithError(err).Fatal("load private key file failed")
		}
		publicKey = privateKey.PubKey()
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(
		signalCh,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	signal.Ignore(syscall.SIGHUP, syscall.SIGTTIN, syscall.SIGTTOU)

	cpuCount := runtime.NumCPU()
	log.Infof("cpu: %#v\n", cpuCount)
	nonceChs := make([]chan mine.NonceInfo, cpuCount)
	stopChs := make([]chan struct{}, cpuCount)

	rand.Seed(time.Now().UnixNano())
	step := math.MaxUint64 / uint64(cpuCount)

	for i := 0; i < cpuCount; i++ {
		nonceChs[i] = make(chan mine.NonceInfo)
		stopChs[i] = make(chan struct{})
		go func(i int) {
			miner := mine.NewCPUMiner(stopChs[i])
			nonceCh := nonceChs[i]
			block := mine.MiningBlock{
				Data:      publicKey.Serialize(),
				NonceChan: nonceCh,
				Stop:      nil,
			}
			start := mine.Uint256{D: step*uint64(i) + uint64(rand.Uint32())}
			log.Infof("miner #%#v start: %#v\n", i, start)
			miner.ComputeBlockNonce(block, start, difficulty)
		}(i)
	}

	sig := <-signalCh
	log.Infof("received signal %#v\n", sig)
	for i := 0; i < cpuCount; i++ {
		close(stopChs[i])
	}

	max := mine.NonceInfo{}
	for i := 0; i < cpuCount; i++ {
		newNonce := <-nonceChs[i]
		if max.Difficulty < newNonce.Difficulty {
			max = newNonce
		}
	}

	// verify result
	log.Infof("verify result: %#v\n", kms.IsIDPubNonceValid(&proto.RawNodeID{Hash: max.Hash}, &max.Nonce, publicKey))

	// print result
	fmt.Printf("nonce: %v\n", max)
	fmt.Printf("node id: %v\n", max.Hash.String())
}
