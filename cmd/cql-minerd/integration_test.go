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
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
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
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	sqlite3 "github.com/CovenantSQL/go-sqlite3-encrypt"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	baseDir                   = utils.GetProjectSrcDir()
	testWorkingDir            = FJ(baseDir, "./test/")
	logDir                    = FJ(testWorkingDir, "./log/")
	testGasPrice       uint64 = 1
	testAdvancePayment uint64 = 20000000
)

var nodeCmds []*utils.CMD

var FJ = filepath.Join

func startNodes() {
	ctx := context.Background()

	// wait for ports to be available
	var err error

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
		"leader", testWorkingDir, logDir, true,
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	err = utils.WaitToConnect(ctx, "127.0.0.1", []int{
		3122,
		3121,
		3120,
	}, time.Second)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		2144,
		2145,
		2146,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	time.Sleep(10 * time.Second)

	// start 3miners
	os.RemoveAll(FJ(testWorkingDir, "./integration/node_miner_0/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./integration/node_miner_0/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-minerd/miner0.cover.out"),
		},
		"miner0", testWorkingDir, logDir, true,
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
		"miner1", testWorkingDir, logDir, true,
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
		"miner2", testWorkingDir, logDir, true,
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
			//"-traceFile", FJ(baseDir, "./cmd/cql-minerd/miner0.trace"),
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
			//"-traceFile", FJ(baseDir, "./cmd/cql-minerd/miner1.trace"),
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
			//"-traceFile", FJ(baseDir, "./cmd/cql-minerd/miner2.trace"),
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
			grepRace := exec.Command("/bin/sh", "-c", "grep -a -A 50 'DATA RACE' "+thisCmd.LogPath)
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

	Convey("test full process", t, func(c C) {
		startNodes()
		defer stopNodes()
		var err error

		time.Sleep(10 * time.Second)

		So(err, ShouldBeNil)

		err = client.Init(FJ(testWorkingDir, "./integration/node_c/config.yaml"), []byte(""))
		So(err, ShouldBeNil)

		var (
			clientPrivKey *asymmetric.PrivateKey
			clientAddr    proto.AccountAddress

			minersPrivKeys = make([]*asymmetric.PrivateKey, 3)
			minersAddrs    = make([]proto.AccountAddress, 3)
		)

		// get miners' private keys
		minersPrivKeys[0], err = kms.LoadPrivateKey(FJ(testWorkingDir, "./integration/node_miner_0/private.key"), []byte{})
		So(err, ShouldBeNil)
		minersPrivKeys[1], err = kms.LoadPrivateKey(FJ(testWorkingDir, "./integration/node_miner_1/private.key"), []byte{})
		So(err, ShouldBeNil)
		minersPrivKeys[2], err = kms.LoadPrivateKey(FJ(testWorkingDir, "./integration/node_miner_2/private.key"), []byte{})
		So(err, ShouldBeNil)
		clientPrivKey, err = kms.LoadPrivateKey(FJ(testWorkingDir, "./integration/node_c/private.key"), []byte{})
		So(err, ShouldBeNil)

		// get miners' addr
		minersAddrs[0], err = crypto.PubKeyHash(minersPrivKeys[0].PubKey())
		So(err, ShouldBeNil)
		minersAddrs[1], err = crypto.PubKeyHash(minersPrivKeys[1].PubKey())
		So(err, ShouldBeNil)
		minersAddrs[2], err = crypto.PubKeyHash(minersPrivKeys[2].PubKey())
		So(err, ShouldBeNil)
		clientAddr, err = crypto.PubKeyHash(clientPrivKey.PubKey())
		So(err, ShouldBeNil)

		// client send create database transaction
		meta := client.ResourceMeta{
			ResourceMeta: types.ResourceMeta{
				TargetMiners: minersAddrs,
				Node:         uint16(len(minersAddrs)),
			},
			GasPrice:       testGasPrice,
			AdvancePayment: testAdvancePayment,
		}

		dsn, err := client.Create(meta)
		So(err, ShouldBeNil)
		dsnCfg, err := client.ParseDSN(dsn)
		So(err, ShouldBeNil)

		// create dsn
		log.Infof("the created database dsn is %v", dsn)

		db, err := sql.Open("covenantsql", dsn)
		So(err, ShouldBeNil)

		// wait for creation
		var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		err = bp.WaitDatabaseCreation(ctx, proto.DatabaseID(dsnCfg.DatabaseID), db, 3*time.Second)
		So(err, ShouldBeNil)

		// check sqlchain profile exist
		dbID := proto.DatabaseID(dsnCfg.DatabaseID)
		profileReq := &types.QuerySQLChainProfileReq{}
		profileResp := &types.QuerySQLChainProfileResp{}

		profileReq.DBID = dbID
		err = rpc.RequestBP(route.MCCQuerySQLChainProfile.String(), profileReq, profileResp)
		So(err, ShouldBeNil)
		profile := profileResp.Profile
		So(profile.Address.DatabaseID(), ShouldEqual, dbID)
		So(profile.Owner.String(), ShouldEqual, clientAddr.String())
		So(profile.TokenType, ShouldEqual, types.Particle)
		minersMap := make(map[proto.AccountAddress]bool)
		for _, miner := range profile.Miners {
			minersMap[miner.Address] = true
		}
		for _, miner := range minersAddrs {
			So(minersMap[miner], ShouldBeTrue)
		}
		usersMap := make(map[proto.AccountAddress]types.PermStat)
		for _, user := range profile.Users {
			usersMap[user.Address] = types.PermStat{
				Permission: user.Permission,
				Status:     user.Status,
			}
		}
		permStat, ok := usersMap[clientAddr]
		So(ok, ShouldBeTrue)
		So(permStat.Permission, ShouldEqual, types.Admin)
		So(permStat.Status, ShouldEqual, types.Normal)

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

		Convey("test query cancel", FailureContinues, func(c C) {
			/* test cancel write query */
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				db.Exec("INSERT INTO test VALUES(sleep(10000000000))")
			}()
			time.Sleep(time.Second)
			wg.Add(1)
			go func() {
				defer wg.Done()
				var err error
				_, err = db.Exec("UPDATE test SET test = 100;")
				// should be canceled
				c.So(err, ShouldNotBeNil)
			}()
			time.Sleep(time.Second)
			for _, n := range conf.GConf.KnownNodes {
				if n.Role == proto.Miner {
					rpc.GetSessionPoolInstance().Remove(n.ID)
				}
			}
			time.Sleep(time.Second)

			// ensure connection
			db.Query("SELECT 1")

			// test before write operation complete
			var result int
			err = db.QueryRow("SELECT * FROM test WHERE test = 4 LIMIT 1").Scan(&result)
			c.So(err, ShouldBeNil)
			c.So(result, ShouldEqual, 4)

			wg.Wait()

			/* test cancel read query */
			go func() {
				_, err = db.Query("SELECT * FROM test WHERE test = sleep(10000000000)")
				// call write query using read query interface
				//_, err = db.Query("INSERT INTO test VALUES(sleep(10000000000))")
				c.So(err, ShouldNotBeNil)
			}()
			time.Sleep(time.Second)
			for _, n := range conf.GConf.KnownNodes {
				if n.Role == proto.Miner {
					rpc.GetSessionPoolInstance().Remove(n.ID)
				}
			}
			time.Sleep(time.Second)
			// ensure connection
			db.Query("SELECT 1")

			/* test long running write query */
			row = db.QueryRow("SELECT * FROM test WHERE test = 10000000000 LIMIT 1")
			err = row.Scan(&result)
			c.So(err, ShouldBeNil)
			c.So(result, ShouldEqual, 10000000000)

			c.So(err, ShouldBeNil)
		})

		time.Sleep(20 * time.Second)

		profileReq = &types.QuerySQLChainProfileReq{}
		profileResp = &types.QuerySQLChainProfileResp{}
		profileReq.DBID = dbID
		err = rpc.RequestBP(route.MCCQuerySQLChainProfile.String(), profileReq, profileResp)
		So(err, ShouldBeNil)
		for _, user := range profileResp.Profile.Users {
			log.Infof("user (%s) left advance payment: %d", user.Address.String(), user.AdvancePayment)
			So(user.AdvancePayment, ShouldNotEqual, testAdvancePayment)
		}
		for _, miner := range profileResp.Profile.Miners {
			So(miner.PendingIncome != 0 || miner.ReceivedIncome != 0, ShouldBeTrue)
		}

		err = db.Close()
		So(err, ShouldBeNil)

		// TODO(lambda): Drop database
	})
}

const ROWSTART = 1000000
const TABLENAME = "insert_table0"

func prepareBenchTable(db *sql.DB) {
	_, err := db.Exec("DROP TABLE IF EXISTS " + TABLENAME + ";")
	So(err, ShouldBeNil)

	_, err = db.Exec(`CREATE TABLE ` + TABLENAME + ` ("k" INT, "v1" TEXT, PRIMARY KEY("k"))`)
	So(err, ShouldBeNil)

	_, err = db.Exec("REPLACE INTO "+TABLENAME+" VALUES(?, ?)", ROWSTART-1, "test")
	So(err, ShouldBeNil)
}

func cleanBenchTable(db *sql.DB) {
	_, err := db.Exec("DELETE FROM "+TABLENAME+" WHERE k >= ?", ROWSTART)
	So(err, ShouldBeNil)
}

func benchDB(b *testing.B, db *sql.DB, createDB bool) {
	var err error
	if createDB {
		prepareBenchTable(db)
	}

	cleanBenchTable(db)

	var i int64
	i = -1

	b.Run("benchmark INSERT", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ii := atomic.AddInt64(&i, 1)
				index := ROWSTART + ii
				//start := time.Now()
				_, err = db.Exec("INSERT INTO "+TABLENAME+" ( k, v1 ) VALUES"+
					"(?, ?)", index, ii,
				)
				//log.Warnf("insert index = %d %v", index, time.Since(start))
				for err != nil && err.Error() == sqlite3.ErrBusy.Error() {
					// retry forever
					log.Warnf("index = %d retried", index)
					_, err = db.Exec("INSERT INTO "+TABLENAME+" ( k, v1 ) VALUES"+
						"(?, ?)", index, ii,
					)
				}
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	routineCount := runtime.NumGoroutine()
	if routineCount > 100 {
		b.Errorf("go routine count: %d", routineCount)
	} else {
		log.Infof("go routine count: %d", routineCount)
	}

	rowCount := db.QueryRow("SELECT COUNT(1) FROM " + TABLENAME)
	var count int64
	err = rowCount.Scan(&count)
	if err != nil {
		b.Fatal(err)
	}
	log.Warnf("row Count: %v", count)

	b.Run("benchmark SELECT", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var index int64
				if createDB { //only data by insert
					index = rand.Int63n(count-1) + ROWSTART
				} else { //has data before ROWSTART
					index = rand.Int63n(count - 1)
				}
				//log.Debugf("index = %d", index)
				//start := time.Now()
				row := db.QueryRow("SELECT v1 FROM "+TABLENAME+" WHERE k = ? LIMIT 1", index)
				//log.Warnf("select index = %d %v", index, time.Since(start))
				var result []byte
				err = row.Scan(&result)
				if err != nil || (len(result) == 0) {
					log.Errorf("index = %d", index)
					b.Fatal(err)
				}
			}
		})
	})

	routineCount = runtime.NumGoroutine()
	if routineCount > 100 {
		b.Errorf("go routine count: %d", routineCount)
	} else {
		log.Infof("go routine count: %d", routineCount)
	}

	//row := db.QueryRow("SELECT nonIndexedColumn FROM test LIMIT 1")

	//var result int
	//err = row.Scan(&result)
	//So(err, ShouldBeNil)
	//So(result, ShouldEqual, 4)

	err = db.Close()
	So(err, ShouldBeNil)
}

func benchMiner(b *testing.B, minerCount uint16, bypassSign bool) {
	log.Warnf("benchmark for %d Miners, BypassSignature: %v", minerCount, bypassSign)
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
		}, 2*time.Second)
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
		meta := client.ResourceMeta{}
		meta.Node = minerCount
		dsn, err = client.Create(meta)
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

	// wait for creation
	dsnCfg, err := client.ParseDSN(dsn)
	So(err, ShouldBeNil)
	var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	err = bp.WaitDatabaseCreation(ctx, proto.DatabaseID(dsnCfg.DatabaseID), db, 3*time.Second)
	So(err, ShouldBeNil)

	benchDB(b, db, minerCount > 0)

	err = client.Drop(dsn)
	So(err, ShouldBeNil)
	time.Sleep(5 * time.Second)
	stopNodes()
}

func BenchmarkSQLite(b *testing.B) {
	var db *sql.DB
	var createDB bool
	millionFile := fmt.Sprintf("/data/sqlite_bigdata/insert_multi_sqlitedb0_1_%v", ROWSTART)
	f, err := os.Open(millionFile)
	if err != nil && os.IsNotExist(err) {
		os.Remove("./foo.db")
		defer os.Remove("./foo.db")

		db, err = sql.Open("sqlite3", "./foo.db?_journal_mode=WAL&_synchronous=NORMAL&cache=shared")
		if err != nil {
			log.Fatal(err)
		}
		createDB = true
		defer db.Close()
	} else {
		f.Close()
		db, err = sql.Open("sqlite3", millionFile+"?_journal_mode=WAL&_synchronous=NORMAL&cache=shared")
		log.Infof("testing sqlite3 million data exist file %v", millionFile)
		if err != nil {
			log.Fatal(err)
		}
		createDB = false
		defer db.Close()
	}

	Convey("bench SQLite", b, func() {
		benchDB(b, db, createDB)
	})
}

func benchGNTEMiner(b *testing.B, minerCount uint16, bypassSign bool) {
	log.Warnf("benchmark GNTE for %d Miners, BypassSignature: %v", minerCount, bypassSign)
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
		meta := client.ResourceMeta{}
		meta.Node = minerCount
		dsn, err = client.Create(meta)
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

	dsnCfg, err := client.ParseDSN(dsn)
	So(err, ShouldBeNil)

	// wait for creation
	var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	err = bp.WaitDatabaseCreation(ctx, proto.DatabaseID(dsnCfg.DatabaseID), db, 3*time.Second)
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

func BenchmarkMinerGNTE1(b *testing.B) {
	Convey("bench GNTE one node", b, func() {
		benchGNTEMiner(b, 1, false)
	})
}
func BenchmarkMinerGNTE2(b *testing.B) {
	Convey("bench GNTE two node", b, func() {
		benchGNTEMiner(b, 2, false)
	})
}

func BenchmarkMinerGNTE3(b *testing.B) {
	Convey("bench GNTE three node", b, func() {
		benchGNTEMiner(b, 3, false)
	})
}

func BenchmarkMinerGNTE4(b *testing.B) {
	Convey("bench GNTE three node", b, func() {
		benchGNTEMiner(b, 4, false)
	})
}

func BenchmarkMinerGNTE8(b *testing.B) {
	Convey("bench GNTE three node", b, func() {
		benchGNTEMiner(b, 8, false)
	})
}
