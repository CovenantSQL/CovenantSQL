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
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
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
	ctx := context.Background()

	// wait for ports to be available
	var err error
	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		2144,
		2145,
		2146,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		3122,
		3121,
		3120,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	// start 3bps
	var cmd *utils.CMD
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_0/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-minerd/leader.cover.out"),
		},
		"leader", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_1/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-minerd/follower1.cover.out"),
		},
		"follower1", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_2/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-minerd/follower2.cover.out"),
		},
		"follower2", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	time.Sleep(time.Second * 3)

	// start 3miners
	os.RemoveAll(FJ(testWorkingDir, "./integration/node_miner_0/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_miner_0/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-minerd/miner0.cover.out"),
		},
		"miner0", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	os.RemoveAll(FJ(testWorkingDir, "./integration/node_miner_1/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_miner_1/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-minerd/miner1.cover.out"),
		},
		"miner1", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	os.RemoveAll(FJ(testWorkingDir, "./integration/node_miner_2/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_miner_2/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-minerd/miner2.cover.out"),
		},
		"miner2", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
}
func startNodesProfile(bypassSign bool) {
	ctx := context.Background()
	bypassArg := ""
	if bypassSign {
		bypassArg = "-bypassSignature"
	}

	// wait for ports to be available
	var err error
	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		2144,
		2145,
		2146,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		3122,
		3121,
		3120,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	// start 3bps
	var cmd *utils.CMD
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_0/config.yaml"),
			bypassArg,
		},
		"leader", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_1/config.yaml"),
			bypassArg,
		},
		"follower1", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_2/config.yaml"),
			bypassArg,
		},
		"follower2", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	time.Sleep(time.Second * 3)

	// start 3miners
	os.RemoveAll(FJ(testWorkingDir, "./integration/node_miner_0/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_miner_0/config.yaml"),
			"-cpu-profile", FJ(baseDir, "./cmd/cql-minerd/miner0.profile"),
			"-traceFile", FJ(baseDir, "./cmd/cql-minerd/miner0.trace"),
			"-metricGraphiteServer", "192.168.2.100:2003",
			"-profileServer", "0.0.0.0:8080",
			"-metricLog",
			bypassArg,
		},
		"miner0", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	os.RemoveAll(FJ(testWorkingDir, "./integration/node_miner_1/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_miner_1/config.yaml"),
			"-cpu-profile", FJ(baseDir, "./cmd/cql-minerd/miner1.profile"),
			"-traceFile", FJ(baseDir, "./cmd/cql-minerd/miner1.trace"),
			"-metricGraphiteServer", "192.168.2.100:2003",
			"-profileServer", "0.0.0.0:8081",
			"-metricLog",
			bypassArg,
		},
		"miner1", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	os.RemoveAll(FJ(testWorkingDir, "./integration/node_miner_2/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_miner_2/config.yaml"),
			"-cpu-profile", FJ(baseDir, "./cmd/cql-minerd/miner2.profile"),
			"-traceFile", FJ(baseDir, "./cmd/cql-minerd/miner2.trace"),
			"-metricGraphiteServer", "192.168.2.100:2003",
			"-profileServer", "0.0.0.0:8082",
			"-metricLog",
			bypassArg,
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
			grepRace := exec.Command("/bin/sh", "-c", "grep -A 50 'DATA RACE' "+thisCmd.LogPath)
			out, _ := grepRace.Output()
			if len(out) > 2 {
				log.Fatalf("DATA RACE in %s :\n%s", thisCmd.Cmd.Path, string(out))
			}
		}(nodeCmd)
	}

	wg.Wait()
}

func TestFullProcess(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	Convey("test full process", t, func() {
		startNodes()
		defer stopNodes()
		var err error
		err = utils.WaitToConnect(context.Background(), "127.0.0.1", []int{
			2144,
			2145,
			2146,
			3122,
			3121,
			3120,
		}, time.Millisecond*200)
		time.Sleep(time.Second)
		So(err, ShouldBeNil)

		err = client.Init(FJ(testWorkingDir, "./integration/node_c/config.yaml"), []byte(""))
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

		err = client.Drop(dsn)
		So(err, ShouldBeNil)
	})
}

func benchDB(b *testing.B, db *sql.DB, createDB bool) {
	var err error
	if createDB {
		_, err := db.Exec("DROP TABLE IF EXISTS test;")
		So(err, ShouldBeNil)

		_, err = db.Exec("CREATE TABLE test ( indexedColumn, nonIndexedColumn );")
		So(err, ShouldBeNil)

		_, err = db.Exec("CREATE INDEX testIndexedColumn ON test ( indexedColumn );")
		So(err, ShouldBeNil)

		_, err = db.Exec("INSERT INTO test VALUES(?, ?)", 4, 4)
		So(err, ShouldBeNil)
	}

	rand.Seed(time.Now().UnixNano())
	start := (rand.Int31() % 100) * 10000

	var i int32
	var insertedCount int
	b.Run("benchmark INSERT", func(b *testing.B) {
		b.ResetTimer()
		insertedCount = b.N
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ii := atomic.AddInt32(&i, 1)
				_, err = db.Exec("INSERT INTO test ( indexedColumn, nonIndexedColumn ) VALUES"+
					"(?, ?)", start+ii, ii,
				)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	rowCount := db.QueryRow("SELECT COUNT(1) FROM test")
	var count int
	err = rowCount.Scan(&count)
	if err != nil {
		b.Fatal(err)
	}
	log.Warnf("Row Count: %d", count)

	b.Run("benchmark SELECT", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i%1000 == 0 {
				log.Warnf("NOW index: %d", i)
			}
			row := db.QueryRow("SELECT nonIndexedColumn FROM test WHERE indexedColumn = ? LIMIT 1", i%insertedCount)
			var result int
			err = row.Scan(&result)
			if err != nil || result < 0 {
				log.Errorf("i = %d", i)
				b.Fatal(err)
			}
		}
	})

	row := db.QueryRow("SELECT nonIndexedColumn FROM test LIMIT 1")

	var result int
	err = row.Scan(&result)
	So(err, ShouldBeNil)
	So(result, ShouldEqual, 4)

	err = db.Close()
	So(err, ShouldBeNil)
}

func benchMiner(b *testing.B, minerCount uint16, bypassSign bool) {
	log.Warnf("Benchmark for %d Miners, BypassSignature: %v", minerCount, bypassSign)
	asymmetric.BypassSignature = bypassSign
	if minerCount > 0 {
		startNodesProfile(bypassSign)
		utils.WaitToConnect(context.Background(), "127.0.0.1", []int{
			2144,
			2145,
			2146,
			3122,
			3121,
			3120,
		}, time.Millisecond*200)
		time.Sleep(time.Second)
	}

	// Create temp directory
	testDataDir, err := ioutil.TempDir(testWorkingDir, "covenantsql")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(testDataDir)
	clientConf := FJ(testWorkingDir, "./integration/node_c/config.yaml")
	tempConf := FJ(testDataDir, "config.yaml")
	clientKey := FJ(testWorkingDir, "./integration/node_c/private.key")
	tempKey := FJ(testDataDir, "private.key")
	utils.CopyFile(clientConf, tempConf)
	utils.CopyFile(clientKey, tempKey)

	err = client.Init(tempConf, []byte(""))
	So(err, ShouldBeNil)

	dsnFile := FJ(baseDir, "./cmd/cql-minerd/.dsn")
	var dsn string
	if minerCount > 0 {
		// create
		dsn, err = client.Create(client.ResourceMeta{Node: minerCount})
		So(err, ShouldBeNil)

		log.Infof("the created database dsn is %v", dsn)
		err = ioutil.WriteFile(dsnFile, []byte(dsn), 0666)
		if err != nil {
			log.Errorf("write .dsn failed: %v", err)
		}
		defer os.Remove(dsnFile)
	} else {
		dsn = os.Getenv("DSN")
	}

	db, err := sql.Open("covenantsql", dsn)
	So(err, ShouldBeNil)

	benchDB(b, db, minerCount > 0)

	err = client.Drop(dsn)
	So(err, ShouldBeNil)
	time.Sleep(5 * time.Second)
	stopNodes()
}

func BenchmarkSQLite(b *testing.B) {
	os.Remove("./foo.db")
	defer os.Remove("./foo.db")

	db, err := sql.Open("sqlite3", "./foo.db?_journal_mode=WAL&_synchronous=NORMAL&cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	Convey("bench SQLite", b, func() {
		benchDB(b, db, true)
	})
}

func benchGNTEMiner(b *testing.B, minerCount uint16, bypassSign bool) {
	log.Warnf("Benchmark GNTE for %d Miners, BypassSignature: %v", minerCount, bypassSign)
	asymmetric.BypassSignature = bypassSign

	// Create temp directory
	testDataDir, err := ioutil.TempDir(testWorkingDir, "covenantsql")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(testDataDir)
	clientConf := FJ(testWorkingDir, "./GNTE/conf/node_c/config.yaml")
	tempConf := FJ(testDataDir, "config.yaml")
	clientKey := FJ(testWorkingDir, "./GNTE/conf/node_c/private.key")
	tempKey := FJ(testDataDir, "private.key")
	utils.CopyFile(clientConf, tempConf)
	utils.CopyFile(clientKey, tempKey)

	err = client.Init(tempConf, []byte(""))
	So(err, ShouldBeNil)

	dsnFile := FJ(baseDir, "./cmd/cql-minerd/.dsn")
	var dsn string
	if minerCount > 0 {
		// create
		dsn, err = client.Create(client.ResourceMeta{Node: minerCount})
		So(err, ShouldBeNil)

		log.Infof("the created database dsn is %v", dsn)
		err = ioutil.WriteFile(dsnFile, []byte(dsn), 0666)
		if err != nil {
			log.Errorf("write .dsn failed: %v", err)
		}
		defer os.Remove(dsnFile)
	} else {
		dsn = os.Getenv("DSN")
	}

	db, err := sql.Open("covenantsql", dsn)
	So(err, ShouldBeNil)

	benchDB(b, db, minerCount > 0)

	err = client.Drop(dsn)
	So(err, ShouldBeNil)
	time.Sleep(5 * time.Second)
	stopNodes()
}

func BenchmarkMinerOneNoSign(b *testing.B) {
	Convey("bench single node", b, func() {
		benchMiner(b, 1, true)
	})
}

func BenchmarkMinerTwoNoSign(b *testing.B) {
	Convey("bench two node", b, func() {
		benchMiner(b, 2, true)
	})
}

func BenchmarkMinerThreeNoSign(b *testing.B) {
	Convey("bench three node", b, func() {
		benchMiner(b, 3, true)
	})
}

func BenchmarkMinerOne(b *testing.B) {
	Convey("bench single node", b, func() {
		benchMiner(b, 1, false)
	})
}

func BenchmarkMinerTwo(b *testing.B) {
	Convey("bench two node", b, func() {
		benchMiner(b, 2, false)
	})
}

func BenchmarkMinerThree(b *testing.B) {
	Convey("bench three node", b, func() {
		benchMiner(b, 3, false)
	})
}

func BenchmarkClientOnly(b *testing.B) {
	Convey("bench three node", b, func() {
		benchMiner(b, 0, false)
	})
}

func BenchmarkMinerGNTETwo(b *testing.B) {
	Convey("bench GNTE two node", b, func() {
		benchGNTEMiner(b, 2, false)
	})
}
