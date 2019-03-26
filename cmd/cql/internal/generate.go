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
	"os"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/crypto"
)

// CmdGenerate is cql generate command entity.
var CmdGenerate = &Command{
	UsageLine: "cql generate [-config file] config/wallet/public",
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

func askDeletePath(path string) {
	if _, err := os.Stat(path); err == nil {
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
		if strings.Compare(t, "y") == 0 || strings.Compare(t, "yes") == 0 {
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
	case "wallet":
		configInit()
		walletGen()
	case "public":
		configInit()
		publicKey := getPublic()
		fmt.Printf("Public key's hex: %s\n", hex.EncodeToString(publicKey.Serialize()))
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

	publicKey := getPublic()

	keyHash, err := crypto.PubKeyHash(publicKey)
	if err != nil {
		ConsoleLog.WithError(err).Error("unexpected error")
		SetExitStatus(1)
		return
	}

	fmt.Printf("wallet address: %s\n", keyHash.String())

	//TODO store in config.yaml
}
