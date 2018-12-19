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
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	yaml "gopkg.in/yaml.v2"
)

type adapterConfig struct {
	ListenAddr        string   `yaml:"ListenAddr"`
	CertificatePath   string   `yaml:"CertificatePath"`
	PrivateKeyPath    string   `yaml:"PrivateKeyPath"`
	VerifyCertificate bool     `yaml:"VerifyCertificate"`
	ClientCAPath      string   `yaml:"ClientCAPath"`
	AdminCerts        []string `yaml:"AdminCerts"`
	WriteCerts        []string `yaml:"WriteCerts"`
	StorageDriver     string   `yaml:"StorageDriver"`
	StorageRoot       string   `yaml:"StorageRoot"`
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
	newAddr := readDataFromStdin("ListenAddr (default: %v): ", c.ListenAddr)
	if newAddr != "" {
		c.ListenAddr = newAddr
	}
}

func (c *adapterConfig) readCertificatePath() {
	newCertPath := readDataFromStdin("CertificatePath (default: %v): ", c.CertificatePath)
	if newCertPath != "" {
		c.CertificatePath = newCertPath
	}
}

func (c *adapterConfig) readPrivateKeyPath() {
	newPrivateKeyPath := readDataFromStdin("PrivateKeyPath (default: %v): ", c.PrivateKeyPath)
	if newPrivateKeyPath != "" {
		c.PrivateKeyPath = newPrivateKeyPath
	}
}

func (c *adapterConfig) readVerifyCertificate() bool {
	shouldVerifyCertificate := readDataFromStdin(
		"VerifyCertificate (default: %v) (y/n): ", c.VerifyCertificate)
	if shouldVerifyCertificate != "" {
		switch shouldVerifyCertificate {
		case "y":
			fallthrough
		case "Y":
			c.VerifyCertificate = true
		case "N":
			fallthrough
		case "n":
			c.VerifyCertificate = false
		}
	}

	return c.VerifyCertificate
}

func (c *adapterConfig) readClientCAPath() {
	newClientCAPath := readDataFromStdin("ClientCAPath (default: %v)", c.ClientCAPath)
	if newClientCAPath != "" {
		c.ClientCAPath = newClientCAPath
	}
}

func (c *adapterConfig) readAdminCerts() {
	var newCerts []string

	for {
		record := readDataFromStdin("AdminCerts: ")

		if record == "" {
			break
		}

		newCerts = append(newCerts, record)
	}

	c.AdminCerts = newCerts
}

func (c *adapterConfig) readWriteCerts() {
	var newCerts []string

	for {
		record := readDataFromStdin("WriteCerts: ")

		if record == "" {
			break
		}

		newCerts = append(newCerts, record)
	}

	c.WriteCerts = newCerts
}

func (c *adapterConfig) readStorageDriver() {
	newStorageDriver := readDataFromStdin("StorageDriver (default: %v)", c.StorageDriver)
	if newStorageDriver != "" {
		c.StorageDriver = newStorageDriver
	}
}

func (c *adapterConfig) readStorageRoot() {
	newStorageRoot := readDataFromStdin("StorageRoot (default: %v)", c.StorageRoot)
	if newStorageRoot != "" {
		c.StorageRoot = newStorageRoot
	}
}

func (c *adapterConfig) loadFromExistingConfig(rawConfig yaml.MapSlice) {
	if rawConfig == nil {
		return
	}

	// find adapter config in map slice
	var originalConfig interface{}

	for _, item := range rawConfig {
		if keyStr, ok := item.Key.(string); ok && keyStr == "Adapter" {
			originalConfig = item.Value
		}
	}

	if originalConfig == nil {
		return
	}

	// fill values to config structure
	var configBytes []byte
	var err error

	if configBytes, err = yaml.Marshal(originalConfig); err != nil {
		log.WithError(err).Warning("load previous adapter config failed")
		return
	}

	if err = yaml.Unmarshal(configBytes, c); err != nil {
		log.WithError(err).Warning("load previous adapter config failed")
		return
	}

	return
}

func (c *adapterConfig) writeToRawConfig(rawConfig yaml.MapSlice) yaml.MapSlice {
	if rawConfig == nil {
		return rawConfig
	}

	// find adapter config in map slice
	for i, item := range rawConfig {
		if keyStr, ok := item.Key.(string); ok && keyStr == "Adapter" {
			rawConfig[i].Value = c
			return rawConfig
		}
	}

	// not found
	rawConfig = append(rawConfig, yaml.MapItem{
		Key:   "Adapter",
		Value: c,
	})

	return rawConfig
}

func (c *adapterConfig) readAllConfig() {
	c.readListenAddr()
	c.readCertificatePath()
	c.readPrivateKeyPath()

	if c.readVerifyCertificate() {
		c.readClientCAPath()
		c.readAdminCerts()
		c.readWriteCerts()
	}

	c.readStorageDriver()
	c.readStorageRoot()
}

func readDataFromStdin(prompt string, values ...interface{}) (s string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(prompt, values...)
	s, _ = reader.ReadString('\n')
	s = strings.TrimSpace(s)
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
		log.WithError(err).Error("an existing config file is required for adapterconfggen")
		os.Exit(1)
	}

	// load config
	var rawConfig yaml.MapSlice
	if err = yaml.Unmarshal(configBytes, &rawConfig); err != nil {
		log.WithError(err).Error("a valid config file is required for adapterconfgen")
		os.Exit(1)
	}

	if rawConfig == nil {
		log.WithError(err).Error("a confgen generated config is required for adapterconfgen")
		os.Exit(1)
	}

	// backup config
	bakConfigFile := configFile + ".bak"
	if err = ioutil.WriteFile(bakConfigFile, configBytes, 0600); err != nil {
		log.WithError(err).Error("backup config file failed")
		os.Exit(1)
	}

	defaultAdapterConfig.loadFromExistingConfig(rawConfig)
	defaultAdapterConfig.readAllConfig()
	rawConfig = defaultAdapterConfig.writeToRawConfig(rawConfig)

	if configBytes, err = yaml.Marshal(rawConfig); err != nil {
		log.WithError(err).Error("marshal config failed")
		os.Exit(1)
	}

	if err = ioutil.WriteFile(configFile, configBytes, 0600); err != nil {
		log.WithError(err).Error("write config to file failed")
		os.Exit(1)
	}

	log.Info("adapter config generated")

	return
}
