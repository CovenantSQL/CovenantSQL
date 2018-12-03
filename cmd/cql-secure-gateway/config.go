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
	"sync"

	cw "github.com/CovenantSQL/CovenantSQL/cmd/cql-secure-gateway/casbin"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/casbin/casbin"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type encryptionConfig struct {
	KeyPath string                 `yaml:"Key"`    // key path for encryption config
	Key     *asymmetric.PrivateKey `yaml:"-"`      // encryption key
	Fields  []cw.Field             `yaml:"Fields"` // encrypted fields config, support field group defines in grants config
}

type authConfig struct {
	encryptionCacheLock          sync.RWMutex
	Users                        map[string]string   `yaml:"Users"`      // user to password mapping
	Grants                       *cw.Config          `yaml:"Grants"`     // casbin grants config
	Enforcer                     *casbin.Enforcer    `yaml:"-"`          // casbin enforcer
	Encryption                   []*encryptionConfig `yaml:"Encryption"` // encryption fields settings
	EncryptionFieldCache         map[string]int      `yaml:"-"`          // field cache of encryption
	WildcardEncryptionFieldCache map[string]int      `yaml:"-"`          // field cache of wildcard encryption fields
}

type sgConfig struct {
	ListenAddr string      `yaml:"ListenAddr"` // mysql compatible rpc listen address
	AuthConfig *authConfig `yaml:"Auth"`       // authentication related config
}

type configWrapper struct {
	Config *sgConfig `yaml:"SecureGateway"` // secure gateway config
}

func loadConfig(configFile string) (cfg *sgConfig, err error) {
	var (
		configBytes []byte
		wrapper     *configWrapper
	)
	if configBytes, err = ioutil.ReadFile(configFile); err != nil {
		log.WithError(err).Error("failed to read config file")
		return
	}
	if err = yaml.Unmarshal(configBytes, &wrapper); err != nil {
		log.WithError(err).Error("failed to parse config file")
		return
	}

	cfg = wrapper.Config
	err = cfg.buildAndValidate()

	return
}

func (sc *sgConfig) buildAndValidate() (err error) {
	if sc == nil {
		err = errors.Wrap(ErrInvalidConfig, "nil secure gateway config")
		return
	}
	if sc.ListenAddr == "" {
		err = errors.Wrap(ErrInvalidConfig, "empty listen addr")
		log.WithError(err).Error("invalid config")
		return
	}
	if sc.AuthConfig == nil {
		err = errors.Wrap(ErrInvalidConfig, "nil auth config")
		log.WithError(err).Error("invalid config")
		return
	}
	if err = sc.AuthConfig.buildEnforcer(); err != nil {
		log.WithError(err).Error("failed to load grants")
		return
	}
	if err = sc.AuthConfig.buildEncryption(); err != nil {
		log.WithError(err).Error("failed to load encryption config")
		return
	}

	return
}

func (ac *authConfig) buildEnforcer() (err error) {
	ac.Enforcer, err = cw.NewCasbin(ac.Grants)
	return
}

func (ac *authConfig) buildEncryption() (err error) {
	var workingRootDir string

	if conf.GConf != nil && conf.GConf.WorkingRoot != "" {
		workingRootDir = conf.GConf.WorkingRoot
	} else {
		if workingRootDir, err = os.Getwd(); err != nil {
			return
		}
	}

	ac.EncryptionFieldCache = map[string]int{}
	ac.WildcardEncryptionFieldCache = map[string]int{}

	ac.encryptionCacheLock.Lock()
	defer ac.encryptionCacheLock.Unlock()

	// load and validate encryption config to match with grants
	for i, ec := range ac.Encryption {
		if err = ac.buildSingleEncryptionConfig(workingRootDir, i, ec); err != nil {
			// return
			break
		}
	}

	return
}

func (ac *authConfig) buildSingleEncryptionConfig(wd string, configIndex int, ec *encryptionConfig) (err error) {
	// validate encryption keys existence
	if !path.IsAbs(ec.KeyPath) {
		ec.KeyPath = path.Join(wd, ec.KeyPath)
	}

	// load key, does not support password empowered private key
	if ec.Key, err = kms.LoadPrivateKey(ec.KeyPath, []byte{}); err != nil {
		err = errors.Wrapf(err, "load key %s failed", ec.KeyPath)
		return
	}

	// validate fields
	for _, f := range ec.Fields {
		// set to flatten mapping
		if err = ac.checkAndBuildEncryptionCache(&f, configIndex); err != nil {
			return
		}
	}

	return
}

func (ac *authConfig) checkAndBuildEncryptionCache(f *cw.Field, ecOffset int) (err error) {
	// check field encryption configuration over-lapping
	// iterate existing setting

	for field, o := range ac.EncryptionFieldCache {
		if f.MatchesString(field) {
			if o != ecOffset {
				err = errors.Wrapf(ErrFieldEncryption,
					"duplicated encryption assigned to column (previous: %s -> %s, current: %s -> %s)",
					field, ac.Encryption[o].KeyPath,
					f.String(), ac.Encryption[ecOffset].KeyPath,
				)
				return
			}
		}
	}

	if !f.IsWildcard() {
		for field, o := range ac.WildcardEncryptionFieldCache {
			if f.MatchesString(field) {
				if o != ecOffset {
					err = errors.Wrapf(ErrFieldEncryption,
						"duplicated encryption assigned to column (previous: %s -> %s, current: %s -> %s)",
						field, ac.Encryption[o].KeyPath,
						f.String(), ac.Encryption[ecOffset].KeyPath,
					)
					return
				}
			}
		}

		// set
		ac.EncryptionFieldCache[f.String()] = ecOffset
	} else {
		ac.WildcardEncryptionFieldCache[f.String()] = ecOffset
	}

	return
}

// GetEncryptionConfig returns encryption config for specified field.
func (ac *authConfig) GetEncryptionConfig(f *cw.Field) (ecConfig *encryptionConfig, err error) {
	if f == nil {
		return
	}
	if f.IsWildcard() {
		err = errors.Wrap(ErrInvalidField, "could not get wildcard field encryption key")
		return
	}

	var (
		fieldStr        = f.String()
		ecOffset        = -1
		shouldFillCache bool
	)

	func() {
		ac.encryptionCacheLock.RLock()
		defer ac.encryptionCacheLock.RUnlock()
		if o, exists := ac.EncryptionFieldCache[fieldStr]; exists {
			// hit
			ecOffset = o
			return
		}
		for wf, o := range ac.WildcardEncryptionFieldCache {
			if f.MatchesString(wf) {
				// hit
				ecOffset = o
				shouldFillCache = true
				return
			}
		}
		// plain field
		shouldFillCache = true
	}()

	if shouldFillCache {
		func() {
			ac.encryptionCacheLock.Lock()
			defer ac.encryptionCacheLock.Unlock()

			ac.EncryptionFieldCache[fieldStr] = ecOffset
		}()
	}

	if ecOffset < 0 {
		return
	}

	ecConfig = ac.Encryption[ecOffset]
	return
}
