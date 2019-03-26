/*
 * Copyright 2018-2019 The CovenantSQL Authors.
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

package internal

import (
	"encoding/hex"
	"fmt"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
)

// CmdGenerate is cql generate command entity.
var CmdGenerate = &Command{
	UsageLine: "cql generate [-config file] config/private/addr/public/nonce",
	Short:     "generate config related file or keys",
	Long: `
Generate command can generate private.key and config.yaml for CovenantSQL.
e.g.
    cql generate config
`,
}

func init() {
	CmdGenerate.Run = runGenerate

	addCommonFlags(CmdGenerate)
}

func runGenerate(cmd *Command, args []string) {
	if len(args) != 1 {
		ConsoleLog.Error("Generate command need specific type as params")
		SetExitStatus(1)
		return
	}
	genType := args[0]

	switch genType {
	case "config":
	case "private":
	case "addr":
		addrGen()
	case "public":
		configInit()
	case "nonce":
		configInit()
	default:
		cmd.Usage()
		SetExitStatus(1)
		return
	}
}

func configGen() {
}

func privateGen() {
}

func addrGen() {
	configInit()

	var publicKey *asymmetric.PublicKey

	//TODO if config has addr, print

	//TODO if config has public, use it
	publicKeyHex := ""
	if publicKeyHex != "" {
		publicKeyBytes, err := hex.DecodeString(publicKeyHex)
		if err != nil {
			ConsoleLog.WithError(err).Error("error converting hex")
			SetExitStatus(1)
			return
		}
		publicKey, err = asymmetric.ParsePubKey(publicKeyBytes)
		if err != nil {
			ConsoleLog.WithError(err).Error("error converting public key")
			SetExitStatus(1)
			return
		}
	} else {
		//use config specific private key file(already init by configInit())
		privateKey, err := kms.LoadPrivateKey(conf.GConf.PrivateKeyFile, []byte(password))
		if err != nil {
			ConsoleLog.WithError(err).Fatal("load private key file failed")
			SetExitStatus(1)
			return
		}
		publicKey = privateKey.PubKey()
	}

	keyHash, err := crypto.PubKeyHash(publicKey)
	if err != nil {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}

	fmt.Printf("wallet address: %s\n", keyHash.String())

	//TODO store in config.yaml
}

func publicGen() {
}

func nonceGen() {
}
