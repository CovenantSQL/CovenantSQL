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

package conf

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	yaml "gopkg.in/yaml.v2"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const testFile = "./.configtest"

func TestConf(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	var BPPubkey = new(asymmetric.PublicKey)
	pubKeyBytes, err := hex.DecodeString("02c1db96f2ba7e1cb4e9822d12de0f63fb666feb828c7f509e81fab9bd7a34039c")
	if err != nil {
		t.Errorf("decode pubkey failed: %v", err)
	}
	err = BPPubkey.UnmarshalBinary(pubKeyBytes)
	if err != nil {
		t.Errorf("unmarshal pubkey failed: %v", err)
	}
	log.Debugf("pubkey: %x", BPPubkey.Serialize())
	rawBytes := []byte("Space Cowboy")
	h := hash.THashH(rawBytes)

	log.SetLevel(log.DebugLevel)
	BP := &BPInfo{
		PublicKey: BPPubkey,
		NodeID:    "00000000000589366268c274fdc11ec8bdb17e668d2f619555a2e9c1a29c91d8",
		RawNodeID: proto.RawNodeID{},
		Nonce: cpuminer.Uint256{
			A: 14396347928,
			B: 0,
			C: 0,
			D: 6148914694092305796,
		},
		ChainFileName: "",
		BPGenesis: BPGenesisInfo{
			Version:   1,
			Timestamp: time.Now().UTC(),
		},
	}
	Convey("LoadConfig", t, func() {
		defer os.Remove(testFile)
		config := &Config{
			UseTestMasterKey: false,
			GenerateKeyPair:  false,
			WorkingRoot:      "",
			PubKeyStoreFile:  "",
			PrivateKeyFile:   "",
			DHTFileName:      "",
			ListenAddr:       "",
			ThisNodeID:       "",
			ValidDNSKeys: map[string]string{
				// Cloudflare.com DNSKEY. SEE: `dig +multi cloudflare.com DNSKEY`
				"koPbw9wmYZ7ggcjnQ6ayHyhHaDNMYELKTqT+qRGrZpWSccr/lBcrm10Z1PuQHB3Azhii+sb0PYFkH1ruxLhe5g==": "cloudflare.com",
				"mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==": "cloudflare.com",
			},
			MinNodeIDDifficulty: 2,
			DNSSeed: DNSSeed{
				EnforcedDNSSEC: true,
				DNSServers: []string{
					"1.1.1.1",
					"202.46.34.74",
					"202.46.34.75",
					"202.46.34.76",
				},
			},
			BP: BP,
			KnownNodes: []proto.Node{
				{
					ID:        BP.NodeID,
					Nonce:     BP.Nonce,
					PublicKey: nil,
					Addr:      "127.0.0.1:2122",
					Role:      proto.Leader,
				},
				{
					ID:        proto.NodeID("000000000013fd4b3180dd424d5a895bc57b798e5315087b7198c926d8893f98"),
					PublicKey: nil,
					Nonce: cpuminer.Uint256{
						A: 789554103,
						B: 0,
						C: 0,
						D: 8070450536379825883,
					},
					Addr: "127.0.0.1:2121",
					Role: proto.Follower,
				},
				{
					ID:        proto.NodeID("0000000000293f7216362791b6b1c9772184d6976cb34310c42547735410186c"),
					PublicKey: nil,
					Nonce: cpuminer.Uint256{
						A: 746598970,
						B: 0,
						C: 0,
						D: 10808639108098016056,
					},
					Addr: "127.0.0.1:2120",
					Role: proto.Follower,
				},
				{
					// {{22403 0 0 0} 20 00000f3b43288fe99831eb533ab77ec455d13e11fc38ec35a42d4edd17aa320d}
					ID: proto.NodeID("00000f3b43288fe99831eb533ab77ec455d13e11fc38ec35a42d4edd17aa320d"),
					Nonce: cpuminer.Uint256{
						A: 22403,
						B: 0,
						C: 0,
						D: 0,
					},
					PublicKey: nil,
					Addr:      "",
					Role:      proto.Client,
				},
			},
		}
		sConfig, _ := yaml.Marshal(config)
		hConfig, _ := yaml.Marshal(h)
		log.Debugf("hash: \n %s", hConfig)
		log.Debugf("config:\n%s", sConfig)
		ioutil.WriteFile(testFile, sConfig, 0600)
		configNew, err := LoadConfig(testFile)
		So(err, ShouldBeNil)
		So(configNew.BP.NodeID, ShouldResemble, config.BP.NodeID)
		So(configNew.BP.Nonce, ShouldResemble, config.BP.Nonce)
		So(configNew.BP.PublicKey.Serialize(), ShouldResemble, config.BP.PublicKey.Serialize())

		configNew, err = LoadConfig("notExistFile")
		So(err, ShouldNotBeNil)

		ioutil.WriteFile(testFile, []byte("xx:1"), 0600)
		_, err = LoadConfig(testFile)
		So(err, ShouldNotBeNil)
	})
}
