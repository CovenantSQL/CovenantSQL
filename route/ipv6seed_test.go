/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package route

import (
	"encoding/hex"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const TestDomain = "unittest.optool.net"
const IntergrationTestDomain = "bp00.intergrationtest.gridb.io"

func TestIPv6Seed(t *testing.T) {
	isc := IPv6SeedClient{}
	log.SetLevel(log.DebugLevel)
	Convey("", t, func() {
		var pub asymmetric.PublicKey
		_ = pub.UnmarshalBinary(
			[]byte{2, 151, 232, 88, 201, 127, 111, 128,
				208, 117, 192, 223, 212, 5, 209, 42,
				214, 62, 89, 253, 18, 51, 73, 188,
				178, 136, 185, 2, 158, 56, 217, 104, 154})

		node := proto.Node{
			ID:        proto.NodeID("0000000001f26f2145dc770edc385806c6ef131a472ea9ae0f9073d03b4b96d8"),
			Addr:      "111.111.111.111:11111",
			PublicKey: &pub,
			Nonce:     cpuminer.Uint256{1, 2, 3, 4},
		}

		out, err := isc.GenBPIPv6(&node, TestDomain)
		if err != nil {
			t.Errorf("gen ipv6 failed: %v", err)
			return
		}
		log.Debug(out)

		nodeBuf, err := utils.EncodeMsgPack(node)
		if err != nil {
			t.Errorf("marshal node info failed: %v", err)
			return
		}
		log.Debugf("node: %s", nodeBuf)

		m, err := isc.GetBPFromDNSSeed(TestDomain)
		So(err, ShouldBeNil)
		So(len(m), ShouldEqual, 1)
		So(m[*node.ID.ToRawNodeID()].ID, ShouldResemble, node.ID)
		So(m[*node.ID.ToRawNodeID()].Addr, ShouldResemble, node.Addr)
		So(m[*node.ID.ToRawNodeID()].PublicKey.Serialize(), ShouldResemble, node.PublicKey.Serialize())
		So(m[*node.ID.ToRawNodeID()].Nonce, ShouldResemble, node.Nonce)
	})
	Convey(IntergrationTestDomain, t, func() {
		var pub asymmetric.PublicKey
		pubKeyBytes, _ := hex.DecodeString("02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24")
		_ = pub.UnmarshalBinary(pubKeyBytes)

		node := proto.Node{
			ID:        proto.NodeID("00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9"),
			Addr:      "127.0.0.1:3122",
			PublicKey: &pub,
			Nonce:     cpuminer.Uint256{313283, 0, 0, 0},
		}

		out, err := isc.GenBPIPv6(&node, IntergrationTestDomain)
		if err != nil {
			t.Errorf("gen ipv6 failed: %v", err)
			return
		}
		log.Debug(out)

		m, err := isc.GetBPFromDNSSeed(IntergrationTestDomain)
		So(err, ShouldBeNil)
		So(len(m), ShouldEqual, 1)
		So(m[*node.ID.ToRawNodeID()].ID, ShouldResemble, node.ID)
		So(m[*node.ID.ToRawNodeID()].Addr, ShouldResemble, node.Addr)
		So(m[*node.ID.ToRawNodeID()].PublicKey.Serialize(), ShouldResemble, node.PublicKey.Serialize())
		So(m[*node.ID.ToRawNodeID()].Nonce, ShouldResemble, node.Nonce)
	})

}

func TestGenTestNetDomain(t *testing.T) {
	isc := IPv6SeedClient{}
	Convey("generate testnet domain", t, func() {
		log.SetLevel(log.DebugLevel)
		var (
			baseDir     = utils.GetProjectSrcDir()
			testnetConf = utils.FJ(baseDir, "./conf/testnet/testnet-bp.yaml")
		)

		conf, err := conf.LoadConfig(testnetConf)
		So(err, ShouldBeNil)
		var i int
		for _, node := range conf.KnownNodes {
			if node.Role == proto.Leader || node.Role == proto.Follower {
				if node.Addr[:4] != fmt.Sprintf("bp%02d", i) {
					t.Errorf("BP order in yaml should follow 00,01...")
				}
				bpDomain := fmt.Sprintf("bp%02d.testnet.gridb.io", i)
				i++
				out, err := isc.GenBPIPv6(&node, bpDomain)
				if err != nil {
					t.Errorf("gen ipv6 failed: %v", err)
					return
				}
				fmt.Print(out)

				m, err := isc.GetBPFromDNSSeed(bpDomain)
				So(err, ShouldBeNil)
				So(len(m), ShouldEqual, 1)
				So(m[*node.ID.ToRawNodeID()].ID, ShouldResemble, node.ID)
				So(m[*node.ID.ToRawNodeID()].Addr, ShouldResemble, node.Addr)
				So(m[*node.ID.ToRawNodeID()].PublicKey.Serialize(), ShouldResemble, node.PublicKey.Serialize())
				So(m[*node.ID.ToRawNodeID()].Nonce, ShouldResemble, node.Nonce)
			}
		}
	})
}
