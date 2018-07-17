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
	"flag"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	mine "gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

var (
	version        = "unknown"
	tool           string
	publicKeyHex   string
	privateKeyFile string
	difficulty     int
)

func init() {
	flag.StringVar(&tool, "tool", "miner", "tool type, miner, keygen, keytool")
	flag.StringVar(&publicKeyHex, "public", "", "public key hex string to mine node id/nonce")
	flag.StringVar(&privateKeyFile, "private", "", "private key file to generate/show")
	flag.IntVar(&difficulty, "difficulty", 256, "difficulty for miner to mine nodes")
}

func main() {
	log.Infof("idminer build: %s", version)
	flag.Parse()

	switch tool {
	case "miner":
		if publicKeyHex == "" && privateKeyFile == "" {
			// error
			log.Error("publicKey or privateKey is required in miner mode")
			os.Exit(1)
		}
		runMiner()
	case "keygen":
		if privateKeyFile == "" {
			// error
			log.Error("privateKey path is required for keygen")
			os.Exit(1)
		}
		runKeygen()
	case "keytool":
		if privateKeyFile == "" {
			// error
			log.Error("privateKey path is required for keytool")
			os.Exit(1)
		}
		runKeytool()
	default:
		flag.Usage()
		os.Exit(1)
	}

}

func runMiner() {
	var publicKey *asymmetric.PublicKey

	if publicKeyHex != "" {
		publicKeyBytes, err := hex.DecodeString(publicKeyHex)
		if err != nil {
			log.Fatalf("error converting hex: %s", err)
		}
		publicKey, err = asymmetric.ParsePubKey(publicKeyBytes)
		if err != nil {
			log.Fatalf("error converting public key: %s", err)
		}
	} else if privateKeyFile != "" {
		privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(""))
		if err != nil {
			log.Fatalf("load private key file faile: %v", err)
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
			miner.ComputeBlockNonce(block, start, difficulty)
		}(i)
	}

	sig := <-signalCh
	log.Infof("received signal %s", sig)
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
	log.Infof("nonce: %v", max)
}

func runKeygen() {
	privateKey, _, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		log.Fatalf("generate key pair failed: %v", err)
	}

	if err = kms.SavePrivateKey(privateKeyFile, privateKey, []byte("")); err != nil {
		log.Fatalf("save generated keypair failed: %v", err)
	}

	log.Infof("pubkey hex is: %s", hex.EncodeToString(privateKey.PubKey().Serialize()))
}

func runKeytool() {
	privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(""))
	if err != nil {
		log.Fatalf("load private key failed: %v", err)
	}

	log.Infof("pubkey hex is: %s", hex.EncodeToString(privateKey.PubKey().Serialize()))
}
