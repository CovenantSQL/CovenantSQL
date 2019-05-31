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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/conf/testnet"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

// CmdGenerate is cql generate command entity.
var CmdGenerate = &Command{
	UsageLine: "cql generate [common params] [-source template_file] [-miner listen_addr] [-private existing_private_key] [dest_path]",
	Short:     "generate a folder contains config file and private key",
	Long: `
Generate generates private.key and config.yaml for CovenantSQL.
You can input a passphrase for local encrypt your private key file by set -with-password
e.g.
    cql generate

or input a passphrase by

    cql generate -with-password
`,
	Flag:       flag.NewFlagSet("Generate params", flag.ExitOnError),
	CommonFlag: flag.NewFlagSet("Common params", flag.ExitOnError),
	DebugFlag:  flag.NewFlagSet("Debug params", flag.ExitOnError),
}

var (
	privateKeyParam string
	source          string
	minerListenAddr string
)

func init() {
	CmdGenerate.Run = runGenerate
	CmdGenerate.Flag.StringVar(&privateKeyParam, "private", "",
		"Generate config using an existing private key")
	CmdGenerate.Flag.StringVar(&source, "source", "",
		"Generate config using the specified config template")
	CmdGenerate.Flag.StringVar(&minerListenAddr, "miner", "",
		"Generate miner config with specified miner address")

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

	var workingRoot string
	if len(args) == 0 {
		workingRoot = utils.HomeDirExpand("~/.cql")
	} else if args[0] == "" {
		workingRoot = utils.HomeDirExpand("~/.cql")
	} else {
		workingRoot = utils.HomeDirExpand(args[0])
	}

	if workingRoot == "" {
		ConsoleLog.Error("config directory is required for generate config")
		SetExitStatus(1)
		return
	}

	if strings.HasSuffix(workingRoot, "config.yaml") {
		workingRoot = filepath.Dir(workingRoot)
	}

	privateKeyFileName := "private.key"
	privateKeyFile := path.Join(workingRoot, privateKeyFileName)

	var (
		privateKey *asymmetric.PrivateKey
		err        error
	)

	// detect customized private key
	if privateKeyParam != "" {
		var oldPassword string

		if password == "" {
			fmt.Println("Please enter the passphrase of the existing private key")
			oldPassword = readMasterKey(!withPassword)
		} else {
			oldPassword = password
		}

		privateKey, err = kms.LoadPrivateKey(privateKeyParam, []byte(oldPassword))

		if err != nil {
			ConsoleLog.WithError(err).Error("load specified private key failed")
			SetExitStatus(1)
			return
		}
	}

	var port string
	if minerListenAddr != "" {
		minerListenAddrSplit := strings.Split(minerListenAddr, ":")
		if len(minerListenAddrSplit) != 2 {
			ConsoleLog.Error("-miner only accepts listen address in ip:port format. e.g. 127.0.0.1:7458")
			SetExitStatus(1)
			return
		}
		port = minerListenAddrSplit[1]
	}

	var rawConfig *conf.Config

	if source == "" {
		// Load testnet config
		rawConfig = testnet.GetTestNetConfig()
		if minerListenAddr != "" {
			testnet.SetMinerConfig(rawConfig)
			rawConfig.ListenAddr = "0.0.0.0:" + port
		}
	} else {
		// Load from template file
		sourceConfig, err := ioutil.ReadFile(source)
		if err != nil {
			ConsoleLog.WithError(err).Error("read config template failed")
			SetExitStatus(1)
			return
		}
		rawConfig = &conf.Config{}
		if err = yaml.Unmarshal(sourceConfig, rawConfig); err != nil {
			ConsoleLog.WithError(err).Error("load config template failed")
			SetExitStatus(1)
			return
		}
	}

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

	err = os.Mkdir(workingRoot, 0755)
	if err != nil && !os.IsExist(err) {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}

	fmt.Println("Generating private key...")
	if password == "" {
		fmt.Println("Please enter passphrase for new private key")
		password = readMasterKey(!withPassword)
	}

	if privateKeyParam == "" {
		privateKey, _, err = asymmetric.GenSecp256k1KeyPair()
		if err != nil {
			ConsoleLog.WithError(err).Error("generate key pair failed")
			SetExitStatus(1)
			return
		}
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

	// Add client config
	rawConfig.PrivateKeyFile = privateKeyFileName
	rawConfig.WalletAddress = walletAddr
	rawConfig.ThisNodeID = cliNodeID
	if rawConfig.KnownNodes == nil {
		rawConfig.KnownNodes = make([]proto.Node, 0, 1)
	}
	node := proto.Node{
		ID:        cliNodeID,
		Role:      proto.Client,
		Addr:      "0.0.0.0:15151",
		PublicKey: publicKey,
		Nonce:     nonce.Nonce,
	}
	if minerListenAddr != "" {
		node.Role = proto.Miner
		node.Addr = minerListenAddr
	}
	rawConfig.KnownNodes = append(rawConfig.KnownNodes, node)

	// Write config
	out, err := yaml.Marshal(rawConfig)
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

	fmt.Printf("\nWallet address: %s\n", walletAddr)
	fmt.Printf(`
Any further command could costs PTC.
You can get some free PTC from:
	https://testnet.covenantsql.io/wallet/`)
	fmt.Println(walletAddr)

	if password != "" {
		fmt.Println("Your private key had been encrypted by a passphrase, add -with-password in any further command")
	}
}
