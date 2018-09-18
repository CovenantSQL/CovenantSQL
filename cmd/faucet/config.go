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
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"gopkg.in/yaml.v2"
)

// Config defines the configurable options for faucet application backend.
type Config struct {
	// faucet server related
	ListenAddr           string        `yaml:"ListenAddr"`
	URLRequired          string        `yaml:"URLRequired"` // can be a part of a valid url
	ContentRequired      string        `yaml:"ContentRequired"`
	FaucetAmount         int64         `yaml:"FaucetAmount"`
	DatabaseID           string        `yaml:"DatabaseID"`       // database id for persistence
	LocalDatabase        bool          `yaml:"UseLocalDatabase"` // use local sqlite3 database for persistence
	AddressDailyLimit    uint          `yaml:"AddressDailyLimit"`
	AccountDailyLimit    uint          `yaml:"AccountDailyLimit"`
	VerificationInterval time.Duration `yaml:"VerificationInterval"`
}

type confWrapper struct {
	Faucet *Config `yaml:"Faucet"`
}

// LoadConfig load the common covenantsql client config again for extra faucet config.
func LoadConfig(configPath string) (config *Config, err error) {
	var configBytes []byte
	if configBytes, err = ioutil.ReadFile(configPath); err != nil {
		log.Errorf("read config file failed: %v", err)
		return
	}

	configWrapper := &confWrapper{}
	if err = yaml.Unmarshal(configBytes, configWrapper); err != nil {
		log.Errorf("unmarshal config file failed: %v", err)
		return
	}

	if configWrapper.Faucet == nil {
		err = ErrInvalidFaucetConfig
		log.Errorf("could not read faucet config: %v", err)
		return
	}

	config = configWrapper.Faucet

	// validate config
	if config.ListenAddr == "" {
		err = ErrInvalidFaucetConfig
		log.Error("ListenAddr is not defined in faucet config")
		return
	}

	if config.URLRequired == "" && config.ContentRequired == "" {
		err = ErrInvalidFaucetConfig
		log.Error("at least one URL/Content config for faucet application is required")
		return
	}

	if config.DatabaseID == "" {
		err = ErrInvalidFaucetConfig
		log.Error("a database id is required for faucet application persistence")
		return
	}

	if config.FaucetAmount <= 0 {
		err = ErrInvalidFaucetConfig
		log.Error("a positive faucet amount is required for every application")
		return
	}

	if config.AddressDailyLimit == 0 || config.AccountDailyLimit == 0 {
		log.Warningf("AddressDailyLimit & AccountDailyLimit should be valid positive number, 1 assumed")

		if config.AddressDailyLimit == 0 {
			config.AddressDailyLimit = 1
		}
		if config.AccountDailyLimit == 0 {
			config.AccountDailyLimit = 1
		}

		return
	}

	if config.VerificationInterval.Nanoseconds() <= 0 {
		log.Warningf("a valid VerificationInterval is required, 30 seconds assumed")

		config.VerificationInterval = 30 * time.Second
	}

	return
}
