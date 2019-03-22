// +build !testbinary

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
	"context"
	"syscall"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/test"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

func TestCQLD(t *testing.T) {
	Convey("Test cqld 3BPs", t, func() {
		var (
			ctx1, ctx2 context.Context
			ccl1, ccl2 context.CancelFunc
			err        error
		)
		start3BPs()
		defer stopNodes()
		So(len(nodeCmds), ShouldEqual, 3)

		ctx1, ccl1 = context.WithTimeout(context.Background(), 30*time.Second)
		defer ccl1()

		err = utils.WaitToConnect(ctx1, "127.0.0.1", []int{2122, 2121, 2120}, 10*time.Second)
		So(err, ShouldBeNil)

		// Initialize local client
		conf.GConf, err = conf.LoadConfig(FJ(testWorkingDir, "./node_c/config.yaml"))
		So(err, ShouldBeNil)
		route.InitKMS(conf.GConf.PubKeyStoreFile)
		err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, []byte{})
		So(err, ShouldBeNil)

		// Wait BP chain service to be ready
		ctx2, ccl2 = context.WithTimeout(context.Background(), 30*time.Second)
		defer ccl2()
		err = test.WaitBPChainService(ctx2, 3*time.Second)
		So(err, ShouldBeNil)

		// Wait for block producing
		time.Sleep(15 * time.Second)

		// Kill one BP follower
		err = nodeCmds[2].Cmd.Process.Signal(syscall.SIGTERM)
		So(err, ShouldBeNil)
		time.Sleep(15 * time.Second)

		// set current bp to leader bp
		for _, n := range conf.GConf.KnownNodes {
			if n.Role == proto.Leader {
				rpc.SetCurrentBP(n.ID)
				break
			}
		}

		// The other peers should be waiting
		var (
			req  = &types.FetchLastIrreversibleBlockReq{}
			resp = &types.FetchLastIrreversibleBlockResp{}

			lastBlockCount uint32
		)
		err = rpc.RequestBP(route.MCCFetchLastIrreversibleBlock.String(), req, resp)
		So(err, ShouldBeNil)
		lastBlockCount = resp.Count
		time.Sleep(15 * time.Second)
		err = rpc.RequestBP(route.MCCFetchLastIrreversibleBlock.String(), req, resp)
		So(err, ShouldBeNil)
		So(resp.Count, ShouldEqual, lastBlockCount)
	})
}
