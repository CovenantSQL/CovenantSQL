/*
 * Copyright 2018 The ThunderDB Authors.
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
	"io/ioutil"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gopkg.in/yaml.v2"
)

const testFile = "./.configtest"

func TestConf(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	Convey("LoadConfig", t, func() {
		defer os.Remove(testFile)
		config := &Config{
			BP: kms.BP,
			KnownNodes: &[]NodeInfo{
				{
					ID:        kms.BP.NodeID,
					Nonce:     kms.BP.Nonce,
					PublicKey: nil,
					Addr:      "127.0.0.1:2122",
					Role:      kayak.Leader,
				},
				{
					ID:        proto.NodeID("000000000013fd4b3180dd424d5a895bc57b798e5315087b7198c926d8893f98"),
					PublicKey: nil,
					Nonce:     cpuminer.Uint256{789554103, 0, 0, 8070450536379825883},
					Addr:      "127.0.0.1:2121",
					Role:      kayak.Follower,
				},
				{
					ID:        proto.NodeID("0000000000293f7216362791b6b1c9772184d6976cb34310c42547735410186c"),
					PublicKey: nil,
					Nonce:     cpuminer.Uint256{746598970, 0, 0, 10808639108098016056},
					Addr:      "127.0.0.1:2120",
					Role:      kayak.Follower,
				},
				{
					// {{22403 0 0 0} 20 00000f3b43288fe99831eb533ab77ec455d13e11fc38ec35a42d4edd17aa320d}
					ID:        proto.NodeID("00000f3b43288fe99831eb533ab77ec455d13e11fc38ec35a42d4edd17aa320d"),
					Nonce:     cpuminer.Uint256{22403, 0, 0, 0},
					PublicKey: nil,
					Addr:      "",
					Role:      kayak.Client,
				},
			},
		}
		sConfig, _ := yaml.Marshal(config)
		log.Debugf("config:\n%s", sConfig)
		ioutil.WriteFile(testFile, sConfig, 0600)
		configNew, err := LoadConfig(testFile)
		So(err, ShouldBeNil)
		So(configNew.BP.NodeID, ShouldResemble, config.BP.NodeID)
		So(configNew.BP.Nonce, ShouldResemble, config.BP.Nonce)
		So(configNew.BP.PublicKeyStr, ShouldResemble, config.BP.PublicKeyStr)

		configNew, err = LoadConfig("notExistFile")
		So(err, ShouldNotBeNil)

		ioutil.WriteFile(testFile, []byte("xx:1"), 0600)
		_, err = LoadConfig(testFile)
		So(err, ShouldNotBeNil)
	})
}
