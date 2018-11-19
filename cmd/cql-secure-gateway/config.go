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
	"strings"

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
	Fields  []string               `yaml:"Fields"` // encrypted fields config, support field group defines in grants config
}

type authConfig struct {
	Users                   map[string]string   `yaml:"Users"`      // user to password mapping
	Grants                  *cw.Config          `yaml:"Grants"`     // casbin grants config
	Enforcer                *casbin.Enforcer    `yaml:"-"`          // casbin enforcer
	Encryption              []*encryptionConfig `yaml:"Encryption"` // encryption fields settings
	FlattenEncryptionConfig map[string]int      `yaml:"-"`          // field to key offset
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

	ac.FlattenEncryptionConfig = make(map[string]int)

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
		if ac.Grants != nil && ac.Grants.FieldGroup != nil {
			if _, exists := ac.Grants.FieldGroup[f]; exists {
				// exists in field group
				// flatten

				for _, sf := range ac.Grants.FieldGroup[f] {
					if err = ac.setFieldEncryptionKey(sf, configIndex); err != nil {
						return
					}
				}

				continue
			}
		}

		// validate as normal <db>.<table>.<column> field
		if err = ac.validateField(f); err != nil {
			return
		}

		// set to flatten mapping
		if err = ac.setFieldEncryptionKey(f, configIndex); err != nil {
			return
		}
	}

	return
}

func (ac *authConfig) setFieldEncryptionKey(f string, ecOffset int) (err error) {
	// check field encryption configuration over-lapping
	// iterate existing setting

	for p, o := range ac.FlattenEncryptionConfig {
		var m bool
		if m, err = fieldMatches(p, f); err != nil {
			return
		}

		if m && o != ecOffset {
			err = errors.Wrapf(ErrFieldEncryption,
				"duplicated encryption assigned to column (previous: %s -> %s, current: %s -> %s)",
				p, ac.Encryption[o].KeyPath,
				f, ac.Encryption[ecOffset].KeyPath,
			)
			return
		}
	}

	// set new record
	ac.FlattenEncryptionConfig[f] = ecOffset

	return
}

func (ac *authConfig) validateField(f string) (err error) {
	parts := strings.Split(f, ".")

	if len(parts) != 3 {
		err = errors.Wrapf(ErrInvalidField,
			"invalid field: %s, should be in form \"<db>.<table>.<column>\", wildcard is supported", f)
		return
	}

	for _, p := range parts {
		p = strings.TrimSpace(p)

		if p == "" {
			err = errors.Wrapf(ErrInvalidField,
				"invalid field: %s, <db>/<table>/<column> parts should not be blanked", f)
			return
		}
	}

	return
}
