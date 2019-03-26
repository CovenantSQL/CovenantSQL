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
	UsageLine: "cql generate [-config file] config/wallet/public/nonce",
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
	case "wallet":
		configInit()
		walletGen()
	case "public":
		configInit()
		publicKey := publicGen()
		fmt.Printf("Public key's hex: %s\n", hex.EncodeToString(publicKey.Serialize()))
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

func walletGen() {
	//TODO if config has wallet, print and return

	var publicKey *asymmetric.PublicKey

	//if config has public, use it
	for _, node := range conf.GConf.KnownNodes {
		if node.ID == conf.GConf.ThisNodeID {
			publicKey = node.PublicKey
			break
		}
	}

	if publicKey != nil {
		ConsoleLog.Infof("use public key in config file: %s", configFile)
	} else {
		//use config specific private key file(already init by configInit())
		ConsoleLog.Infof("generate wallet address directly from private key: %s", conf.GConf.PrivateKeyFile)
		publicKey = publicGen()
		ExitIfErrors()
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

func publicGen() *asymmetric.PublicKey {
	privateKey, err := kms.LoadPrivateKey(conf.GConf.PrivateKeyFile, []byte(password))
	if err != nil {
		ConsoleLog.WithError(err).Error("load private key file failed")
		SetExitStatus(1)
		return nil
	}
	return privateKey.PubKey()
}

func nonceGen() {
}
