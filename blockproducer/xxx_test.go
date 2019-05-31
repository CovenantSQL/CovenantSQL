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

package blockproducer

import (
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	genesisHash               = hash.Hash{}
	testingDataDir            string
	testingConfigFile         = "../test/node_standalone/config.yaml"
	testingPrivateKeyFile     = "../test/node_standalone/private.key"
	testingPublicKeyStoreFile string
	testingNonceDifficulty    int
	testingPrivateKey         *ca.PrivateKey
	testingPublicKey          *ca.PublicKey
)

func setup() {
	var err error
	rand.Seed(time.Now().UnixNano())
	rand.Read(genesisHash[:])

	// Create temp dir for test data
	if testingDataDir, err = ioutil.TempDir("", "CovenantSQL"); err != nil {
		panic(err)
	}

	// Initialze kms
	testingNonceDifficulty = 2
	testingPublicKeyStoreFile = path.Join(testingDataDir, "public.keystore")

	if conf.GConf, err = conf.LoadConfig(testingConfigFile); err != nil {
		panic(err)
	}
	route.InitKMS(testingPublicKeyStoreFile)
	if err = kms.InitLocalKeyPair(testingPrivateKeyFile, []byte{}); err != nil {
		panic(err)
	}
	if testingPrivateKey, err = kms.GetLocalPrivateKey(); err != nil {
		panic(err)
	}
	testingPublicKey = testingPrivateKey.PubKey()

	// Setup logging
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func teardown() {
	if err := os.RemoveAll(testingDataDir); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	os.Exit(func() int {
		setup()
		defer teardown()
		return m.Run()
	}())
}
