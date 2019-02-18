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

package route

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
)

const PubKeyStorePath = "./acl.keystore"

func TestIsPermitted(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	os.Remove(PubKeyStorePath)
	defer os.Remove(PubKeyStorePath)

	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_0/config.yaml")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %#v", conf.GConf)
	// reset the once
	Once = sync.Once{}
	InitKMS(PubKeyStorePath)

	Convey("test IsPermitted", t, func() {
		nodeID := proto.NodeID("0000")
		testEnv := &proto.Envelope{NodeID: nodeID.ToRawNodeID()}
		testAnonymous := &proto.Envelope{NodeID: kms.AnonymousRawNodeID}
		So(IsPermitted(&proto.Envelope{NodeID: &conf.GConf.BP.RawNodeID}, KayakCall), ShouldBeTrue)
		So(IsPermitted(testEnv, KayakCall), ShouldBeFalse)
		So(IsPermitted(testEnv, DHTFindNode), ShouldBeTrue)
		So(IsPermitted(testEnv, RemoteFunc(9999)), ShouldBeFalse)
		So(IsPermitted(testAnonymous, DHTFindNode), ShouldBeFalse)
	})

	Convey("string RemoteFunc", t, func() {
		for i := DHTPing; i <= MCCQueryTxState; i++ {
			So(fmt.Sprintf("%s", RemoteFunc(i)), ShouldContainSubstring, ".")
		}
		So(fmt.Sprintf("%s", RemoteFunc(9999)), ShouldContainSubstring, "Unknown")
	})

}
