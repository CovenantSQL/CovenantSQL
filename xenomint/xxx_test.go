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

package xenomint

import (
	"database/sql"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"sync"
	"syscall"
	"testing"
	"time"

	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	pc "github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

var (
	testingDataDir            string
	testingDHTDBFile          string
	testingPrivateKeyFile     string
	testingPublicKeyStoreFile string
	testingNonceDifficulty    int

	testingMasterKey = []byte(`?08Rl%WUih4V0H+c`)
)

func buildQuery(query string, args ...interface{}) wt.Query {
	var nargs = make([]sql.NamedArg, len(args))
	for i := range args {
		nargs[i] = sql.NamedArg{
			Name:  "",
			Value: args[i],
		}
	}
	return wt.Query{
		Pattern: query,
		Args:    nargs,
	}
}

func buildRequest(qt wt.QueryType, qs []wt.Query) *wt.Request {
	return &wt.Request{
		Header: wt.SignedRequestHeader{
			RequestHeader: wt.RequestHeader{
				Timestamp: time.Now().UTC(),
				QueryType: qt,
			},
		},
		Payload: wt.RequestPayload{Queries: qs},
	}
}

func concat(args [][]interface{}) (ret []interface{}) {
	var (
		tlen int
	)
	for _, v := range args {
		tlen += len(v)
	}
	ret = make([]interface{}, 0, tlen)
	for _, v := range args {
		ret = append(ret, v...)
	}
	return
}

func mineNoncesFromPublicKey(
	pub *ca.PublicKey, diff int, num int) (nis []proto.Node, err error,
) {
	var (
		nic   = make(chan pc.NonceInfo)
		block = pc.MiningBlock{Data: pub.Serialize(), NonceChan: nic, Stop: nil}
		miner = pc.NewCPUMiner(nil)
		wg    = &sync.WaitGroup{}

		next pc.Uint256
		ni   pc.NonceInfo
	)

	defer func() {
		wg.Wait()
		close(nic)
	}()

	nis = make([]proto.Node, num)
	for i := range nis {
		wg.Add(1)
		go func() {
			defer wg.Done()
			miner.ComputeBlockNonce(block, next, diff)
		}()
		ni = <-nic
		nis[i] = proto.Node{
			ID:        proto.NodeID(ni.Hash.String()),
			Nonce:     ni.Nonce,
			PublicKey: pub,
		}
		next = ni.Nonce
		next.Inc()
	}

	return
}

func setup() {
	const minNoFile uint64 = 4096
	var (
		err error
		lmt syscall.Rlimit
	)

	if testingDataDir, err = ioutil.TempDir("", "CovenantSQL"); err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())

	// Set NOFILE limit
	if err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lmt); err != nil {
		panic(err)
	}
	if lmt.Max < minNoFile {
		panic("insufficient max RLIMIT_NOFILE")
	}
	lmt.Cur = lmt.Max
	if err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &lmt); err != nil {
		panic(err)
	}

	// Initialze kms
	var (
		priv *ca.PrivateKey
		pub  *ca.PublicKey
	)
	testingNonceDifficulty = 2
	testingDHTDBFile = path.Join(testingDataDir, "dht.db")
	testingPrivateKeyFile = path.Join(testingDataDir, "private.key")
	testingPublicKeyStoreFile = path.Join(testingDataDir, "public.keystore")
	if err = kms.InitPublicKeyStore(testingPublicKeyStoreFile, nil); err != nil {
		panic(err)
	}
	if priv, pub, err = ca.GenSecp256k1KeyPair(); err != nil {
		panic(err)
	}
	kms.Unittest = true
	kms.SetLocalKeyPair(priv, pub)
	if err = kms.SavePrivateKey(testingPrivateKeyFile, priv, testingMasterKey); err != nil {
		panic(err)
	}

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
