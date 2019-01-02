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

// Package testnet contains the paraemters of the CovenantSQL TestNet.
package testnet

import (
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	yaml "gopkg.in/yaml.v2"
)

const (
	// TestNetConfigYAML is the config string in YAML format of the CovenantSQL TestNet.
	TestNetConfigYAML = `IsTestMode: true
StartupSyncHoles: true
WorkingRoot: "./"
PubKeyStoreFile: "public.keystore"
PrivateKeyFile: "private.key"
DHTFileName: "dht.db"
ListenAddr: "0.0.0.0:15151"
ThisNodeID: "00000f3b43288fe99831eb533ab77ec455d13e11fc38ec35a42d4edd17aa320d"
QPS: 1000
BillingPeriod: 60
BPPeriod: 10s
BPTick: 3s
SQLChainPeriod: 60s
SQLChainTick: 10s
SQLChainTTL: 10
MinProviderDeposit: 1000000
ValidDNSKeys:
  koPbw9wmYZ7ggcjnQ6ayHyhHaDNMYELKTqT+qRGrZpWSccr/lBcrm10Z1PuQHB3Azhii+sb0PYFkH1ruxLhe5g==: cloudflare.com
  mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ==: cloudflare.com
  oJMRESz5E4gYzS/q6XDrvU1qMPYIjCWzJaOau8XNEZeqCYKD5ar0IRd8KqXXFJkqmVfRvMGPmM1x8fGAa2XhSA==: cloudflare.com
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
    Timestamp: 2018-12-28T12:20:59.12Z
    BaseAccounts:
      - Address: ba0ba731c7a76ccef2c1170f42038f7e228dfb474ef0190dfe35d9a37911ed37
        StableCoinBalance: 10000000000000000000
        CovenantCoinBalance: 10000000000000000000
      - Address: 1a7b0959bbd0d0ec529278a61c0056c277bffe75b2646e1699b46b10a90210be
        StableCoinBalance: 10000000000000000000
        CovenantCoinBalance: 10000000000000000000
      - Address: 9235bc4130a2ed4e6c35ea189dab35198ebb105640bedb97dd5269cc80863b16
        StableCoinBalance: 10000000000000000000
        CovenantCoinBalance: 10000000000000000000
      - Address: 9e1618775cceeb19f110e04fbc6c5bca6c8e4e9b116e193a42fe69bf602e7bcd
        StableCoinBalance: 10000000000000000000
        CovenantCoinBalance: 10000000000000000000
KnownNodes:
- ID: 00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9
  Nonce:
    a: 313283
    b: 0
    c: 0
    d: 0
  Addr: bp00.cn.gridb.io:7777
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  Role: Leader
- ID: 00000381d46fd6cf7742d7fb94e2422033af989c0e348b5781b3219599a3af35
  Nonce:
    a: 478373
    b: 0
    c: 0
    d: 2305843009893772025
  Addr: bp01.cn.gridb.io:7777
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  Role: Follower
- ID: 000000172580063ded88e010556b0aca2851265be8845b1ef397e8fce6ab5582
  Nonce:
    a: 259939
    b: 0
    c: 0
    d: 2305843012544226372
  Addr: bp02.cn.gridb.io:7777
  PublicKey: "02c76216704d797c64c58bc11519fb68582e8e63de7e5b3b2dbbbe8733efe5fd24"
  Role: Follower
- ID: 000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade
  Nonce:
    a: 567323
    b: 0
    c: 0
    d: 3104982049
  Addr: miner00.cn.gridb.io:7778
  PublicKey: 0367aa51809a7c1dc0f82c02452fec9557b3e1d10ce7c919d8e73d90048df86d20
  Role: Miner
- ID: 000005f4f22c06f76c43c4f48d5a7ec1309cc94030cbf9ebae814172884ac8b5
  Nonce:
    a: 240524
    b: 0
    c: 0
    d: 2305843010430351476
  Addr: miner01.cn.gridb.io:7778
  PublicKey: 02914bca0806f040dd842207c44474ab41ecd29deee7f2d355789c5c78d448ca16
  Role: Miner`
)

// GetTestNetConfig parses and returns the CovenantSQL TestNet config.
func GetTestNetConfig() (config *conf.Config) {
	var err error
	config = &conf.Config{}
	if err = yaml.Unmarshal([]byte(TestNetConfigYAML), config); err != nil {
		log.WithError(err).Fatal("failed to unmarshal testnet config")
	}
	return
}
