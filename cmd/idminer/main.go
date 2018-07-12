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

package main

import (
	"encoding/hex"
	"os"
	"os/signal"
	"syscall"

	"runtime"

	"math"

	"math/rand"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	mine "gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

var (
	version = "unknown"
)

func main() {
	// TODO(auxten): keypair and nodeID generator
	log.Infof("idminer build: %s", version)
	if len(os.Args) != 2 {
		log.Error("usage: ./idminer publicKeyHex")
		os.Exit(1)
	}
	publicKeyString := os.Args[1]
	publicKeyBytes, err := hex.DecodeString(publicKeyString)
	if err != nil {
		log.Fatalf("error converting hex: %s", err)
	}
	publicKey, err := asymmetric.ParsePubKey(publicKeyBytes)
	if err != nil {
		log.Fatalf("error converting public key: %s", err)
	}
	signalCh := make(chan os.Signal, 1)
	signal.Notify(
		signalCh,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	signal.Ignore(syscall.SIGHUP, syscall.SIGTTIN, syscall.SIGTTOU)

	cpuCount := runtime.NumCPU()
	log.Infof("cpu: %d", cpuCount)
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
			start := mine.Uint256{0, 0, 0, step*uint64(i) + uint64(rand.Uint32())}
			log.Infof("miner #%d start: %v", i, start)
			miner.ComputeBlockNonce(block, start, 256)
		}(i)
	}

	sig := <-signalCh
	log.Infof("received signal %s", sig)
	for i := 0; i < cpuCount; i++ {
		stopChs[i] <- struct{}{}
	}

	max := mine.NonceInfo{}
	for i := 0; i < cpuCount; i++ {
		newNonce := <-nonceChs[i]
		if max.Difficulty < newNonce.Difficulty {
			max = newNonce
		}
	}
	log.Infof("nonce: %v", max)
}
