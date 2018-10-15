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
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"gopkg.in/yaml.v2"
)

type adapterConfig struct {
	ListenAddr        string
	CertificatePath   string
	PrivateKeyPath    string
	VerifyCertificate bool
	ClientCAPath      string
	AdminCerts        []string
	WriteCerts        []string
	StorageDriver     string
	StorageRoot       string
}

var (
	defaultAdapterConfig = &adapterConfig{
		ListenAddr:        "0.0.0.0:4661",
		CertificatePath:   "server.pem",
		PrivateKeyPath:    "server-key.pem",
		VerifyCertificate: false,
		ClientCAPath:      "",
		AdminCerts:        []string{},
		WriteCerts:        []string{},
		StorageDriver:     "covenantsql",
		StorageRoot:       "",
	}
)

func (c *adapterConfig) readListenAddr() {

}

func (c *adapterConfig) readCertificatePath() {

}

func (c *adapterConfig) readPrivateKeyPath() {

}

func (c *adapterConfig) readVerifyCertificate() {

}

func (c *adapterConfig) readClientCAPath() {

}

func (c *adapterConfig) readAdminCerts() {

}

func (c *adapterConfig) readWriteCerts() {

}

func (c *adapterConfig) readStorageDriver() {

}

func (c *adapterConfig) readStorageRoot() {

}

func (c *adapterConfig) loadFromExistingConfig(rawConfig map[string]interface{}) {
	if rawConfig == nil || rawConfig["Adapter"] == nil {
		return
	}

	// fill values to config structure
	var configBytes []byte
	var err error

	if configBytes, err = json.Marshal(rawConfig["Adapter"]); err != nil {
		log.Warningf("load previous adapter config failed: %v", err)
		return
	}

	if err = json.Unmarshal(configBytes, c); err != nil {
		log.Warningf("load previous adapter config failed: %v", err)
		return
	}

	return
}

func (c *adapterConfig) writeToRawConfig(rawConfig map[string]interface{}) {
	if rawConfig != nil {
		rawConfig["Adapter"] = c
	}
}

func (c *adapterConfig) readAllConfig() {

}

func readDataFromStdin(prompt string) (s string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	s, _ = reader.ReadString('\n')
	return
}

func runAdapterConfGen() {
	if configFile == "" {
		log.Error("config file path is required for adapterconfgen tool")
		os.Exit(1)
	}

	var err error
	var configBytes []byte
	if configBytes, err = ioutil.ReadFile(configFile); err != nil {
		log.Errorf("an existing config file is required for adapterconfggen: %v", err)
		os.Exit(1)
	}

	// load config
	var rawConfig map[string]interface{}
	if err = yaml.Unmarshal(configBytes, &rawConfig); err != nil {
		log.Errorf("a valid config file is required for adapterconfgen: %v", err)
		os.Exit(1)
	}

	if rawConfig == nil {
		log.Errorf("a confgen generated config is required for adapterconfgen: %v", err)
		os.Exit(1)
	}

	// backup config
	bakConfigFile := configFile + ".bak"
	if err = ioutil.WriteFile(bakConfigFile, configBytes, 0600); err != nil {
		log.Errorf("backup config file failed: %v", err)
		os.Exit(1)
	}

	defaultAdapterConfig.loadFromExistingConfig(rawConfig)
	defaultAdapterConfig.readAllConfig()
	defaultAdapterConfig.writeToRawConfig(rawConfig)

	if configBytes, err = yaml.Marshal(rawConfig); err != nil {
		log.Errorf("marshal config failed: %v", err)
		os.Exit(1)
	}

	if err = ioutil.WriteFile(configFile, configBytes, 0600); err != nil {
		log.Errorf("write config to file failed: %v", err)
		os.Exit(1)
	}

	log.Infof("adapter config generated")

	return
}
