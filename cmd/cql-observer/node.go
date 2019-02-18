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
	"fmt"
	"syscall"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"golang.org/x/crypto/ssh/terminal"
)

func initNode() (err error) {
	var masterKey []byte
	if !conf.GConf.IsTestMode {
		fmt.Print("Type in Master key to continue:")
		masterKey, err = terminal.ReadPassword(syscall.Stdin)
		if err != nil {
			fmt.Printf("Failed to read Master key: %v", err)
		}
		fmt.Println("")
	}

	if err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, masterKey); err != nil {
		log.WithError(err).Error("init local key pair failed")
		return
	}

	log.Info("init routes")

	// init kms routing
	route.InitKMS(conf.GConf.PubKeyStoreFile)

	return
}
