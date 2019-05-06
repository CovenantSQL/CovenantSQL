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
	"bufio"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/conf/testnet"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	yaml "gopkg.in/yaml.v2"
)

// CmdGenerate is cql generate command entity.
var CmdGenerate = &Command{
	UsageLine: "cql generate [common params]",
	Short:     "generate a folder contains config file and private key",
	Long: `
Generate generates private.key and config.yaml for CovenantSQL.
You can input a passphrase for local encrypt your private key file by set -no-password=false
e.g.
    cql generate

or input a passphrase by

    cql generate -no-password=false
`,
}

func init() {
	CmdGenerate.Run = runGenerate

	addCommonFlags(CmdGenerate)
}

func askDeleteFile(file string) {
	if fileinfo, err := os.Stat(file); err == nil {
		if fileinfo.IsDir() {
			return
		}
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("\"%s\" already exists. \nDo you want to delete it? (y or n, press Enter for default n):\n",
			file)
		t, err := reader.ReadString('\n')
		t = strings.Trim(t, "\n")
		if err != nil {
			ConsoleLog.WithError(err).Error("unexpected error")
			SetExitStatus(1)
			Exit()
		}
		if strings.EqualFold(t, "y") || strings.EqualFold(t, "yes") {
			err = os.Remove(file)
			if err != nil {
				ConsoleLog.WithError(err).Error("unexpected error")
				SetExitStatus(1)
				Exit()
			}
		} else {
			Exit()
		}
	}
}

func runGenerate(cmd *Command, args []string) {
	commonFlagsInit(cmd)

	workingRoot := utils.HomeDirExpand(configFile)
	if workingRoot == "" {
		ConsoleLog.Error("config directory is required for generate config")
		SetExitStatus(1)
		return
	}

	if strings.HasSuffix(workingRoot, "config.yaml") {
		workingRoot = filepath.Dir(workingRoot)
	}

	var err error
	var fileinfo os.FileInfo
	if fileinfo, err = os.Stat(workingRoot); err == nil {
		if fileinfo.IsDir() {
			err = filepath.Walk(workingRoot, func(filepath string, f os.FileInfo, err error) error {
				if f == nil {
					return err
				}
				if f.IsDir() {
					return nil
				}
				if strings.Contains(f.Name(), "config.yaml") ||
					strings.Contains(f.Name(), "private.key") ||
					strings.Contains(f.Name(), "public.keystore") ||
					strings.Contains(f.Name(), ".dsn") {
					askDeleteFile(filepath)
				}
				return nil
			})
		} else {
			askDeleteFile(workingRoot)
		}
	}

	if err != nil && !os.IsNotExist(err) {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}

	privateKeyFileName := "private.key"
	privateKeyFile := path.Join(workingRoot, privateKeyFileName)

	err = os.Mkdir(workingRoot, 0755)
	if err != nil && !os.IsExist(err) {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}

	fmt.Println("Generating private key...")
	if password == "" {
		password = readMasterKey(noPassword)
	}

	privateKey, _, err := asymmetric.GenSecp256k1KeyPair()
	if err != nil {
		ConsoleLog.WithError(err).Error("generate key pair failed")
		SetExitStatus(1)
		return
	}

	if err = kms.SavePrivateKey(privateKeyFile, privateKey, []byte(password)); err != nil {
		ConsoleLog.WithError(err).Error("save generated keypair failed")
		SetExitStatus(1)
		return
	}
	fmt.Println("Generated private key.")

	publicKey := privateKey.PubKey()
	keyHash, err := crypto.PubKeyHash(publicKey)
	if err != nil {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}

	walletAddr := keyHash.String()

	fmt.Println("Generating nonce...")
	nonce := nonceGen(publicKey)
	cliNodeID := proto.NodeID(nonce.Hash.String())
	fmt.Println("Generated nonce.")

	fmt.Println("Generating config file...")
	// Load testnet config
	testnetConfig := testnet.GetTestNetConfig()
	// Add client config
	testnetConfig.PrivateKeyFile = privateKeyFileName
	testnetConfig.WalletAddress = walletAddr
	testnetConfig.ThisNodeID = cliNodeID
	if testnetConfig.KnownNodes == nil {
		testnetConfig.KnownNodes = make([]proto.Node, 0, 1)
	}
	testnetConfig.KnownNodes = append(testnetConfig.KnownNodes, proto.Node{
		ID:        cliNodeID,
		Role:      proto.Client,
		Addr:      "0.0.0.0:15151",
		PublicKey: publicKey,
		Nonce:     nonce.Nonce,
	})

	// Write config
	out, err := yaml.Marshal(testnetConfig)
	if err != nil {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}
	configFilePath := path.Join(workingRoot, "config.yaml")
	err = ioutil.WriteFile(configFilePath, out, 0644)
	if err != nil {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}
	fmt.Println("Generated config.")

	fmt.Printf("\nConfig file:      %s\n", configFilePath)
	fmt.Printf("Private key file: %s\n", privateKeyFile)
	fmt.Printf("Public key's hex: %s\n", hex.EncodeToString(publicKey.Serialize()))

	fmt.Printf(`
Any further command could costs PTC.
You can get some free PTC from:
	https://testnet.covenantsql.io/wallet/`)
	fmt.Println(walletAddr)

	if password != "" {
		fmt.Println("Your private key had been encrypted by a passphrase, add -no-password=false in any further command")
	}
}
