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
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	yaml "gopkg.in/yaml.v2"
)

// CmdGenerate is cql generate command entity.
var CmdGenerate = &Command{
	UsageLine: "cql generate [common params] config | public",
	Short:     "generate config related file or keys",
	Long: `
Generate generates private.key and config.yaml for CovenantSQL.
e.g.
    cql generate config
`,
}

func init() {
	CmdGenerate.Run = runGenerate

	addCommonFlags(CmdGenerate)
}

func askDeletePath(path string) {
	if fileinfo, err := os.Stat(path); err == nil {
		if !fileinfo.IsDir() {
			path = filepath.Dir(path)
		}
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("\"%s\" already exists. \nDo you want to delete it? (y or n, press Enter for default n):\n",
			path)
		t, err := reader.ReadString('\n')
		t = strings.Trim(t, "\n")
		if err != nil {
			ConsoleLog.WithError(err).Error("unexpected error")
			SetExitStatus(1)
			Exit()
		}
		if strings.EqualFold(t, "y") || strings.EqualFold(t, "yes") {
			err = os.RemoveAll(path)
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
	if len(args) != 1 {
		ConsoleLog.Error("Generate command need specific type as params")
		SetExitStatus(1)
		return
	}
	genType := args[0]

	switch genType {
	case "config":
		configGen()
	case "public":
		publicKey := getPublicFromConfig()
		fmt.Printf("Public key's hex: %s\n", hex.EncodeToString(publicKey.Serialize()))
	default:
		cmd.Usage()
		SetExitStatus(1)
		return
	}
}

func configGen() {
	workingRoot := utils.HomeDirExpand(configFile)
	if workingRoot == "" {
		ConsoleLog.Error("config directory is required for generate config")
		SetExitStatus(1)
		return
	}

	if strings.HasSuffix(workingRoot, "config.yaml") {
		workingRoot = filepath.Dir(workingRoot)
	}

	askDeletePath(workingRoot)

	privateKeyFileName := "private.key"
	privateKeyFile := path.Join(workingRoot, privateKeyFileName)

	err := os.Mkdir(workingRoot, 0755)
	if err != nil {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}

	fmt.Println("Generating key pair...")

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

	fmt.Printf("Private key file: %s\n", privateKeyFile)
	fmt.Printf("Public key's hex: %s\n", hex.EncodeToString(privateKey.PubKey().Serialize()))
	publicKey := privateKey.PubKey()

	fmt.Println("Generated key pair.")

	fmt.Println("Generating nonce...")
	nonce := nonceGen(publicKey)
	cliNodeID := proto.NodeID(nonce.Hash.String())
	fmt.Println("Generated nonce.")

	fmt.Println("Generating config file...")
	// Load testnet config
	testnetConfig := testnet.GetTestNetConfig()
	// Add client config
	testnetConfig.PrivateKeyFile = privateKeyFileName
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
	err = ioutil.WriteFile(path.Join(workingRoot, "config.yaml"), out, 0644)
	if err != nil {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}
	fmt.Println("Generated config.")

}
