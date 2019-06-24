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

package main

import (
	"context"
	"errors"
	"flag"
	"io/ioutil"
	"math/rand"
	"time"

	validator "gopkg.in/go-playground/validator.v9"
	yaml "gopkg.in/yaml.v2"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	configFile string
	logLevel   string
)

func init() {
	flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file path")
	flag.StringVar(&logLevel, "log-level", "", "Log level")
}

func main() {
	flag.Parse()
	log.SetStringLevel(logLevel, log.InfoLevel)
	configFile = utils.HomeDirExpand(configFile)
	rand.Seed(time.Now().UnixNano())

	cfg, err := loadConfig()
	if err != nil {
		return
	}

	xchg, err := NewExchange(cfg)
	if err != nil {
		return
	}

	err = xchg.Start(context.Background())
	if err != nil {
		return
	}

	<-utils.WaitForExit()

	xchg.Stop()
}

func loadConfig() (cfg *ExchangeConfig, err error) {
	_, err = conf.LoadConfig(configFile)
	if err != nil {
		log.WithError(err).Error("read cql config failed")
		return
	}

	var configBytes []byte
	if configBytes, err = ioutil.ReadFile(configFile); err != nil {
		log.WithError(err).Error("read config file failed")
		return
	}

	r := &struct {
		Exchange *ExchangeConfig `json:"Exchange,omitempty" yaml:"Exchange,omitempty"`
	}{}
	if err = yaml.Unmarshal(configBytes, r); err != nil {
		log.WithError(err).Error("unmarshal config file failed")
		return
	}

	if r.Exchange == nil {
		err = errors.New("nil exchange config")
		log.Error("could not read exchange config")
		return
	}

	validate := validator.New()
	if err = validate.Struct(*r.Exchange); err != nil {
		log.WithError(err).Error("validate config failed")
		return
	}

	cfg = r.Exchange
	return
}
