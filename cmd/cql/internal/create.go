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
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	pb "gopkg.in/cheggaaa/pb.v1"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

var meta client.ResourceMeta

// CmdCreate is cql create command entity.
var CmdCreate = &Command{
	UsageLine: "cql create [common params] [-wait-tx-confirm] [db_meta_params]",
	Short:     "create a database",
	Long: `
Create command creates a CovenantSQL database by database meta params. The meta info must include
node count.
e.g.
    cql create -db-node 2

Since CovenantSQL is built on top of blockchains, you may want to wait for the transaction
confirmation before the creation takes effect.
e.g.
    cql create -wait-tx-confirm -db-node 2
`,
	Flag:       flag.NewFlagSet("DB meta params", flag.ExitOnError),
	CommonFlag: flag.NewFlagSet("Common params", flag.ExitOnError),
	DebugFlag:  flag.NewFlagSet("Debug params", flag.ExitOnError),
}

func init() {
	CmdCreate.Run = runCreate

	addCommonFlags(CmdCreate)
	addConfigFlag(CmdCreate)
	addWaitFlag(CmdCreate)
	addCreateFlags(CmdCreate)
}

// List is a list of strings for flag usage.
type List struct {
	Values []string
}

// String is a function in List struct for flag usage.
func (l *List) String() string {
	return strings.Join(l.Values, ",")
}

// Set is a function in List struct for flag usage.
func (l *List) Set(raw string) error {
	l.Values = strings.Split(raw, ",")
	return nil
}

// Get is a function in List struct for flag usage.
func (l *List) Get() interface{} { return List(*l) }

var targetMiners List
var node32 uint

func addCreateFlags(cmd *Command) {
	cmd.Flag.Var(&targetMiners, "db-target-miners", "List of target miner addresses(separated by ',')")
	cmd.Flag.UintVar(&node32, "db-node", 0, "Target node number")
	cmd.Flag.Uint64Var(&meta.Space, "db-space", 0, "Minimum disk space requirement, 0 for none")
	cmd.Flag.Uint64Var(&meta.Memory, "db-memory", 0, "Minimum memory requirement, 0 for none")
	cmd.Flag.Float64Var(&meta.LoadAvgPerCPU, "db-load-avg-per-cpu", 0, "Minimum idle CPU requirement, 0 for none")
	cmd.Flag.StringVar(&meta.EncryptionKey, "db-encrypt-key", "", "Encryption key for persistence data")
	cmd.Flag.BoolVar(&meta.UseEventualConsistency, "db-eventual-consistency", false, "Use eventual consistency to sync among miner nodes")
	cmd.Flag.Float64Var(&meta.ConsistencyLevel, "db-consistency-level", 0, "Consistency level, node*consistency_level is the node number to perform strong consistency")
	cmd.Flag.IntVar(&meta.IsolationLevel, "db-isolation-level", 0, "Isolation level in a single node")
	cmd.Flag.Uint64Var(&meta.GasPrice, "db-gas-price", 0, "Customized gas price")
	cmd.Flag.Uint64Var(&meta.AdvancePayment, "db-advance-payment", 0, "Customized advance payment")
}

func runCreate(cmd *Command, args []string) {
	commonFlagsInit(cmd)

	if len(args) > 1 {
		ConsoleLog.Error("create params should set by specific param name like -node")
		SetExitStatus(1)
		printCommandHelp(cmd)
		Exit()
	}

	for _, miner := range targetMiners.Values {
		targetMiner, err := hash.NewHashFromStr(miner)
		if err != nil {
			ConsoleLog.Error("create target-miners param has invalid node address: ", miner)
			SetExitStatus(1)
			return
		}
		meta.TargetMiners = append(meta.TargetMiners, proto.AccountAddress(*targetMiner))
	}

	if node32 > math.MaxUint16 {
		ConsoleLog.Error("create node param should not greater than uint16")
		SetExitStatus(1)
		return
	}
	meta.Node = uint16(node32)

	if len(args) == 1 && args[0] != "" {
		// fill the meta with params
		if err := json.Unmarshal([]byte(args[0]), &meta); err != nil {
			ConsoleLog.Error("create node json param is not valid")
			SetExitStatus(1)
			return
		}

		// 0.5.0 version forward compatibility
		var tempMeta map[string]json.RawMessage
		if err := json.Unmarshal([]byte(args[0]), &tempMeta); err == nil {
			for k, v := range tempMeta {
				switch strings.ToLower(k) {
				case "targetminers":
					_ = json.Unmarshal(v, &meta.TargetMiners)
				case "loadavgpercpu":
					_ = json.Unmarshal(v, &meta.LoadAvgPerCPU)
				case "encryptionkey":
					_ = json.Unmarshal(v, &meta.EncryptionKey)
				case "useeventualconsistency":
					_ = json.Unmarshal(v, &meta.UseEventualConsistency)
				case "consistencylevel":
					_ = json.Unmarshal(v, &meta.ConsistencyLevel)
				case "isolationlevel":
					_ = json.Unmarshal(v, &meta.IsolationLevel)
				case "gasprice":
					_ = json.Unmarshal(v, &meta.GasPrice)
				case "advancepayment":
					_ = json.Unmarshal(v, &meta.AdvancePayment)
				}
			}
		} else {
			err = nil
		}
	}

	if meta.Node == 0 {
		ConsoleLog.Error("create database failed: request node count must > 1")
		SetExitStatus(1)
		return
	}

	configInit()

	// create database
	// parse instance requirement

	txHash, dsn, err := client.Create(meta)
	if err != nil {
		ConsoleLog.WithError(err).Error("create database failed")
		SetExitStatus(1)
		return
	}

	ConsoleLog.Info("create database requested")

	if waitTxConfirmation {
		cancelLoading := printLoading(int(waitTxConfirmationMaxDuration / time.Second))
		defer cancelLoading()

		wait(txHash)

		var ctx, cancel = context.WithTimeout(context.Background(), waitTxConfirmationMaxDuration)
		defer cancel()
		err = client.WaitDBCreation(ctx, dsn)
		if err != nil {
			ConsoleLog.WithError(err).Error("create database failed durating creation")
			SetExitStatus(1)
			return
		}
	}

	fmt.Printf("\nThe newly created database is: %#v\n", dsn)
	storeOneDSN(dsn)
	fmt.Printf("The connecting string beginning with 'covenantsql://' could be used as a dsn for `cql console`\n or any command, or be used in website like https://web.covenantsql.io\n")
}

func printLoading(loadingMax int) func() {
	var ctxLoading, cancelLoading = context.WithCancel(context.Background())
	go func(ctx context.Context) {
		bar := pb.StartNew(loadingMax)
		bar.Output = os.Stderr
		for {
			select {
			case <-ctx.Done():
				bar.Finish()
				return
			default:
				bar.Increment()
				time.Sleep(time.Second)
			}
		}
	}(ctxLoading)
	return cancelLoading
}
