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

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func runKeygen(privateKeyPath string) *asymmetric.PublicKey {

	askDeletePath(privateKeyPath)

	masterKey, err := readMasterKey()
	if err != nil {
		log.WithError(err).Fatal("read master key failed")
	}

	privateKey, _, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		log.WithError(err).Fatal("generate key pair failed")
	}

	if err = kms.SavePrivateKey(privateKeyPath, privateKey, []byte(masterKey)); err != nil {
		log.WithError(err).Fatal("save generated keypair failed")
	}

	fmt.Printf("Private key file: %s\n", privateKeyPath)
	fmt.Printf("Public key's hex: %s\n", hex.EncodeToString(privateKey.PubKey().Serialize()))
	return privateKey.PubKey()
}
