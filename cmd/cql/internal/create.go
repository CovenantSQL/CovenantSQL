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
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"gopkg.in/cheggaaa/pb.v1"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
)

var meta client.ResourceMeta

// CmdCreate is cql create command entity.
var CmdCreate = &Command{
	UsageLine: "cql create [common params] [-wait-tx-confirm] [db_meta params]",
	Short:     "create a database",
	Long: `
Create creates a CovenantSQL database by database meta info JSON string. The meta info must include
node count.
e.g.
    cql create  -node 2

A complete introduction of db_meta params fields：

    target-miners          []string  // List of target miner addresses
    node                   uint      // Target node number
    space                  uint      // Minimum disk space requirement, 0 for none
    memory                 uint      // Minimum memory requirement, 0 for none
    load-avg-per-cpu       float     // Minimum idle CPU requirement, 0 for none
    encrypt-key            string    // Encryption key for persistence data
    eventual-consistency   bool      // Use eventual consistency to sync among miner nodes
    consistency-level      float     // Consistency level, node*consistency_level is the node number to perform strong consistency
    isolation-level        int       // Isolation level in a single node
    gas-price              uint      // customized gas price
    advance-payment        uint      // customized advance payment

Since CovenantSQL is built on top of blockchains, you may want to wait for the transaction
confirmation before the creation takes effect.
e.g.
    cql create -wait-tx-confirm -node 2
`,
}

func init() {
	CmdCreate.Run = runCreate

	addCommonFlags(CmdCreate)
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
	cmd.Flag.Var(&targetMiners, "target-miners", "List of target miner addresses(seperated by ',')")
	cmd.Flag.UintVar(&node32, "node", 0, "Target node number")
	cmd.Flag.Uint64Var(&meta.Space, "space", 0, "Minimum disk space requirement, 0 for none")
	cmd.Flag.Uint64Var(&meta.Memory, "memory", 0, "Minimum memory requirement, 0 for none")
	cmd.Flag.Float64Var(&meta.LoadAvgPerCPU, "load-avg-per-cpu", 0, "Minimum idle CPU requirement, 0 for none")
	cmd.Flag.StringVar(&meta.EncryptionKey, "encrypt-key", "", "Encryption key for persistence data")
	cmd.Flag.BoolVar(&meta.UseEventualConsistency, "eventual-consistency", false, "Use eventual consistency to sync among miner nodes")
	cmd.Flag.Float64Var(&meta.ConsistencyLevel, "consistency-level", 0, "Consistency level, node*consistency_level is the node number to perform strong consistency")
	cmd.Flag.IntVar(&meta.IsolationLevel, "isolation-level", 0, "Isolation level in a single node")
	cmd.Flag.Uint64Var(&meta.GasPrice, "gas-price", 0, "customized gas price")
	cmd.Flag.Uint64Var(&meta.AdvancePayment, "advance-payment", 0, "customized advance payment")
}

func runCreate(cmd *Command, args []string) {
	for _, miner := range targetMiners.Values {
		targetMiner, err := hash.NewHashFromStr(miner)
		if err != nil {
			ConsoleLog.Error("Create target-miners param has invalid node address: ", miner)
			SetExitStatus(1)
			return
		}
		meta.TargetMiners = append(meta.TargetMiners, proto.AccountAddress(*targetMiner))
	}

	if node32 > math.MaxUint16 {
		ConsoleLog.Error("Create node param should not greater than uint16")
		SetExitStatus(1)
		return
	}
	meta.Node = uint16(node32)

	if meta.Node == 0 {
		ConsoleLog.Error("Create database failed: request node count must > 1")
		SetExitStatus(1)
		help = true
	}

	configInit(cmd)

	// create database
	// parse instance requirement

	txHash, dsn, err := client.Create(meta)
	if err != nil {
		ConsoleLog.WithError(err).Error("create database failed")
		SetExitStatus(1)
		return
	}

	ConsoleLog.Info("Create database requested")

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
