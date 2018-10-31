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

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
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

func TestBuild(t *testing.T) {
	Convey("build", t, func() {
		log.SetLevel(log.DebugLevel)
		So(utils.Build(), ShouldBeNil)
	})
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

	utils.CleanupDB()

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
	}, time.Millisecond*200)
	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	time.Sleep(10 * time.Second)
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		4144,
		4145,
		4146,
	}, time.Millisecond*200)
	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	// start 3miners
	os.RemoveAll(FJ(testWorkingDir, "./observation/node_miner_0/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./observation/node_miner_0/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-observer/miner0.cover.out"),
		},
		"miner0", testWorkingDir, logDir, true,
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
		"miner1", testWorkingDir, logDir, true,
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
		"miner2", testWorkingDir, logDir, true,
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
			grepRace := exec.Command("/bin/sh", "-c", "grep -A 50 'DATA RACE' "+thisCmd.LogPath)
			out, _ := grepRace.Output()
			if len(out) > 2 {
				log.Fatal(string(out))
			}
		}(nodeCmd)
	}

	wg.Wait()
}

func getJSON(pattern string, args ...interface{}) (result *jsonq.JsonQuery, err error) {
	url := "http://localhost:4663/v1/" + fmt.Sprintf(pattern, args...)
	resp, err := http.Get(url)
	if err != nil {
		return
	}

	var res map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&res)
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
		startNodes()
		defer stopNodes()
		time.Sleep(10 * time.Second)

		var err error
		err = client.Init(FJ(testWorkingDir, "./observation/node_c/config.yaml"), []byte(""))
		So(err, ShouldBeNil)

		// create
		dsn, err := client.Create(client.ResourceMeta{Node: 1})
		So(err, ShouldBeNil)

		log.Infof("the created database dsn is %v", dsn)

		db, err := sql.Open("covenantsql", dsn)
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
		dsn2, err := client.Create(client.ResourceMeta{Node: 1})
		So(err, ShouldBeNil)

		log.Infof("the created database dsn is %v", dsn2)

		db2, err := sql.Open("covenantsql", dsn2)
		So(err, ShouldBeNil)

		cfg2, err := client.ParseDSN(dsn2)
		dbID2 := cfg2.DatabaseID
		So(dbID, ShouldNotResemble, dbID2)

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

		cfg, err := client.ParseDSN(dsn)
		dbID := cfg.DatabaseID

		// remove previous observation result
		os.Remove(FJ(testWorkingDir, "./observation/node_observer/observer.db"))

		var observerCmd *utils.CMD
		observerCmd, err = utils.RunCommandNB(
			FJ(baseDir, "./bin/cql-observer.test"),
			[]string{"-config", FJ(testWorkingDir, "./observation/node_observer/config.yaml"),
				"-database", dbID, "-reset", "oldest",
				"-test.coverprofile", FJ(baseDir, "./cmd/cql-observer/observer.cover.out"),
			},
			"observer", testWorkingDir, logDir, true,
		)
		So(err, ShouldBeNil)

		defer func() {
			observerCmd.Cmd.Process.Signal(os.Interrupt)
			observerCmd.Cmd.Wait()
		}()

		// wait for the observer to collect blocks, two periods is enough
		time.Sleep(blockProducePeriod * 2)

		// test get genesis block by height
		res, err := getJSON("height/%v/0", dbID)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.Interface("block")), ShouldNotBeNil)
		So(ensureSuccess(res.Int("block", "height")), ShouldEqual, 0)
		genesisHash := ensureSuccess(res.String("block", "hash")).(string)

		// test get first containable block
		res, err = getJSON("height/%v/1", dbID)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.Interface("block")), ShouldNotBeNil)
		So(ensureSuccess(res.Int("block", "height")), ShouldEqual, 1)
		So(ensureSuccess(res.String("block", "hash")), ShouldNotBeEmpty)
		So(ensureSuccess(res.String("block", "genesis_hash")), ShouldEqual, genesisHash)
		So(ensureSuccess(res.ArrayOfStrings("block", "queries")), ShouldNotBeEmpty)
		blockHash := ensureSuccess(res.String("block", "hash")).(string)
		byHeightBlockResult := ensureSuccess(res.Interface())

		// test get block by hash
		res, err = getJSON("block/%v/%v", dbID, blockHash)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.Interface()), ShouldResemble, byHeightBlockResult)

		ackHashes, err := res.ArrayOfStrings("block", "queries")
		So(err, ShouldBeNil)
		So(ackHashes, ShouldNotBeEmpty)

		// test get acked query in block
		var logOffset int
		var reqHash string

		for _, ackHash := range ackHashes {
			res, err = getJSON("ack/%v/%v", dbID, ackHash)
			So(err, ShouldBeNil)
			So(ensureSuccess(res.Interface("ack")), ShouldNotBeNil)
			So(ensureSuccess(res.String("ack", "hash")), ShouldNotBeEmpty)
			So(ensureSuccess(res.String("ack", "request", "hash")), ShouldNotBeEmpty)
			So(ensureSuccess(res.String("ack", "response", "hash")), ShouldNotBeEmpty)

			queryType, err := res.String("ack", "request", "type")
			So(err, ShouldBeNil)
			So(queryType, ShouldBeIn, []string{wt.WriteQuery.String(), wt.ReadQuery.String()})

			if queryType == wt.WriteQuery.String() {
				logOffset, err = res.Int("ack", "response", "log_position")
				So(err, ShouldBeNil)
				So(logOffset, ShouldBeGreaterThanOrEqualTo, 0)
				reqHash, err = res.String("ack", "request", "hash")
				So(err, ShouldBeNil)
				So(reqHash, ShouldNotBeEmpty)
			}
		}

		// must contains a write query
		So(reqHash, ShouldNotBeEmpty)
		So(logOffset, ShouldBeGreaterThanOrEqualTo, 0)

		// test get request entity by request hash
		res, err = getJSON("request/%v/%v", dbID, reqHash)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.Interface("request")), ShouldNotBeNil)
		So(ensureSuccess(res.String("request", "hash")), ShouldNotBeEmpty)
		So(ensureSuccess(res.String("request", "type")), ShouldEqual, wt.WriteQuery.String())
		So(ensureSuccess(res.Int("request", "count")), ShouldEqual, 1) // no transaction batch is used
		So(ensureSuccess(res.ArrayOfObjects("request", "queries")), ShouldNotBeEmpty)
		So(ensureSuccess(res.String("request", "queries", "0", "pattern")), ShouldNotBeEmpty)
		byHashRequestResult := ensureSuccess(res.Interface())

		// test get request entity by log offset
		res, err = getJSON("offset/%v/%v", dbID, logOffset)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.Interface()), ShouldResemble, byHashRequestResult)

		// test get first log offset, should be a create table statement
		res, err = getJSON("offset/%v/1", dbID)
		So(err, ShouldBeNil)
		So(ensureSuccess(res.String("request", "queries", "0", "pattern")), ShouldContainSubstring, "CREATE TABLE")

		// test get genesis block by height
		res, err = getJSON("height/%v/0", dbID2)
		So(err, ShouldNotBeNil)
		log.Info(err, res)

		// test get genesis block by height
		res, err = getJSON("head/%v", dbID2)
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
