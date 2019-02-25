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

	yaml "gopkg.in/yaml.v2"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// Config defines the configurable options for faucet application backend.
type Config struct {
	// faucet server related
	ListenAddr           string        `yaml:"ListenAddr"`
	URLRequired          string        `yaml:"URLRequired"` // can be a part of a valid url
	ContentRequired      []string      `yaml:"ContentRequired"`
	FaucetAmount         int64         `yaml:"FaucetAmount"`
	DatabaseID           string        `yaml:"DatabaseID"`       // database id for persistence
	LocalDatabase        bool          `yaml:"UseLocalDatabase"` // use local sqlite3 database for persistence
	AddressDailyQuota    uint          `yaml:"AddressDailyQuota"`
	AccountDailyQuota    uint          `yaml:"AccountDailyQuota"`
	VerificationInterval time.Duration `yaml:"VerificationInterval"`
}

type confWrapper struct {
	Faucet *Config `yaml:"Faucet"`
}

// LoadConfig load the common covenantsql client config again for extra faucet config.
func LoadConfig(configPath string) (config *Config, err error) {
	var configBytes []byte
	if configBytes, err = ioutil.ReadFile(configPath); err != nil {
		log.WithError(err).Error("read config file failed")
		return
	}

	configWrapper := &confWrapper{}
	if err = yaml.Unmarshal(configBytes, configWrapper); err != nil {
		log.WithError(err).Error("unmarshal config file failed")
		return
	}

	if configWrapper.Faucet == nil {
		err = ErrInvalidFaucetConfig
		log.WithError(err).Error("could not read faucet config")
		return
	}

	config = configWrapper.Faucet

	// validate config
	if config.ListenAddr == "" {
		err = ErrInvalidFaucetConfig
		log.Error("ListenAddr is not defined in faucet config")
		return
	}

	if config.URLRequired == "" && len(config.ContentRequired) == 0 {
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

	if config.AddressDailyQuota == 0 || config.AccountDailyQuota == 0 {
		log.Warning("AddressDailyQuota & AccountDailyQuota should be valid positive number, 1 assumed")

		if config.AddressDailyQuota == 0 {
			config.AddressDailyQuota = 1
		}
		if config.AccountDailyQuota == 0 {
			config.AccountDailyQuota = 1
		}

		return
	}

	if config.VerificationInterval.Nanoseconds() <= 0 {
		log.Warning("a valid VerificationInterval is required, 30 seconds assumed")

		config.VerificationInterval = 30 * time.Second
	}

	return
}
