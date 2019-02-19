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

package observer

import (
	"io/ioutil"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	yaml "gopkg.in/yaml.v2"
)

// Database defines single database subscription status.
type Database struct {
	ID       string `yaml:"ID"`
	Position string `yaml:"Position"`
}

// Config defines subscription settings for observer.
type Config struct {
	Databases []Database `yaml:"Databases"`
}

type configWrapper struct {
	Observer *Config `yaml:"Observer"`
}

func loadConfig(path string) (config *Config, err error) {
	var (
		content []byte
		wrapper = &configWrapper{}
	)
	if content, err = ioutil.ReadFile(path); err != nil {
		log.WithError(err).Error("failed to read config file")
		return
	}
	if err = yaml.Unmarshal(content, wrapper); err != nil {
		log.WithError(err).Error("failed to unmarshal config file")
		return
	}
	config = wrapper.Observer
	return
}
