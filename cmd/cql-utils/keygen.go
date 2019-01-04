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
	"fmt"
	"os"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func runKeygen() *asymmetric.PublicKey {
	if _, err := os.Stat(privateKeyFile); err == nil {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Private key file \"%s\" already exists. \nDo you want to delete it? (y or n, press Enter for default n):\n",
			privateKeyFile)
		t, err := reader.ReadString('\n')
		t = strings.Trim(t, "\n")
		if err != nil {
			log.WithError(err).Error("unexpected error")
			os.Exit(1)
		}
		if strings.Compare(t, "y") == 0 || strings.Compare(t, "yes") == 0 {
			err = os.Remove(privateKeyFile)
			if err != nil {
				log.WithError(err).Error("unexpected error")
				os.Exit(1)
			}
		} else {
			os.Exit(0)
		}
	}

	privateKey, _, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		log.WithError(err).Fatal("generate key pair failed")
	}

	masterKey, err := readMasterKey()
	if err != nil {
		log.WithError(err).Fatal("read master key failed")
	}

	if err = kms.SavePrivateKey(privateKeyFile, privateKey, []byte(masterKey)); err != nil {
		log.WithError(err).Fatal("save generated keypair failed")
	}

	fmt.Printf("Private key file: %s\n", privateKeyFile)
	fmt.Printf("Public key's hex: %s\n", hex.EncodeToString(privateKey.PubKey().Serialize()))
	return privateKey.PubKey()
}
