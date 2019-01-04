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
BillingBlockCount: 60
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
  PublicKey: "02c1db96f2ba7e1cb4e9822d12de0f63fb666feb828c7f509e81fab9bd7a34039c"
  NodeID: 00000000000589366268c274fdc11ec8bdb17e668d2f619555a2e9c1a29c91d8
  Nonce:
    a: 14396347928
    b: 0
    c: 0
    d: 6148914694092305796
  ChainFileName: "chain.db"
  BPGenesisInfo:
    Version: 1
    BlockHash: f745ca6427237aac858dd3c7f2df8e6f3c18d0f1c164e07a1c6b8eebeba6b154
    Producer: 0000000000000000000000000000000000000000000000000000000000000001
    MerkleRoot: 0000000000000000000000000000000000000000000000000000000000000001
    ParentHash: 0000000000000000000000000000000000000000000000000000000000000001
    Timestamp: 2019-01-02T13:33:00.00Z
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
      - Address: 58aceaf4b730b54bf00c0fb3f7b14886de470767f313c2d108968cd8bf0794b7
        StableCoinBalance: 10000000000000000000
        CovenantCoinBalance: 10000000000000000000
KnownNodes:
- ID: 00000000000589366268c274fdc11ec8bdb17e668d2f619555a2e9c1a29c91d8
  Nonce:
    a: 14396347928
    b: 0
    c: 0
    d: 6148914694092305796
  Addr: bp00.cn.gridb.io:7777
  PublicKey: "02c1db96f2ba7e1cb4e9822d12de0f63fb666feb828c7f509e81fab9bd7a34039c"
  Role: Leader
- ID: 000000000013fd4b3180dd424d5a895bc57b798e5315087b7198c926d8893f98
  Nonce:
    a: 789554103
    b: 0
    c: 0
    d: 8070450536379825883
  Addr: bp01.cn.gridb.io:7777
  PublicKey: "02c1db96f2ba7e1cb4e9822d12de0f63fb666feb828c7f509e81fab9bd7a34039c"
  Role: Follower
- ID: 00000000001771e2b2e12b6f9f85d58ef5261a4b98a2e80bba0c5ef7bd72c499
  Nonce:
    a: 1822880492
    b: 0
    c: 0
    d: 8646911286604382906
  Addr: bp02.cn.gridb.io:7777
  PublicKey: "02c1db96f2ba7e1cb4e9822d12de0f63fb666feb828c7f509e81fab9bd7a34039c"
  Role: Follower
- ID: 000000000014a2f14e79aec0a27a2a669aab416c392d5577760d43ed8503020d
  Nonce:
    a: 2552803966
    b: 0
    c: 0
    d: 9079256850862786277
  Addr: bp03.cn.gridb.io:7777
  PublicKey: "02c1db96f2ba7e1cb4e9822d12de0f63fb666feb828c7f509e81fab9bd7a34039c"
  Role: Follower
- ID: 00000000003b2bd120a7d07f248b181fc794ba8b278f07f9a780e61eb77f6abb
  Nonce:
    a: 2449538793
    b: 0
    c: 0
    d: 8791026473473316840
  Addr: bp04.hk.gridb.io:7777
  PublicKey: "02c1db96f2ba7e1cb4e9822d12de0f63fb666feb828c7f509e81fab9bd7a34039c"
  Role: Follower
- ID: 0000000000293f7216362791b6b1c9772184d6976cb34310c42547735410186c
  Nonce:
    a: 746598970
    b: 0
    c: 0
    d: 10808639108098016056
  Addr: bp05.cn.gridb.io:7777
  PublicKey: "02c1db96f2ba7e1cb4e9822d12de0f63fb666feb828c7f509e81fab9bd7a34039c"
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
  Role: Miner
- ID: 00000f3b43288fe99831eb533ab77ec455d13e11fc38ec35a42d4edd17aa320d
  Nonce:
    a: 22403
    b: 0
    c: 0
    d: 0
  Addr: ""
  PublicKey: 02ec784ca599f21ef93fe7abdc68d78817ab6c9b31f2324d15ea174d9da498b4c4
  Role: Client
`
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
