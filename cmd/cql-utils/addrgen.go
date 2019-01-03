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
	"os"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func runAddrgen() {
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
			fmt.Printf("read master key failed: %v\n", err)
			os.Exit(1)
		}
		privateKey, err := kms.LoadPrivateKey(privateKeyFile, []byte(masterKey))
		if err != nil {
			log.WithError(err).Fatal("load private key file failed")
		}
		publicKey = privateKey.PubKey()
	} else {
		fmt.Println("privateKey path or publicKey hex is required for addrgen")
		os.Exit(1)
	}

	keyHash, err := crypto.PubKeyHash(publicKey)
	if err != nil {
		log.WithError(err).Fatal("unexpected error")
	}

	fmt.Printf("wallet address: %s\n", keyHash.String())
}
