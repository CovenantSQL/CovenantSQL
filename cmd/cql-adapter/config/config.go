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

package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	yaml "gopkg.in/yaml.v2"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/cmd/cql-adapter/storage"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	// global config object.
	currentConfig *Config

	// global config lock.
	currentConfigLock sync.Mutex
)

// Config defines adapter specific configuration.
type Config struct {
	// server related
	ListenAddr        string          `yaml:"ListenAddr"`
	CertificatePath   string          `yaml:"CertificatePath"`
	PrivateKeyPath    string          `yaml:"PrivateKeyPath"`
	ServerCertificate tls.Certificate `yaml:"-"`
	TLSConfig         *tls.Config     `yaml:"-"`

	// client related
	VerifyCertificate bool                `yaml:"VerifyCertificate"`
	ClientCAPath      string              `yaml:"ClientCAPath"`
	ClientCertPool    *x509.CertPool      `yaml:"-"`
	AdminCertFiles    []string            `yaml:"AdminCerts"`
	WriteCertFiles    []string            `yaml:"WriteCerts"`
	AdminCertificates []*x509.Certificate `yaml:"-"`
	WriteCertificates []*x509.Certificate `yaml:"-"`

	// storage config
	StorageDriver   string          `yaml:"StorageDriver"` // sqlite3 or covenantsql
	StorageRoot     string          `yaml:"StorageRoot"`
	StorageInstance storage.Storage `yaml:"-"`
}

type confWrapper struct {
	Adapter Config `yaml:"Adapter"`
}

// LoadConfig load and verify config in config file and set to global config instance.
func LoadConfig(configPath string, password string) (config *Config, err error) {
	var workingRoot string
	var configBytes []byte
	if configBytes, err = ioutil.ReadFile(configPath); err != nil {
		log.WithError(err).Error("read config file failed")
	}
	configWrapper := &confWrapper{}
	if err = yaml.Unmarshal(configBytes, configWrapper); err != nil {
		log.WithError(err).Error("unmarshal config file failed")
		return
	}

	config = &configWrapper.Adapter

	if len(config.StorageDriver) == 0 {
		config.StorageDriver = "covenantsql"
	}
	if config.StorageDriver == "covenantsql" {
		// init client
		if err = client.Init(configPath, []byte(password)); err != nil {
			return
		}
		workingRoot = conf.GConf.WorkingRoot
	} else {
		if workingRoot, err = os.Getwd(); err != nil {
			return
		}
	}

	if config.CertificatePath == "" || config.PrivateKeyPath == "" {
		// http mode
		log.Info("running in http mode")
	} else {
		// tls mode
		// init tls config
		config.TLSConfig = &tls.Config{}
		certPath := filepath.Join(workingRoot, config.CertificatePath)
		privateKeyPath := filepath.Join(workingRoot, config.PrivateKeyPath)

		if config.ServerCertificate, err = tls.LoadX509KeyPair(certPath, privateKeyPath); err != nil {
			return
		}

		config.TLSConfig.Certificates = []tls.Certificate{config.ServerCertificate}

		if config.VerifyCertificate && config.ClientCAPath != "" {
			clientCAPath := filepath.Join(workingRoot, config.ClientCAPath)

			// load client CA
			caCertPool := x509.NewCertPool()
			var caCert []byte
			if caCert, err = ioutil.ReadFile(clientCAPath); err != nil {
				return
			}
			caCertPool.AppendCertsFromPEM(caCert)

			config.ClientCertPool = caCertPool
			config.TLSConfig.ClientCAs = caCertPool
			config.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert

			// load admin certs
			config.AdminCertificates = make([]*x509.Certificate, 0)
			for _, certFile := range config.AdminCertFiles {
				certFile = filepath.Join(workingRoot, certFile)

				var cert *x509.Certificate
				if cert, err = loadCert(certFile); err != nil {
					return
				}

				config.AdminCertificates = append(config.AdminCertificates, cert)
			}

			// load write certs
			config.WriteCertificates = make([]*x509.Certificate, 0)
			for _, certFile := range config.WriteCertFiles {
				certFile = filepath.Join(workingRoot, certFile)

				var cert *x509.Certificate
				if cert, err = loadCert(certFile); err != nil {
					return
				}

				config.WriteCertificates = append(config.WriteCertificates, cert)
			}

		} else {
			config.TLSConfig.ClientAuth = tls.NoClientCert
		}
	}

	// load storage
	switch config.StorageDriver {
	case "covenantsql":
		config.StorageInstance = storage.NewCovenantSQLStorage()
	case "sqlite3":
		storageRoot := filepath.Join(workingRoot, config.StorageRoot)
		if config.StorageInstance, err = storage.NewSQLite3Storage(storageRoot); err != nil {
			return
		}
	default:
		err = ErrInvalidStorageConfig
		return
	}

	currentConfigLock.Lock()
	currentConfig = config
	currentConfigLock.Unlock()

	return
}

// GetConfig returns global initialized config.
func GetConfig() *Config {
	currentConfigLock.Lock()
	defer currentConfigLock.Unlock()

	return currentConfig
}

func loadCert(pemFile string) (cert *x509.Certificate, err error) {
	// only the first pem section is parsed and identified as certificate.
	var certBytes []byte

	if certBytes, err = ioutil.ReadFile(pemFile); err != nil {
		return
	}

	var pemBlock *pem.Block
	if pemBlock, _ = pem.Decode(certBytes); pemBlock == nil {
		err = ErrInvalidCertificateFile
		return
	}

	if pemBlock.Type != "CERTIFICATE" || len(pemBlock.Headers) != 0 {
		err = ErrInvalidCertificateFile
		return
	}

	return x509.ParseCertificate(pemBlock.Bytes)
}
