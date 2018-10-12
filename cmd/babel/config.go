package main

import (
	"io/ioutil"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"gopkg.in/yaml.v2"
)

// BabelConfig includes the config information to interact with ethereum contract.
type BabelConfig struct {
	// Ethereum endpoint
	Endpoint string `yaml:"Endpoint"`

	// contract info
	ContractAddress string `yaml:"ContractAddress"`
	ContractAbiFile string `yaml:"ContractABIFile"`

	// filter
	EventSignature string `yaml:"EventSignature"`

	// db path
	BabelDB string `yaml:"BabelDB"`
}

// LoadConfig loads config from configPath.
func LoadConfig(configPath string) (config *BabelConfig, err error) {
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Errorf("read config file failed: %s", err)
		return
	}
	config = &BabelConfig{}
	err = yaml.Unmarshal(configBytes, config)
	if err != nil {
		log.Errorf("unmarshal config file failed: %s", err)
		return
	}

	return
}
