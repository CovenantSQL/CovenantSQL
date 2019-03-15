/*
 * Copyright 2016-2018 Kenneth Shaw.
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
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/CovenantSQL/CovenantSQL/cmd/cql/internal"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	version = "unknown"

//	// Shard chain explorer/adapter stuff
//	tmpPath      string // background observer and explorer block and log file path
//	bgLogLevel   string // background log level
//	explorerAddr string // explorer Web addr
//	adapterAddr  string // adapter listen addr
//
//	// DML variables
//	updatePermission        string // update user's permission on specific sqlchain
//	transferToken           string // transfer token to target account
//
//	service    *observer.Service
//	httpServer *http.Server
)

type userPermission struct {
	TargetChain proto.AccountAddress `json:"chain"`
	TargetUser  proto.AccountAddress `json:"user"`
	Perm        json.RawMessage      `json:"perm"`
}

type userPermPayload struct {
	// User role to access database.
	Role types.UserPermissionRole `json:"role"`
	// SQL pattern regulations for user queries
	// only a fully matched (case-sensitive) sql query is permitted to execute.
	Patterns []string `json:"patterns"`
}

type tranToken struct {
	TargetUser proto.AccountAddress `json:"addr"`
	Amount     string               `json:"amount"`
}

func init() {
	// // Explorer/Adapter
	// flag.StringVar(&tmpPath, "tmp-path", "", "Background service temp file path, use os.TempDir for default")
	// flag.StringVar(&bgLogLevel, "bg-log-level", "", "Background service log level") // flag.StringVar(&explorerAddr, "web", "", "Address to serve a database chain explorer, e.g. :8546")
	// flag.StringVar(&adapterAddr, "adapter", "", "Address to serve a database chain adapter, e.g. :7784")

	// // DML flags
	// flag.StringVar(&updatePermission, "update-perm", "", "Update user's permission on specific sqlchain")
	// flag.StringVar(&transferToken, "transfer", "", "Transfer token to target account")

	internal.CqlCommands = []*internal.Command{
		internal.CmdConsole,
		internal.CmdVersion,
		internal.CmdBalance,
		internal.CmdCreate,
		internal.CmdDrop,
	}
}

func main() {

	internal.Version = version

	// set random
	rand.Seed(time.Now().UnixNano())

	flag.Usage = mainUsage
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		mainUsage()
	}

	internal.CmdName = args[0] // for error messages
	if args[0] == "help" {
		mainUsage()
		return
	}

	internal.ConsoleLog = logrus.New()

	internal.ConsoleLog.Infof("cql build: %#v\n", internal.Version)
	//
	//	if tmpPath == "" {
	//		tmpPath = os.TempDir()
	//	}
	//	logPath := filepath.Join(tmpPath, "covenant_service.log")
	//	bgLog, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	//	if err != nil {
	//		fmt.Fprintf(os.Stderr, "open log file failed: %s, %v", logPath, err)
	//		os.Exit(-1)
	//	}
	//	log.SetOutput(bgLog)
	//	log.SetStringLevel(bgLogLevel, log.InfoLevel)
	//
	//	if explorerAddr != "" {
	//		service, httpServer, err = observer.StartObserver(explorerAddr, internal.Version)
	//		if err != nil {
	//			log.WithError(err).Fatal("start explorer failed")
	//		} else {
	//			internal.ConsoleLog.Infof("explorer started on %s", explorerAddr)
	//		}
	//
	//		defer func() {
	//			_ = observer.StopObserver(service, httpServer)
	//			log.Info("explorer stopped")
	//		}()
	//	}
	//
	//	if adapterAddr != "" {
	//		server, err := adapter.NewHTTPAdapter(adapterAddr, configFile)
	//		if err != nil {
	//			log.WithError(err).Fatal("init adapter failed")
	//		}
	//
	//		if err = server.Serve(); err != nil {
	//			log.WithError(err).Fatal("start adapter failed")
	//		} else {
	//			internal.ConsoleLog.Infof("adapter started on %s", adapterAddr)
	//
	//			defer func() {
	//				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	//				defer cancel()
	//				server.Shutdown(ctx)
	//				log.Info("stopped adapter")
	//			}()
	//		}
	//	}
	//
	//
	//	if updatePermission != "" {
	//		// update user's permission on sqlchain
	//		var perm userPermission
	//		if err := json.Unmarshal([]byte(updatePermission), &perm); err != nil {
	//			internal.ConsoleLog.WithError(err).Errorf("update permission failed: invalid permission description")
	//			os.Exit(-1)
	//			return
	//		}
	//
	//		var permPayload userPermPayload
	//
	//		if err := json.Unmarshal(perm.Perm, &permPayload); err != nil {
	//			// try again using role string representation
	//			if err := json.Unmarshal(perm.Perm, &permPayload.Role); err != nil {
	//				internal.ConsoleLog.WithError(err).Errorf("update permission failed: invalid permission description")
	//				os.Exit(-1)
	//				return
	//			}
	//		}
	//
	//		p := &types.UserPermission{
	//			Role:     permPayload.Role,
	//			Patterns: permPayload.Patterns,
	//		}
	//
	//		if !p.IsValid() {
	//			internal.ConsoleLog.Errorf("update permission failed: invalid permission description")
	//			os.Exit(-1)
	//			return
	//		}
	//
	//		txHash, err := client.UpdatePermission(perm.TargetUser, perm.TargetChain, p)
	//		if err != nil {
	//			internal.ConsoleLog.WithError(err).Error("update permission failed")
	//			os.Exit(-1)
	//			return
	//		}
	//
	//		if waitTxConfirmation {
	//			err = wait(txHash)
	//			if err != nil {
	//				os.Exit(-1)
	//				return
	//			}
	//		}
	//
	//		internal.ConsoleLog.Info("succeed in sending transaction to CovenantSQL")
	//		return
	//	}
	//
	//	if transferToken != "" {
	//		// transfer token
	//		var tran tranToken
	//		if err := json.Unmarshal([]byte(transferToken), &tran); err != nil {
	//			internal.ConsoleLog.WithError(err).Errorf("transfer token failed: invalid transfer description")
	//			os.Exit(-1)
	//			return
	//		}
	//
	//		var validAmount = regexp.MustCompile(`^([0-9]+) *([a-zA-Z]+)$`)
	//		if !validAmount.MatchString(tran.Amount) {
	//			internal.ConsoleLog.Error("transfer token failed: invalid transfer description")
	//			os.Exit(-1)
	//			return
	//		}
	//		amountUnit := validAmount.FindStringSubmatch(tran.Amount)
	//		if len(amountUnit) != 3 {
	//			internal.ConsoleLog.Error("transfer token failed: invalid transfer description")
	//			for _, v := range amountUnit {
	//				internal.ConsoleLog.Error(v)
	//			}
	//			os.Exit(-1)
	//			return
	//		}
	//		amount, err := strconv.ParseUint(amountUnit[1], 10, 64)
	//		if err != nil {
	//			internal.ConsoleLog.Error("transfer token failed: invalid token amount")
	//			os.Exit(-1)
	//			return
	//		}
	//		unit := types.FromString(amountUnit[2])
	//		if !unit.Listed() {
	//			internal.ConsoleLog.Error("transfer token failed: invalid token type")
	//			os.Exit(-1)
	//			return
	//		}
	//
	//		var txHash hash.Hash
	//		txHash, err = client.TransferToken(tran.TargetUser, amount, unit)
	//		if err != nil {
	//			internal.ConsoleLog.WithError(err).Error("transfer token failed")
	//			os.Exit(-1)
	//			return
	//		}
	//
	//		if waitTxConfirmation {
	//			err = wait(txHash)
	//			if err != nil {
	//				os.Exit(-1)
	//				return
	//			}
	//		}
	//
	//		internal.ConsoleLog.Info("succeed in sending transaction to CovenantSQL")
	//		return
	//	}

	for _, cmd := range internal.CqlCommands {
		if cmd.Name() != args[0] {
			continue
		}
		if !cmd.Runnable() {
			continue
		}
		cmd.Flag.Usage = func() { cmd.Usage() }
		cmd.Flag.Parse(args[1:])
		args = cmd.Flag.Args()
		cmd.Run(cmd, args)
		internal.Exit()
		return
	}
	helpArg := ""
	if i := strings.LastIndex(internal.CmdName, " "); i >= 0 {
		helpArg = " " + internal.CmdName[:i]
	}
	fmt.Fprintf(os.Stderr, "cql %s: unknown command\nRun 'cql help%s' for usage.\n", internal.CmdName, helpArg)
	internal.SetExitStatus(2)
	internal.Exit()

	// if web flag is enabled
	//if explorerAddr != "" || adapterAddr != "" {
	//	fmt.Printf("Ctrl + C to stop explorer on %s and adapter on %s\n", explorerAddr, adapterAddr)
	//	<-utils.WaitForExit()
	//	return
	//}
}

func mainUsage() {
	//TODO(laodouya) print stderr main usage
	os.Exit(2)
}
