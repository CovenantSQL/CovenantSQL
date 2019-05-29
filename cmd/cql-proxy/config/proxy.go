/*
 * Copyright 2019 The CovenantSQL Authors.
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

package config

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// FaucetConfig defines the configurable options for public faucet service.
type FaucetConfig struct {
	Enabled           bool   `yaml:"Enabled"`
	Amount            uint64 `yaml:"Amount"`
	AddressDailyQuota int64  `yaml:"AddressDailyQuota"`
	AccountDailyQuota int64  `yaml:"AccountDailyQuota"`
}

// StorageConfig defines the persistence options for proxy service.
type StorageConfig struct {
	// use local sqlite3 database for persistence or not.
	UseLocalDatabase bool   `yaml:"UseLocalDatabase"`
	DatabaseID       string `yaml:"DatabaseID"`
}

// AdminAuthConfig defines the admin auth feature config for proxy service.
type AdminAuthConfig struct {
	// enable github oauth for admin feature, otherwise use admin password instead.
	OAuthEnabled bool          `yaml:"OAuthEnabled"`
	OAuthExpires time.Duration `yaml:"OAuthExpires"`

	// available if admin oauth enabled, used for public proxy service.
	GithubAppID     string `yaml:"GithubAppID"`
	GithubAppSecret string `yaml:"GithubAppSecret"`

	// available if admin oauth disabled, used for private proxy service.
	AdminPassword string   `yaml:"AdminPassword"`
	AdminProjects []string `yaml:"AdminProjects"`
}

// UserAuthConfig defines the user auth feature config for proxy service.
type UserAuthConfig struct {
	// globally enabled oauth/openid providers for all projects.
	Providers []string `yaml:"Providers"`

	// provider specific configs, first key is provider id, second key is provide config item.
	Extra map[string]map[string]interface{} `yaml:"Extra"`
}

// Config defines the configurable options for proxy service.
type Config struct {
	ListenAddr string `yaml:"ListenAddr"`
	// platform wildcard hosts for proxy to accept and dispatch requests.
	// project specific hosts is defined in project admin settings.
	Hosts []string `yaml:"Hosts"`

	// persistence config for proxy service.
	Storage *StorageConfig `yaml:"Storage"`

	// faucet config for public proxy service only.
	Faucet *FaucetConfig `yaml:"Faucet"`

	// admin auth config for proxy service.
	AdminAuth *AdminAuthConfig `yaml:"AdminAuth"`

	// user auth config for proxy service.
	UserAuth *UserAuthConfig `yaml:"UserAuth"`
}

type confWrapper struct {
	Proxy *Config `yaml:"Proxy"`
}

func (c *Config) validate() (err error) {
	if c.ListenAddr == "" {
		err = ErrInvalidProxyConfig
		log.Error("ListenAddr is not defined in faucet config")
		return
	}

	if err = c.Storage.validate(); err != nil {
		return
	}

	if err = c.Faucet.validate(); err != nil {
		return
	}

	if err = c.AdminAuth.validate(); err != nil {
		return
	}

	if err = c.UserAuth.validate(); err != nil {
		return
	}

	return
}

func (fc *FaucetConfig) validate() (err error) {
	if fc == nil {
		return
	}

	if fc.Amount <= 0 {
		err = ErrInvalidProxyConfig
		log.Error("a positive faucet amount is required for every applyToken")
		return
	}

	if fc.AddressDailyQuota == 0 || fc.AccountDailyQuota == 0 {
		log.Warning("the AddressDailyQuota & AccountDailyQuota should be valid positive number, set to 1 by default")

		if fc.AddressDailyQuota == 0 {
			fc.AddressDailyQuota = 1
		}
		if fc.AccountDailyQuota == 0 {
			fc.AccountDailyQuota = 1
		}

		return
	}

	return
}

func (pc *StorageConfig) validate() (err error) {
	if pc == nil {
		// do not store project config
		return
	}

	if pc.DatabaseID == "" {
		err = ErrInvalidProxyConfig
		log.Error("a database id is required for faucet applyToken persistence")
		return
	}

	return
}

func (ac *AdminAuthConfig) validate() (err error) {
	if ac == nil {
		err = ErrInvalidProxyConfig
		log.Error("admin oauth or fixture admin projects is required for proxy admin side config")
		return
	}

	if ac.OAuthEnabled {
		if ac.GithubAppID == "" || ac.GithubAppSecret == "" {
			err = ErrInvalidProxyConfig
			log.Error("admin github oauth config is required if admin github oauth is enabled")
			return
		}
	} else {
		if ac.AdminPassword == "" || len(ac.AdminProjects) == 0 {
			err = ErrInvalidProxyConfig
			log.Error("admin password and serving projects settings is required")
			return
		}
	}

	return
}

func (ac *UserAuthConfig) validate() (err error) {
	if ac == nil || len(ac.Providers) == 0 {
		log.Warning("all oauth provider is enabled for projects")
	}

	return
}

// LoadConfig load the common covenantsql client config again for extra faucet config.
func LoadConfig(listenAddr string, configPath string) (config *Config, err error) {
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

	if configWrapper.Proxy == nil {
		err = ErrInvalidProxyConfig
		log.WithError(err).Error("could not read proxy config")
		return
	}

	config = configWrapper.Proxy

	// override config
	if listenAddr != "" {
		config.ListenAddr = listenAddr
	}

	return
}
