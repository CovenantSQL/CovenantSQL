// +build !testbinary

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
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sync"
	"syscall"
	"testing"
	"time"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/jmoiron/jsonq"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	baseDir        = utils.GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/")
	logDir         = FJ(testWorkingDir, "./log/")
)

var nodeCmds []*utils.CMD

var FJ = filepath.Join

func privKeyStoreToAccountAddr(
	path string, master []byte) (priv *asymmetric.PrivateKey, addr proto.AccountAddress, err error,
) {
	if priv, err = kms.LoadPrivateKey(path, master); err != nil {
		return
	}
	addr, err = crypto.PubKeyHash(priv.PubKey())
	return
}

func startNodes() {
	// wait for ports to be available
	var err error
	ctx := context.Background()
	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		4120,
		4121,
		4122,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	// start 3bps
	var cmd *utils.CMD
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./observation/node_0/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-observer/leader.cover.out"),
		},
		"leader", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./observation/node_1/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-observer/follower1.cover.out"),
		},
		"follower1", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./observation/node_2/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-observer/follower2.cover.out"),
		},
		"follower2", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	err = utils.WaitToConnect(ctx, "127.0.0.1", []int{
		4120,
		4121,
		4122,
	}, time.Second)
	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		4144,
		4145,
		4146,
	}, time.Millisecond*200)
	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	time.Sleep(10 * time.Second)
	// start 3miners
	os.RemoveAll(FJ(testWorkingDir, "./observation/node_miner_0/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./observation/node_miner_0/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-observer/miner0.cover.out"),
		},
		"miner0", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	os.RemoveAll(FJ(testWorkingDir, "./observation/node_miner_1/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./observation/node_miner_1/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-observer/miner1.cover.out"),
		},
		"miner1", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	os.RemoveAll(FJ(testWorkingDir, "./observation/node_miner_2/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./observation/node_miner_2/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-observer/miner2.cover.out"),
		},
		"miner2", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
}

func stopNodes() {
	var wg sync.WaitGroup

	for _, nodeCmd := range nodeCmds {
		wg.Add(1)
		go func(thisCmd *utils.CMD) {
			defer wg.Done()
			thisCmd.Cmd.Process.Signal(syscall.SIGTERM)
			thisCmd.Cmd.Wait()
			grepRace := exec.Command("/bin/sh", "-c", "grep -a -A 50 'DATA RACE' "+thisCmd.LogPath)
			out, _ := grepRace.Output()
			if len(out) > 2 {
				log.Fatal(string(out))
			}
		}(nodeCmd)
	}

	wg.Wait()
}

func getJSON(pattern string, args ...interface{}) (result *jsonq.JsonQuery, err error) {
	url := "http://localhost:4663/" + fmt.Sprintf(pattern, args...)
	resp, err := http.Get(url)
	if err != nil {
		return
	}

	var res map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return
	}
	log.WithFields(log.Fields{
		"pattern":  pattern,
		"args":     args,
		"response": res,
		"code":     resp.StatusCode,
	}).Debug("send test request")
	result = jsonq.NewQuery(res)
	success, err := result.Bool("success")
	if err != nil {
		return
	}
	if !success {
		var status string
		status, err = result.String("status")
		if err != nil {
			return
		}
		err = errors.New(status)
		return
	}
	result = jsonq.NewQuery(ensureSuccess(result.Interface("data")))

	return
}

func ensureSuccess(v interface{}, err error) interface{} {
	if err != nil {
		debug.PrintStack()
	}
	So(err, ShouldBeNil)
	return v
}

func TestFullProcess(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	Convey("test full process", t, func() {
		var (
			err              error
			cliPriv          *asymmetric.PrivateKey
			addr, addr2      proto.AccountAddress
			dsn, dsn2        string
			cfg, cfg2        *client.Config
			dbID, dbID2      string
			ctx1, ctx2, ctx3 context.Context
			ccl1, ccl2, ccl3 context.CancelFunc
		)
		startNodes()
		defer stopNodes()

		err = client.Init(FJ(testWorkingDir, "./observation/node_c/config.yaml"), []byte(""))
		So(err, ShouldBeNil)

		// get miner addresses
		cliPriv, _, err = privKeyStoreToAccountAddr(
			FJ(testWorkingDir, "./observation/node_c/private.key"), []byte{})
		So(err, ShouldBeNil)
		_, addr, err = privKeyStoreToAccountAddr(
			FJ(testWorkingDir, "./observation/node_miner_0/private.key"), []byte{})
		So(err, ShouldBeNil)
		_, addr2, err = privKeyStoreToAccountAddr(
			FJ(testWorkingDir, "./observation/node_miner_1/private.key"), []byte{})
		So(err, ShouldBeNil)

		// wait until bp chain service is ready
		ctx1, ccl1 = context.WithTimeout(context.Background(), 1*time.Minute)
		defer ccl1()
		err = bp.WaitBPChainService(ctx1, 3*time.Second)
		So(err, ShouldBeNil)

		// create
		_, dsn, err = bp.Create(types.ResourceMeta{
			TargetMiners: []proto.AccountAddress{addr},
			Node:         1,
		}, 1, 10000000, cliPriv)
		So(err, ShouldBeNil)
		log.Infof("the created database dsn is %v", dsn)

		db, err := sql.Open("covenantsql", dsn)
		So(err, ShouldBeNil)

		// wait
		cfg, err = client.ParseDSN(dsn)
		So(err, ShouldBeNil)
		dbID = cfg.DatabaseID
		ctx2, ccl2 = context.WithTimeout(context.Background(), 5*time.Minute)
		defer ccl2()
		err = bp.WaitDatabaseCreation(ctx2, proto.DatabaseID(dbID), db, 3*time.Second)
		So(err, ShouldBeNil)

		_, err = db.Exec("CREATE TABLE test (test int)")
		So(err, ShouldBeNil)

		_, err = db.Exec("INSERT INTO test VALUES(?)", 4)
		So(err, ShouldBeNil)

		row := db.QueryRow("SELECT * FROM test LIMIT 1")

		var result int
		err = row.Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldEqual, 4)

		// test timestamp fields
		_, err = db.Exec("CREATE TABLE test_time (test timestamp)")
		So(err, ShouldBeNil)

		_, err = db.Exec("INSERT INTO test_time VALUES(DATE('NOW'))")
		So(err, ShouldBeNil)

		row = db.QueryRow("SELECT * FROM test_time LIMIT 1")

		var tmResult time.Time
		err = row.Scan(&tmResult)
		So(err, ShouldBeNil)
		So(tmResult, ShouldHappenBefore, time.Now())

		// test string fields
		row = db.QueryRow("SELECT name FROM sqlite_master WHERE type = ? LIMIT 1", "table")
		var resultString string
		err = row.Scan(&resultString)
		So(err, ShouldBeNil)
		So(resultString, ShouldBeIn, []string{"test", "test_time"})

		// try raw bytes
		_, err = db.Exec("CREATE TABLE test_raw (test blob)")
		So(err, ShouldBeNil)

		_, err = db.Exec("INSERT INTO test_raw VALUES(?)", []byte("ha\001ppy"))
		So(err, ShouldBeNil)

		row = db.QueryRow("SELECT * FROM test_raw LIMIT 1")
		var resultBytes []byte
		err = row.Scan(&resultBytes)
		So(err, ShouldBeNil)
		So(resultBytes, ShouldResemble, []byte("ha\001ppy"))

		err = db.Close()
		So(err, ShouldBeNil)

		// create
		_, dsn2, err = bp.Create(types.ResourceMeta{
			TargetMiners: []proto.AccountAddress{addr2},
			Node:         1,
		}, 1, 10000000, cliPriv)
		So(err, ShouldBeNil)

		log.Infof("the created database dsn is %v", dsn2)

		db2, err := sql.Open("covenantsql", dsn2)
		So(err, ShouldBeNil)

		// wait
		cfg2, err = client.ParseDSN(dsn2)
		So(err, ShouldBeNil)
		dbID2 = cfg2.DatabaseID
		So(dbID, ShouldNotResemble, dbID2)
		ctx3, ccl3 = context.WithTimeout(context.Background(), 5*time.Minute)
		defer ccl3()
		err = bp.WaitDatabaseCreation(ctx3, proto.DatabaseID(dbID2), db2, 3*time.Second)
		So(err, ShouldBeNil)

		_, err = db2.Exec("CREATE TABLE test (test int)")
		So(err, ShouldBeNil)

		_, err = db2.Exec("INSERT INTO test VALUES(?)", 4)
		So(err, ShouldBeNil)

		row2 := db2.QueryRow("SELECT * FROM test LIMIT 1")

		var result2 int
		err = row2.Scan(&result2)
		So(err, ShouldBeNil)
		So(result2, ShouldEqual, 4)

		err = db2.Close()
		So(err, ShouldBeNil)

		// start the observer and listen for produced blocks
		err = utils.WaitForPorts(context.Background(), "127.0.0.1", []int{4663}, time.Millisecond*200)
		So(err, ShouldBeNil)

		// remove previous observation result
		os.Remove(FJ(testWorkingDir, "./observation/node_observer/observer.db"))

		var observerCmd *utils.CMD
		observerCmd, err = utils.RunCommandNB(
			FJ(baseDir, "./bin/cql-observer.test"),
			[]string{"-config", FJ(testWorkingDir, "./observation/node_observer/config.yaml"),
				"-database", dbID, "-reset", "oldest",
				"-test.coverprofile", FJ(baseDir, "./cmd/cql-observer/observer.cover.out"),
			},
			"observer", testWorkingDir, logDir, false,
		)
		So(err, ShouldBeNil)

		defer func() {
			observerCmd.Cmd.Process.Signal(os.Interrupt)
			observerCmd.Cmd.Wait()
		}()

		// wait for the observer to collect blocks, two periods is enough
		time.Sleep(conf.GConf.SQLChainPeriod * 5)

		// test get genesis block by height
		res, err := getJSON("v1/height/%v/0", dbID)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.Interface("block")), ShouldNotBeNil)
		So(ensureSuccess(res.Int("block", "height")), ShouldEqual, 0)
		genesisHash := ensureSuccess(res.String("block", "hash")).(string)

		res, err = getJSON("v1/head/%v", dbID)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.Interface("block")), ShouldNotBeNil)
		maxHeight := ensureSuccess(res.Int("block", "height")).(int)
		So(maxHeight, ShouldBeGreaterThan, 0)

		// test get first containable block
		var (
			blockHash           string
			byHeightBlockResult interface{}
		)

		// access from max height to found a non-empty block
		for i := maxHeight; i > 0; i-- {
			res, err = getJSON("v3/height/%v/%d", dbID, i)
			So(err, ShouldBeNil)
			So(ensureSuccess(res.Interface("block")), ShouldNotBeNil)
			So(ensureSuccess(res.Int("block", "height")), ShouldEqual, i)
			So(ensureSuccess(res.String("block", "hash")), ShouldNotBeEmpty)
			So(ensureSuccess(res.String("block", "genesis_hash")), ShouldEqual, genesisHash)
			if len(ensureSuccess(res.ArrayOfObjects("block", "queries")).([]map[string]interface{})) == 0 {
				// got empty block
				log.WithField("block", res).Debugf("got empty block, try next index")
				continue
			}
			So(ensureSuccess(res.ArrayOfObjects("block", "queries")), ShouldNotBeEmpty)
			blockHash = ensureSuccess(res.String("block", "hash")).(string)
			byHeightBlockResult = ensureSuccess(res.Interface())
			break
		}

		// test get block by hash
		res, err = getJSON("v3/block/%v/%v?size=1000", dbID, blockHash)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.ArrayOfObjects("block", "queries")), ShouldResemble,
			ensureSuccess(jsonq.NewQuery(byHeightBlockResult).ArrayOfObjects("block", "queries")))

		// test get block by hash v3 with pagination
		res, err = getJSON("v3/block/%v/%v?page=10000&size=10", dbID, blockHash)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.ArrayOfObjects("block", "queries")), ShouldBeEmpty)

		// test get block with page size = 1
		res, err = getJSON("v3/block/%v/%v?page=1&size=1", dbID, blockHash)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.ArrayOfObjects("block", "queries")), ShouldHaveLength, 1)

		// test get block with page size = 2
		res, err = getJSON("v3/block/%v/%v?page=1&size=2", dbID, blockHash)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.ArrayOfObjects("block", "queries")), ShouldHaveLength, 2)

		// test get block with page size = 1, page = 2
		res, err = getJSON("v3/block/%v/%v?page=2&size=1", dbID, blockHash)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.ArrayOfObjects("block", "queries")), ShouldHaveLength, 1)

		// test get block by hash using v1 version, returns ack hashes as queries
		res, err = getJSON("v1/block/%v/%v", dbID, blockHash)
		So(err, ShouldBeNil)

		ackHashes, err := res.ArrayOfStrings("block", "queries")
		So(err, ShouldBeNil)
		So(ackHashes, ShouldNotBeEmpty)

		// test get acked query in block
		var reqHash string

		for _, ackHash := range ackHashes {
			res, err = getJSON("v1/ack/%v/%v", dbID, ackHash)
			So(err, ShouldBeNil)
			So(ensureSuccess(res.Interface("ack")), ShouldNotBeNil)
			So(ensureSuccess(res.String("ack", "hash")), ShouldNotBeEmpty)
			So(ensureSuccess(res.String("ack", "request", "hash")), ShouldNotBeEmpty)
			So(ensureSuccess(res.String("ack", "response", "hash")), ShouldNotBeEmpty)

			queryType, err := res.String("ack", "request", "type")
			So(err, ShouldBeNil)
			So(queryType, ShouldBeIn, []string{types.WriteQuery.String(), types.ReadQuery.String()})

			if queryType == types.WriteQuery.String() {
				reqHash, err = res.String("ack", "request", "hash")
				So(err, ShouldBeNil)
				So(reqHash, ShouldNotBeEmpty)
			}
		}

		// must contains a write query
		So(reqHash, ShouldNotBeEmpty)

		// test get request entity by request hash
		res, err = getJSON("v1/request/%v/%v", dbID, reqHash)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.Interface("request")), ShouldNotBeNil)
		So(ensureSuccess(res.String("request", "hash")), ShouldNotBeEmpty)
		So(ensureSuccess(res.String("request", "type")), ShouldEqual, types.WriteQuery.String())
		So(ensureSuccess(res.Int("request", "count")), ShouldEqual, 1) // no transaction batch is used
		So(ensureSuccess(res.ArrayOfObjects("request", "queries")), ShouldNotBeEmpty)
		So(ensureSuccess(res.String("request", "queries", "0", "pattern")), ShouldNotBeEmpty)

		// test get genesis block by height
		res, err = getJSON("v3/height/%v/0", dbID2)
		So(err, ShouldNotBeNil)
		log.Info(err, res)

		// test get genesis block by height
		res, err = getJSON("v3/head/%v", dbID2)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.Interface("block")), ShouldNotBeNil)
		So(ensureSuccess(res.Int("block", "height")), ShouldEqual, 0)
		log.Info(err, res)

		err = client.Drop(dsn)
		So(err, ShouldBeNil)

		err = client.Drop(dsn2)
		So(err, ShouldBeNil)
	})
}
