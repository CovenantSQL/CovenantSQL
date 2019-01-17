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
	"io/ioutil"
	"path"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	yaml "gopkg.in/yaml.v2"
)

// these const specify the role of this app, which can be "miner", "blockProducer"
const (
	MinerBuildTag         = "M"
	BlockProducerBuildTag = "B"
	ClientBuildTag        = "C"
	UnknownBuildTag       = "U"
)

// StartSucceedMessage is printed when CovenantSQL started successfully.
const StartSucceedMessage = "CovenantSQL Started Successfully"

// RoleTag indicate which role the daemon is playing.
var RoleTag = UnknownBuildTag

// BaseAccountInfo defines base info to build a BaseAccount.
type BaseAccountInfo struct {
	Address             hash.Hash `yaml:"Address"`
	StableCoinBalance   uint64    `yaml:"StableCoinBalance"`
	CovenantCoinBalance uint64    `yaml:"CovenantCoinBalance"`
}

// BPGenesisInfo hold all genesis info fields.
type BPGenesisInfo struct {
	// Version defines the block version
	Version int32 `yaml:"Version"`
	// Timestamp defines the initial time of chain
	Timestamp time.Time `yaml:"Timestamp"`
	// BaseAccounts defines the base accounts for testnet
	BaseAccounts []BaseAccountInfo `yaml:"BaseAccounts"`
}

// BPInfo hold all BP info fields.
type BPInfo struct {
	// PublicKey point to BlockProducer public key
	PublicKey *asymmetric.PublicKey `yaml:"PublicKey"`
	// NodeID is the node id of Block Producer
	NodeID proto.NodeID `yaml:"NodeID"`
	// RawNodeID
	RawNodeID proto.RawNodeID `yaml:"-"`
	// Nonce is the nonce, SEE: cmd/cql-utils for more
	Nonce cpuminer.Uint256 `yaml:"Nonce"`
	// ChainFileName is the chain db's name
	ChainFileName string `yaml:"ChainFileName"`
	// BPGenesis is the genesis block filed
	BPGenesis BPGenesisInfo `yaml:"BPGenesisInfo,omitempty"`
}

// MinerDatabaseFixture config.
type MinerDatabaseFixture struct {
	DatabaseID               proto.DatabaseID `yaml:"DatabaseID"`
	Term                     uint64           `yaml:"Term"`
	Leader                   proto.NodeID     `yaml:"Leader"`
	Servers                  []proto.NodeID   `yaml:"Servers"`
	GenesisBlockFile         string           `yaml:"GenesisBlockFile"`
	AutoGenerateGenesisBlock bool             `yaml:"AutoGenerateGenesisBlock,omitempty"`
}

// MinerInfo for miner config.
type MinerInfo struct {
	// node basic config.
	RootDir                string                 `yaml:"RootDir"`
	MaxReqTimeGap          time.Duration          `yaml:"MaxReqTimeGap,omitempty"`
	ProvideServiceInterval time.Duration          `yaml:"ProvideServiceInterval,omitempty"`
	TargetUsers            []proto.AccountAddress `yaml:"TargetUsers,omitempty"`

	// when test mode, fixture database config is used.
	IsTestMode   bool                    `yaml:"IsTestMode,omitempty"`
	TestFixtures []*MinerDatabaseFixture `yaml:"TestFixtures,omitempty"`
}

// DNSSeed defines seed DNS info.
type DNSSeed struct {
	EnforcedDNSSEC bool     `yaml:"EnforcedDNSSEC"`
	DNSServers     []string `yaml:"DNSServers"`
}

// Config holds all the config read from yaml config file.
type Config struct {
	IsTestMode bool `yaml:"IsTestMode,omitempty"` // when testMode use default empty masterKey and test DNS domain
	// StartupSyncHoles indicates synchronizing hole blocks from other peers on BP
	// startup/reloading.
	StartupSyncHoles bool `yaml:"StartupSyncHoles,omitempty"`
	GenerateKeyPair  bool `yaml:"-"`
	//TODO(auxten): set yaml key for config
	WorkingRoot     string            `yaml:"WorkingRoot"`
	PubKeyStoreFile string            `yaml:"PubKeyStoreFile"`
	PrivateKeyFile  string            `yaml:"PrivateKeyFile"`
	DHTFileName     string            `yaml:"DHTFileName"`
	ListenAddr      string            `yaml:"ListenAddr"`
	ThisNodeID      proto.NodeID      `yaml:"ThisNodeID"`
	ValidDNSKeys    map[string]string `yaml:"ValidDNSKeys"` // map[DNSKEY]domain
	// Check By BP DHT.Ping
	MinNodeIDDifficulty int `yaml:"MinNodeIDDifficulty"`

	DNSSeed DNSSeed `yaml:"DNSSeed"`

	BP    *BPInfo    `yaml:"BlockProducer"`
	Miner *MinerInfo `yaml:"Miner,omitempty"`

	KnownNodes  []proto.Node `yaml:"KnownNodes"`
	SeedBPNodes []proto.Node `yaml:"-"`

	QPS                uint32        `yaml:"QPS"`
	ChainBusPeriod     time.Duration `yaml:"ChainBusPeriod"`
	BillingBlockCount  uint64        `yaml:"BillingBlockCount"` // BillingBlockCount is for sql chain miners syncing billing with main chain
	BPPeriod           time.Duration `yaml:"BPPeriod"`
	BPTick             time.Duration `yaml:"BPTick"`
	SQLChainPeriod     time.Duration `yaml:"SQLChainPeriod"`
	SQLChainTick       time.Duration `yaml:"SQLChainTick"`
	SQLChainTTL        int32         `yaml:"SQLChainTTL"`
	MinProviderDeposit uint64        `yaml:"MinProviderDeposit"`
}

// GConf is the global config pointer.
var GConf *Config

// LoadConfig loads config from configPath.
func LoadConfig(configPath string) (config *Config, err error) {
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.WithError(err).Error("read config file failed")
		return
	}
	config = &Config{}
	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		log.WithError(err).Error("unmarshal config file failed")
		return
	}

	configDir := path.Dir(configPath)
	if !path.IsAbs(config.PubKeyStoreFile) {
		config.PubKeyStoreFile = path.Join(configDir, config.PubKeyStoreFile)
	}

	if !path.IsAbs(config.PrivateKeyFile) {
		config.PrivateKeyFile = path.Join(configDir, config.PrivateKeyFile)
	}

	if !path.IsAbs(config.DHTFileName) {
		config.DHTFileName = path.Join(configDir, config.DHTFileName)
	}

	if !path.IsAbs(config.WorkingRoot) {
		config.WorkingRoot = path.Join(configDir, config.WorkingRoot)
	}

	if config.BP != nil && !path.IsAbs(config.BP.ChainFileName) {
		config.BP.ChainFileName = path.Join(configDir, config.BP.ChainFileName)
	}

	if config.Miner != nil && !path.IsAbs(config.Miner.RootDir) {
		config.Miner.RootDir = path.Join(configDir, config.Miner.RootDir)
	}
	return
}
