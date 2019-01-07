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

package client

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInit(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	// test init
	Convey("test init", t, func() {
		var stopTestService func()
		var confDir string
		var err error
		stopTestService, confDir, err = startTestService()
		So(err, ShouldBeNil)
		defer stopTestService()
		// already init ed
		err = Init(filepath.Join(confDir, "config.yaml"), []byte(""))
		So(err, ShouldNotBeNil)
		// fake driver not initialized
		atomic.StoreUint32(&driverInitialized, 0)
		err = Init(filepath.Join(confDir, "config.yaml"), []byte(""))
		So(err, ShouldBeNil)
		// test loaded block producer nodes
		bps := route.GetBPs()
		So(len(bps), ShouldBeGreaterThanOrEqualTo, 1)
		//So(bps[0].ToRawNodeID().ToNodeID(), ShouldResemble, (*conf.GConf.KnownNodes)[0].ID)
		stopPeersUpdater()
	})
}

func TestCreate(t *testing.T) {
	Convey("test create", t, func() {
		var stopTestService func()
		var err error
		stopTestService, _, err = startTestService()
		So(err, ShouldBeNil)
		defer stopTestService()
		var dsn string
		dsn, err = Create(ResourceMeta{})
		So(err, ShouldBeNil)
		dsnCfg, err := ParseDSN(dsn)
		So(err, ShouldBeNil)

		waitCtx, cancelWait := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancelWait()
		// should not use client.WaitDBCreation, sql.Open is not supported in this test case
		err = bp.WaitDatabaseCreation(waitCtx, proto.DatabaseID(dsnCfg.DatabaseID), nil, 3*time.Second)
		So(err, ShouldResemble, context.DeadlineExceeded)

		// Calculate database ID
		var priv *asymmetric.PrivateKey
		priv, err = kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)
		var addr proto.AccountAddress
		addr, err = crypto.PubKeyHash(priv.PubKey())
		So(err, ShouldBeNil)
		var dbID = string(proto.FromAccountAndNonce(addr, uint32(stubNextNonce)))

		recoveredCfg, err := ParseDSN(dsn)
		So(err, ShouldBeNil)
		So(recoveredCfg, ShouldResemble, &Config{
			DatabaseID: dbID,
			UseLeader:  true,
		})

		waitCtx2, cancelWait2 := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancelWait2()
		// should not use client.WaitDBCreation, sql.Open is not supported in this test case
		err = bp.WaitDatabaseCreation(waitCtx2, proto.DatabaseID(dsnCfg.DatabaseID), nil, 3*time.Second)
		So(err, ShouldBeNil)
	})
}

func TestDrop(t *testing.T) {
	Convey("test drop", t, func() {
		var stopTestService func()
		var err error
		stopTestService, _, err = startTestService()
		So(err, ShouldBeNil)
		defer stopTestService()
		err = Drop("covenantsql://db")
		So(err, ShouldBeNil)
	})
}

func TestGetTokenBalance(t *testing.T) {
	Convey("test get token balance", t, func() {
		var stopTestService func()
		var err error
		stopTestService, _, err = startTestService()
		So(err, ShouldBeNil)
		defer stopTestService()

		var balance uint64
		balance, err = GetTokenBalance(types.Particle)

		So(err, ShouldBeNil)
		So(balance, ShouldEqual, 0)

		balance, err = GetTokenBalance(-1)

		So(err, ShouldEqual, ErrNoSuchTokenBalance)
	})
}
