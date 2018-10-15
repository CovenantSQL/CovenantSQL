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
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	workingRoot string
)

func init() {
	flag.StringVar(&workingRoot, "root", "conf", "confgen root is the working root directory containing all auto-generating keys and certifications")
}

func runConfgen() {
	if workingRoot == "" {
		log.Error("root directory is required for confgen")
		os.Exit(1)
	}

	privateKeyFileName := "private.key"
	publicKeystoreFileName := "public.keystore"

	privateKeyFile = path.Join(workingRoot, privateKeyFileName)

	if _, err := os.Stat(workingRoot); err == nil {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("The directory has already existed. \nDo you want to delete it? (y or n, press Enter for default n):")
		t, err := reader.ReadString('\n')
		t = strings.Trim(t, "\n")
		if err != nil {
			log.Errorf("Unexpected error: %v\n", err)
			os.Exit(1)
		}
		if strings.Compare(t, "y") == 0 || strings.Compare(t, "yes") == 0 {
			os.RemoveAll(workingRoot)
		} else {
			os.Exit(0)
		}
	}

	err := os.Mkdir(workingRoot, 0755)
	if err != nil {
		log.Errorf("Unexpected error: %v", err)
		os.Exit(1)
	}

	fmt.Println("Generating key pair...")
	publicKey := runKeygen()
	fmt.Println("Generated key pair.")

	fmt.Println("Generating nonce...")
	nonce := noncegen(publicKey)
	fmt.Println("Generated nonce.")

	fmt.Println("Generating config file...")

	configContent := fmt.Sprintf(`IsTestMode: true
WorkingRoot: "./"
PrivateKeyFile: "%s"
PubKeyStoreFile: "%s"
DHTFileName: "dht.db"
ListenAddr: "0.0.0.0:4661"
ThisNodeID: %s
MinNodeIDDifficulty: 24
BlockProducer:
  PublicKey: 034b4319f2e2a9d9f3fd55d1233ff7a2f2ea2e815e7227b3861b4a6a24a8d62697
  NodeID: 0000011839f464418166658ef6dec09ea68da1619a7a9e0f247f16e0d6c6504d
  Nonce:
    a: 761802
    b: 0
    c: 0
    d: 4611686019290328603
  ChainFileName: "chain.db"
  BPGenesisInfo:
    Version: 1
    BlockHash: f745ca6427237aac858dd3c7f2df8e6f3c18d0f1c164e07a1c6b8eebeba6b154
    Producer: 0000000000000000000000000000000000000000000000000000000000000001
    MerkleRoot: 0000000000000000000000000000000000000000000000000000000000000001
    ParentHash: 0000000000000000000000000000000000000000000000000000000000000001
    Timestamp: 2018-09-01T00:00:00Z
    BaseAccounts:
      - Address: d3dce44e0a4f1dae79b93f04ce13fb5ab719059f7409d7ca899d4c921da70129
        StableCoinBalance: 100000000
        CovenantCoinBalance: 100000000
KnownNodes:
- ID: 0000011839f464418166658ef6dec09ea68da1619a7a9e0f247f16e0d6c6504d
  Nonce:
    a: 761802
    b: 0
    c: 0
    d: 4611686019290328603
  Addr: 120.79.254.36:11105
  PublicKey: 034b4319f2e2a9d9f3fd55d1233ff7a2f2ea2e815e7227b3861b4a6a24a8d62697
  Role: Leader
- ID: 00000177647ade3bd86a085510113ccae4b8e690424bb99b95b3545039ae8e8c
  Nonce:
    a: 197619
    b: 0
    c: 0
    d: 4611686019249700888
  Addr: 120.79.254.36:11106
  PublicKey: 02d6f3afcd26aa8de25f5d088c5f8d6b052b4ad1b27ce5b84939bc9f105556844e
  Role: Miner
- ID: 000004b0267f959e645b0df5cd38ae0652c1160b960cdcb97b322caafe627e4f
  Nonce:
    a: 455820
    b: 0
    c: 0
    d: 3627017019
  Addr: 120.79.254.36:11107
  PublicKey: 034b4319f2e2a9d9f3fd55d1233ff7a2f2ea2e815e7227b3861b4a6a24a8d62697
  Role: Follower
- ID: 00000328ef30233890f61d7504b640b45e8ba33d5671157a0cee81745e46b963
  Nonce:
    a: 333847
    b: 0
    c: 0
    d: 6917529031239958890
  Addr: 120.79.254.36:11108
  PublicKey: 0202361b87a087cd61137ba3b5bd83c48c180566c8d7f1a0b386c3277bf0dc6ebd
  Role: Miner
- ID: %s
  Nonce:
    a: %d
    b: %d
    c: %d
    d: %d
  Addr: 127.0.0.1:11109
  PublicKey: %s
  Role: Client
`, privateKeyFileName, publicKeystoreFileName,
		nonce.Hash.String(), nonce.Hash.String(),
		nonce.Nonce.A, nonce.Nonce.B, nonce.Nonce.C, nonce.Nonce.D, hex.EncodeToString(publicKey.Serialize()))

	err = ioutil.WriteFile(path.Join(workingRoot, "config.yaml"), []byte(configContent), 0755)
	if err != nil {
		log.Errorf("Unexpected error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated nonce.")
}
