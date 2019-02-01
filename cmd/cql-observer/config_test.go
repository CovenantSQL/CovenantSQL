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
	"io/ioutil"
	"os"
	"path"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestLoadConfig(t *testing.T) {
	// Create
	Convey("Given a working directory", t, func() {
		var (
			tmp, fl string
			err     error
			cfg     *Config
		)
		tmp, err = ioutil.TempDir("", "covenantsql")
		So(err, ShouldBeNil)
		Reset(func() {
			err = os.RemoveAll(tmp)
			So(err, ShouldBeNil)
		})
		fl = path.Join(tmp, t.Name())
		Convey("The LoadConfig func should report error at a nonexist file", func() {
			cfg, err = loadConfig(fl)
			So(err, ShouldNotBeNil)
		})
		Convey("Given a empty config file", func() {
			err = ioutil.WriteFile(fl, nil, 0644)
			So(err, ShouldBeNil)
			Convey("The LoadConfig func should return a nil config", func() {
				cfg, err = loadConfig(fl)
				So(err, ShouldBeNil)
				So(cfg, ShouldBeNil)
			})
		})
		Convey("Given a config file without observer section", func() {
			err = ioutil.WriteFile(fl, []byte(
				`IsTestMode: true
WorkingRoot: "./"
PubKeyStoreFile: "public.keystore"
PrivateKeyFile: "private.key"
DHTFileName: "dht.db"
ListenAddr: "127.0.0.1:2122"
ThisNodeID: "00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9"
ValidDNSKeys:
  koPbw9wmYZ7ggcjnQ6ayHyhHaDNMYELKTqT+qRGrZpWSccr/lBcrm10Z1PuQHB3Azhii+sb0PYFkH1ruxLhe5g==: cloudflare.com
  mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==: cloudflare.com
MinNodeIDDifficulty: 2
DNSSeed:
  EnforcedDNSSEC: false
  DNSServers:
  - 1.1.1.1
  - 202.46.34.74
  - 202.46.34.75
  - 202.46.34.76

BlockProducer:
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  NodeID: 00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9
  Nonce:
    a: 313283
    b: 0
    c: 0
    d: 0
  ChainFileName: "chain.db"
  BPGenesisInfo:
    Version: 1
    BlockHash: f745ca6427237aac858dd3c7f2df8e6f3c18d0f1c164e07a1c6b8eebeba6b154
    Producer: 0000000000000000000000000000000000000000000000000000000000000001
    MerkleRoot: 0000000000000000000000000000000000000000000000000000000000000001
    ParentHash: 0000000000000000000000000000000000000000000000000000000000000001
    Timestamp: "2019-01-10T12:49:07+08:00"
KnownNodes:
- ID: 00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9
  Nonce:
    a: 313283
    b: 0
    c: 0
    d: 0
  Addr: 127.0.0.1:2122
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  Role: Leader
- ID: 00000381d46fd6cf7742d7fb94e2422033af989c0e348b5781b3219599a3af35
  Nonce:
    a: 478373
    b: 0
    c: 0
    d: 2305843009893772025
  Addr: 127.0.0.1:2121
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  Role: Follower
- ID: 000000172580063ded88e010556b0aca2851265be8845b1ef397e8fce6ab5582
  Nonce:
    a: 259939
    b: 0
    c: 0
    d: 2305843012544226372
  Addr: 127.0.0.1:2120
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  Role: Follower
- ID: 00000f3b43288fe99831eb533ab77ec455d13e11fc38ec35a42d4edd17aa320d
  Nonce:
    a: 22403
    b: 0
    c: 0
    d: 0
  Addr: ""
  PublicKey: 02ec784ca599f21ef93fe7abdc68d78817ab6c9b31f2324d15ea174d9da498b4c4
  Role: Client
- ID: 000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade
  Nonce:
    a: 567323
    b: 0
    c: 0
    d: 3104982049
  Addr: 127.0.0.1:2144
  PublicKey: 0367aa51809a7c1dc0f82c02452fec9557b3e1d10ce7c919d8e73d90048df86d20
  Role: Miner
- ID: 000005f4f22c06f76c43c4f48d5a7ec1309cc94030cbf9ebae814172884ac8b5
  Nonce:
    a: 240524
    b: 0
    c: 0
    d: 2305843010430351476
  Addr: 127.0.0.1:2145
  PublicKey: 02914bca0806f040dd842207c44474ab41ecd29deee7f2d355789c5c78d448ca16
  Role: Miner
- ID: 000003f49592f83d0473bddb70d543f1096b4ffed5e5f942a3117e256b7052b8
  Nonce:
    a: 606016
    b: 0
    c: 0
    d: 13835058056920509601
  Addr: 127.0.0.1:2146
  PublicKey: 03ae859eac5b72ee428c7a85f10b2ce748d9de5e480aefbb70f6597dfa8b2175e5
  Role: Miner`), 0644)
			So(err, ShouldBeNil)
			Convey("The LoadConfig func should return a nil config", func() {
				cfg, err = loadConfig(fl)
				So(err, ShouldBeNil)
				So(cfg, ShouldBeNil)
			})
		})
		Convey("Given a config file with observer section", func() {
			err = ioutil.WriteFile(fl, []byte(
				`Observer:
  Databases:
  - ID: xxxxx1
    Position: newest
  - ID: xxxxx2
    Position: oldest
  - ID: xxxxx3
    Position: ""`), 0644)
			So(err, ShouldBeNil)
			Convey("The LoadConfig func should return a pre-defined config", func() {
				cfg, err = loadConfig(fl)
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(len(cfg.Databases), ShouldEqual, 3)
				So(cfg.Databases[2].Position, ShouldEqual, "")
			})
		})
		Convey("Given a full config file", func() {
			err = ioutil.WriteFile(fl, []byte(
				`IsTestMode: true
WorkingRoot: "./"
PubKeyStoreFile: "public.keystore"
PrivateKeyFile: "private.key"
DHTFileName: "dht.db"
ListenAddr: "127.0.0.1:2122"
ThisNodeID: "00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9"
ValidDNSKeys:
  koPbw9wmYZ7ggcjnQ6ayHyhHaDNMYELKTqT+qRGrZpWSccr/lBcrm10Z1PuQHB3Azhii+sb0PYFkH1ruxLhe5g==: cloudflare.com
  mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==: cloudflare.com
MinNodeIDDifficulty: 2
DNSSeed:
  EnforcedDNSSEC: false
  DNSServers:
  - 1.1.1.1
  - 202.46.34.74
  - 202.46.34.75
  - 202.46.34.76
Observer:
  Databases:
  - ID: xxxxx1
    Position: newest
  - ID: xxxxx2
    Position: oldest
  - ID: xxxxx3
    Position: ""
BlockProducer:
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  NodeID: 00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9
  Nonce:
    a: 313283
    b: 0
    c: 0
    d: 0
  ChainFileName: "chain.db"
  BPGenesisInfo:
    Version: 1
    BlockHash: f745ca6427237aac858dd3c7f2df8e6f3c18d0f1c164e07a1c6b8eebeba6b154
    Producer: 0000000000000000000000000000000000000000000000000000000000000001
    MerkleRoot: 0000000000000000000000000000000000000000000000000000000000000001
    ParentHash: 0000000000000000000000000000000000000000000000000000000000000001
    Timestamp: "2019-01-10T12:49:07+08:00"
KnownNodes:
- ID: 00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9
  Nonce:
    a: 313283
    b: 0
    c: 0
    d: 0
  Addr: 127.0.0.1:2122
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  Role: Leader
- ID: 00000381d46fd6cf7742d7fb94e2422033af989c0e348b5781b3219599a3af35
  Nonce:
    a: 478373
    b: 0
    c: 0
    d: 2305843009893772025
  Addr: 127.0.0.1:2121
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  Role: Follower
- ID: 000000172580063ded88e010556b0aca2851265be8845b1ef397e8fce6ab5582
  Nonce:
    a: 259939
    b: 0
    c: 0
    d: 2305843012544226372
  Addr: 127.0.0.1:2120
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  Role: Follower
- ID: 00000f3b43288fe99831eb533ab77ec455d13e11fc38ec35a42d4edd17aa320d
  Nonce:
    a: 22403
    b: 0
    c: 0
    d: 0
  Addr: ""
  PublicKey: 02ec784ca599f21ef93fe7abdc68d78817ab6c9b31f2324d15ea174d9da498b4c4
  Role: Client
- ID: 000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade
  Nonce:
    a: 567323
    b: 0
    c: 0
    d: 3104982049
  Addr: 127.0.0.1:2144
  PublicKey: 0367aa51809a7c1dc0f82c02452fec9557b3e1d10ce7c919d8e73d90048df86d20
  Role: Miner
- ID: 000005f4f22c06f76c43c4f48d5a7ec1309cc94030cbf9ebae814172884ac8b5
  Nonce:
    a: 240524
    b: 0
    c: 0
    d: 2305843010430351476
  Addr: 127.0.0.1:2145
  PublicKey: 02914bca0806f040dd842207c44474ab41ecd29deee7f2d355789c5c78d448ca16
  Role: Miner
- ID: 000003f49592f83d0473bddb70d543f1096b4ffed5e5f942a3117e256b7052b8
  Nonce:
    a: 606016
    b: 0
    c: 0
    d: 13835058056920509601
  Addr: 127.0.0.1:2146
  PublicKey: 03ae859eac5b72ee428c7a85f10b2ce748d9de5e480aefbb70f6597dfa8b2175e5
  Role: Miner`), 0644)
			So(err, ShouldBeNil)
			Convey("The LoadConfig func should return a pre-defined config", func() {
				cfg, err = loadConfig(fl)
				So(err, ShouldBeNil)
				So(cfg, ShouldNotBeNil)
				So(len(cfg.Databases), ShouldEqual, 3)
				So(cfg.Databases[2].Position, ShouldEqual, "")
			})
		})
	})
}
