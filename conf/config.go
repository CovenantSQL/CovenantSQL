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

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/pow/cpuminer"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gopkg.in/yaml.v2"
)

// BPInfo hold all BP info fields
type BPInfo struct {
	// PublicKeyStr is the public key of Block Producer
	PublicKeyStr string `yaml:"PublicKeyStr"`
	// PublicKey point to BlockProducer public key
	PublicKey *asymmetric.PublicKey `yaml:"-"`
	// NodeID is the node id of Block Producer
	NodeID proto.NodeID `yaml:"NodeID"`
	// RawNodeID
	RawNodeID proto.RawNodeID `yaml:"-"`
	// Nonce is the nonce, SEE: cmd/idminer for more
	Nonce cpuminer.Uint256 `yaml:"Nonce"`
}

// NodeInfo for conf generation and load purpose.
type NodeInfo struct {
	ID        proto.NodeID
	Nonce     cpuminer.Uint256
	PublicKey *asymmetric.PublicKey `yaml:"-"`
	Addr      string
	Role      kayak.ServerRole
}

// Config holds all the config read from yaml config file
type Config struct {
	IsTestMode      bool //when testMode use default empty masterKey
	GenerateKeyPair bool `yaml:"-"`
	WorkingRoot     string
	PubKeyStoreFile string
	PrivateKeyFile  string
	DHTFileName     string
	ListenAddr      string
	ThisNodeID      proto.NodeID

	BP *BPInfo `yaml:"BlockProducer"`

	KnownNodes *[]NodeInfo
}

// GConf is the global config pointer
var GConf *Config

// LoadConfig loads config from configPath
func LoadConfig(configPath string) (config *Config, err error) {
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Errorf("read config file failed: %s", err)
		return
	}
	config = &Config{}
	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		log.Errorf("unmarshal config file failed: %s", err)
		return
	}
	return
}
