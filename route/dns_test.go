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
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func TestResolver(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_c/config.yaml")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %v", conf.GConf)

	Convey("resolver init", t, func() {
		setResolveCache(make(NodeIDAddressMap))
		addr, err := GetNodeAddrCache(&proto.RawNodeID{
			Hash: hash.Hash([32]byte{0xde, 0xad}),
		})
		So(err, ShouldEqual, ErrUnknownNodeID)
		So(addr, ShouldBeBlank)

		addr, err = GetNodeAddrCache(nil)
		So(err, ShouldEqual, ErrNilNodeID)
		So(addr, ShouldBeBlank)

		err = SetNodeAddrCache(nil, addr)
		So(err, ShouldEqual, ErrNilNodeID)

		nodeA := &proto.RawNodeID{
			Hash: hash.Hash([32]byte{0xaa, 0xaa}),
		}
		err = SetNodeAddrCache(nodeA, addr)
		So(err, ShouldBeNil)

		addr, err = GetNodeAddrCache(nodeA)
		So(err, ShouldBeNil)
		So(addr, ShouldEqual, addr)

		So(IsBPNodeID(nil), ShouldBeFalse)

		So(IsBPNodeID(nodeA), ShouldBeFalse)

		BPmap := initBPNodeIDs()
		log.Debugf("BPmap: %v", BPmap)
		BPs := GetBPs()
		dc := NewDNSClient()
		ips, err := dc.GetBPFromDNSSeed(BPDomain)

		log.Debugf("BPs: %v", BPs)
		So(len(BPs), ShouldBeGreaterThanOrEqualTo, len(ips))
	})
}
