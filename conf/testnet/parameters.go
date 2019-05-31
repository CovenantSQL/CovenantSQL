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
	yaml "gopkg.in/yaml.v2"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	// CQLConfigYAML is the config string in YAML format of the CovenantSQL TestNet.
	CQLConfigYAML = `
SQLChainPeriod: 60s
DNSSeed:
  Domain: "testnet.gridb.io"
  BPCount: 6
`
	// CQLMinerYAML is the config string in YAML format of the CovenantSQL TestNet for miner.
	CQLMinerYAML = `
BillingBlockCount: 60
BPPeriod: 10s
BPTick: 3s
SQLChainTick: 10s
SQLChainTTL: 10
ChainBusPeriod: 10s
MinProviderDeposit: 1000000
Miner:
  RootDir: './data'
  MaxReqTimeGap: '5m'
  ProvideServiceInterval: '10s'
`
)

// GetTestNetConfig parses and returns the CovenantSQL TestNet config.
func GetTestNetConfig() (config *conf.Config) {
	var err error
	config = &conf.Config{}
	if err = yaml.Unmarshal([]byte(CQLConfigYAML), config); err != nil {
		log.WithError(err).Fatal("failed to unmarshal testnet config")
	}
	return
}

// SetMinerConfig set testnet common config for miner.
func SetMinerConfig(config *conf.Config) {
	if config == nil {
		return
	}
	var err error
	if err = yaml.Unmarshal([]byte(CQLMinerYAML), config); err != nil {
		log.WithError(err).Fatal("failed to unmarshal testnet miner config")
	}
	return
}
