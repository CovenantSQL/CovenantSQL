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

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	validator "gopkg.in/go-playground/validator.v9"
	yaml "gopkg.in/yaml.v2"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// FaucetConfig defines the configurable options for public faucet service.
type FaucetConfig struct {
	Enabled           bool   `yaml:"Enabled"`
	Amount            uint64 `yaml:"Amount" validate:"required_with=Enabled,gt=0"`
	AddressDailyQuota int64  `yaml:"AddressDailyQuota" validate:"required_with=Enabled,gt=0"`
	AccountDailyQuota int64  `yaml:"AccountDailyQuota" validate:"required_with=Enabled,gt=0"`
}

// StorageConfig defines the persistence options for proxy service.
type StorageConfig struct {
	// use local sqlite3 database for persistence or not.
	UseLocalDatabase bool   `yaml:"UseLocalDatabase"`
	DatabaseID       string `yaml:"DatabaseID" validate:"required"`
}

// AdminAuthConfig defines the admin auth feature config for proxy service.
type AdminAuthConfig struct {
	// enable github oauth for admin feature, otherwise use admin password instead.
	OAuthEnabled bool          `yaml:"OAuthEnabled"`
	OAuthExpires time.Duration `yaml:"OAuthExpires" validate:"required,gt=0"`

	// available if admin oauth enabled, used for public proxy service.
	GithubAppID     []string `yaml:"GithubAppID" validate:"required_with=OAuthEnabled,dive,required"`
	GithubAppSecret []string `yaml:"GithubAppSecret" validate:"required_with=OAuthEnabled,dive,required"`

	// available if admin oauth disabled, used for private proxy service.
	AdminPassword string   `yaml:"AdminPassword" validate:"required_without=OAuthEnabled"`
	AdminProjects []string `yaml:"AdminProjects" validate:"required_without=OAuthEnabled,dive,required,len=64"`
}

// UserAuthConfig defines the user auth feature config for proxy service.
type UserAuthConfig struct {
	// globally enabled oauth/openid providers for all projects.
	Providers []string `yaml:"Providers" validate:"required"`

	// provider specific configs, first key is provider id, second key is provide config item.
	Extra map[string]gin.H `yaml:"Extra"`
}

// Config defines the configurable options for proxy service.
type Config struct {
	ListenAddr string `yaml:"ListenAddr" validate:"required"`
	// platform wildcard hosts for proxy to accept and dispatch requests.
	// project specific hosts is defined in project admin settings.
	Hosts []string `yaml:"Hosts" validate:"dive,required"`

	// persistence config for proxy service.
	Storage *StorageConfig `yaml:"Storage" validate:"required"`

	// faucet config for public proxy service only.
	Faucet *FaucetConfig `yaml:"Faucet"`

	// admin auth config for proxy service.
	AdminAuth *AdminAuthConfig `yaml:"AdminAuth" validate:"required"`

	// user auth config for proxy service.
	UserAuth *UserAuthConfig `yaml:"UserAuth" validate:"required"`
}

type confWrapper struct {
	Proxy *Config `yaml:"Proxy"`
}

func (c *Config) Validate() (err error) {
	c.Faucet.fixConfig()

	validate := validator.New()
	if err = validate.Struct(*c); err != nil {
		return
	}
	if c.Storage != nil {
		if err = validate.Struct(*c.Storage); err != nil {
			return
		}
	}
	if c.Faucet != nil {
		if err = validate.Struct(*c.Faucet); err != nil {
			return
		}
	}
	if c.AdminAuth != nil {
		if err = validate.Struct(*c.AdminAuth); err != nil {
			return
		}

		if len(c.AdminAuth.GithubAppID) != len(c.AdminAuth.GithubAppSecret) {
			err = errors.Wrap(ErrInvalidProxyConfig, "mismatched admin appid and appsecret")
			return
		}
	}
	if c.UserAuth != nil {
		if err = validate.Struct(*c.UserAuth); err != nil {
			return
		}
	}

	return
}

func (fc *FaucetConfig) fixConfig() {
	if fc == nil {
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

	err = config.Validate()

	return
}
