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
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	sqlite3 "github.com/CovenantSQL/go-sqlite3-encrypt"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/naconn"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/test"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
)

var (
	baseDir                   = utils.GetProjectSrcDir()
	testWorkingDir            = FJ(baseDir, "./test/")
	gnteConfDir               = FJ(testWorkingDir, "./GNTE/conf/node_c/")
	testnetConfDir            = FJ(baseDir, "./conf/testnet/")
	logDir                    = FJ(testWorkingDir, "./log/")
	testGasPrice       uint64 = 1
	testAdvancePayment uint64 = 20000000

	nodeCmds []*utils.CMD

	FJ = filepath.Join

	// Benchmark flags
	benchMinerCount          int
	benchBypassSignature     bool
	benchEventualConsistency bool
	benchMinerDirectRPC      bool
	benchMinerConfigDir      string
)

func init() {
	flag.IntVar(&benchMinerCount, "bench-miner-count", 1,
		"Benchmark miner count.")
	flag.BoolVar(&benchBypassSignature, "bench-bypass-signature", false,
		"Benchmark bypassing signature.")
	flag.BoolVar(&benchEventualConsistency, "bench-eventual-consistency", false,
		"Benchmark with eventaul consistency.")
	flag.BoolVar(&benchMinerDirectRPC, "bench-direct-rpc", false,
		"Benchmark with with direct RPC protocol.")
	flag.StringVar(&benchMinerConfigDir, "bench-miner-config-dir", "",
		"Benchmark custome miner config directory.")
}

func TestMain(m *testing.M) {
	flag.Parse()
	if benchMinerDirectRPC {
		naconn.RegisterResolver(rpc.NewDirectResolver())
	}
	os.Exit(m.Run())
}

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
			"-metric-web", "0.0.0.0:13122",
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
			"-metric-web", "0.0.0.0:13121",
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
			"-metric-web", "0.0.0.0:13120",
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
			"-metric-web", "0.0.0.0:12144",
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
			"-metric-web", "0.0.0.0:12145",
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
			"-metric-web", "0.0.0.0:12146",
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
		bypassArg = "-bypass-signature"
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
			//"-trace-file", FJ(baseDir, "./cmd/cql-minerd/miner0.trace"),
			"-metric-graphite-server", "192.168.2.100:2003",
			"-profile-server", "0.0.0.0:8080",
			"-metric-log",
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
			//"-trace-file", FJ(baseDir, "./cmd/cql-minerd/miner1.trace"),
			"-metric-graphite-server", "192.168.2.100:2003",
			"-profile-server", "0.0.0.0:8081",
			"-metric-log",
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
			//"-trace-file", FJ(baseDir, "./cmd/cql-minerd/miner2.trace"),
			"-metric-graphite-server", "192.168.2.100:2003",
			"-profile-server", "0.0.0.0:8082",
			"-metric-log",
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
			TargetMiners:   minersAddrs,
			Node:           uint16(len(minersAddrs)),
			IsolationLevel: int(sql.LevelReadUncommitted),
			GasPrice:       testGasPrice,
			AdvancePayment: testAdvancePayment,
		}
		// wait for chain service
		var ctx1, cancel1 = context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel1()
		err = test.WaitBPChainService(ctx1, 3*time.Second)
		if err != nil {
			t.Fatalf("wait for chain service failed: %v", err)
		}

		_, dsn, err := client.Create(meta)
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
		err = client.WaitDBCreation(ctx, dsn)
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
		So(permStat.Permission, ShouldNotBeNil)
		So(permStat.Permission.Role, ShouldEqual, types.Admin)
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

		SkipConvey("test query cancel", FailureContinues, func(c C) {
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
		})

		ctx2, ccl2 := context.WithTimeout(context.Background(), 3*time.Minute)
		defer ccl2()
		err = waitProfileChecking(ctx2, 3*time.Second, dbID, func(profile *types.SQLChainProfile) bool {
			for _, user := range profile.Users {
				if user.AdvancePayment != testAdvancePayment {
					return true
				}
			}
			return false
		})
		So(err, ShouldBeNil)

		ctx3, ccl3 := context.WithTimeout(context.Background(), 1*time.Minute)
		defer ccl3()
		err = waitProfileChecking(ctx3, 3*time.Second, dbID, func(profile *types.SQLChainProfile) bool {
			getIncome := false
			for _, miner := range profile.Miners {
				getIncome = getIncome || (miner.PendingIncome != 0 || miner.ReceivedIncome != 0)
			}
			return getIncome
		})
		So(err, ShouldBeNil)

		err = db.Close()
		So(err, ShouldBeNil)

		// test query from follower node
		dsnCfgMix := *dsnCfg
		dsnCfgMix.UseLeader = true
		dsnCfgMix.UseFollower = true
		dbMix, err := sql.Open("covenantsql", dsnCfgMix.FormatDSN())
		So(err, ShouldBeNil)
		defer dbMix.Close()

		result = 0
		err = dbMix.QueryRow("SELECT * FROM test LIMIT 1").Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldEqual, 4)

		_, err = dbMix.Exec("INSERT INTO test VALUES(2)")
		So(err, ShouldBeNil)

		// test query from follower only
		dsnCfgFollower := *dsnCfg
		dsnCfgFollower.UseLeader = false
		dsnCfgFollower.UseFollower = true
		dbFollower, err := sql.Open("covenantsql", dsnCfgFollower.FormatDSN())
		So(err, ShouldBeNil)
		defer dbFollower.Close()

		err = dbFollower.QueryRow("SELECT * FROM test LIMIT 1").Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldEqual, 4)

		_, err = dbFollower.Exec("INSERT INTO test VALUES(2)")
		So(err, ShouldNotBeNil)

		// TODO(lambda): Drop database
	})
}

func waitProfileChecking(ctx context.Context, period time.Duration, dbID proto.DatabaseID,
	checkFunc func(profile *types.SQLChainProfile) bool) (err error) {
	var (
		ticker = time.NewTicker(period)
		req    = &types.QuerySQLChainProfileReq{}
		resp   = &types.QuerySQLChainProfileResp{}
	)
	defer ticker.Stop()
	req.DBID = dbID

	for {
		select {
		case <-ticker.C:
			err = rpc.RequestBP(route.MCCQuerySQLChainProfile.String(), req, resp)
			if err == nil {
				if checkFunc(&resp.Profile) {
					return
				}
				log.WithFields(log.Fields{
					"dbID":        resp.Profile.Address,
					"num_of_user": len(resp.Profile.Users),
				}).Debugf("get profile but failed to check in waitProfileChecking")
			}
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}
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

func makeBenchName(trailings ...string) string {
	var parts = make([]string, 0, 3+len(trailings))
	parts = append(parts, fmt.Sprintf("%dMiner", benchMinerCount))
	if benchBypassSignature {
		parts = append(parts, "BypassSignature")
	}
	if benchEventualConsistency {
		parts = append(parts, "EventualConsistency")
	}
	parts = append(parts, trailings...)
	return strings.Join(parts, "_")
}

func benchDB(b *testing.B, db *sql.DB, createDB bool) {
	var err error
	if createDB {
		prepareBenchTable(db)
	}

	cleanBenchTable(db)

	var i int64
	i = -1
	db.SetMaxIdleConns(256)

	b.Run(makeBenchName("INSERT"), func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				ii := atomic.AddInt64(&i, 1)
				index := ROWSTART + ii
				//start := time.Now()

				ctx, task := trace.NewTask(context.Background(), "BenchInsert")

				_, err = db.ExecContext(ctx, "INSERT INTO "+TABLENAME+" ( k, v1 ) VALUES"+
					"(?, ?)", index, ii,
				)
				//log.Warnf("insert index = %d %v", index, time.Since(start))
				for err != nil && err.Error() == sqlite3.ErrBusy.Error() {
					// retry forever
					log.Warnf("index = %d retried", index)
					_, err = db.ExecContext(ctx, "INSERT INTO "+TABLENAME+" ( k, v1 ) VALUES"+
						"(?, ?)", index, ii,
					)
				}
				if err != nil {
					b.Fatal(err)
				}

				task.End()
			}
		})
	})

	routineCount := runtime.NumGoroutine()
	if routineCount > 150 {
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

	b.Run(makeBenchName("SELECT"), func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var index int64
				if createDB { //only data by insert
					index = rand.Int63n(count-1) + ROWSTART
				} else { //has data before ROWSTART
					index = rand.Int63n(count - 1)
				}

				ctx, task := trace.NewTask(context.Background(), "BenchSelect")
				//log.Debugf("index = %d", index)
				//start := time.Now()
				row := db.QueryRowContext(ctx, "SELECT v1 FROM "+TABLENAME+" WHERE k = ? LIMIT 1", index)
				//log.Warnf("select index = %d %v", index, time.Since(start))
				var result []byte
				err = row.Scan(&result)
				if err != nil || (len(result) == 0) {
					log.Errorf("index = %d", index)
					b.Fatal(err)
				}
				task.End()
			}
		})
	})

	routineCount = runtime.NumGoroutine()
	if routineCount > 150 {
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

func benchMiner(b *testing.B, minerCount uint16) {
	log.Warnf("benchmark for %d Miners, BypassSignature: %v", minerCount, benchBypassSignature)
	asymmetric.BypassSignature = benchBypassSignature
	if minerCount > 0 {
		startNodesProfile(benchBypassSignature)
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
		meta := client.ResourceMeta{
			Node:                   minerCount,
			UseEventualConsistency: benchEventualConsistency,
			IsolationLevel:         int(sql.LevelReadUncommitted),
		}
		// wait for chain service
		var ctx1, cancel1 = context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel1()
		err = test.WaitBPChainService(ctx1, 3*time.Second)
		if err != nil {
			b.Fatalf("wait for chain service failed: %v", err)
		}

		_, dsn, err = client.Create(meta)
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

	if benchMinerDirectRPC {
		dsnCfg, err := client.ParseDSN(dsn)
		So(err, ShouldBeNil)
		dsnCfg.UseDirectRPC = true
		dsn = dsnCfg.FormatDSN()
	}

	db, err := sql.Open("covenantsql", dsn)
	So(err, ShouldBeNil)

	// wait for creation
	var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	err = client.WaitDBCreation(ctx, dsn)
	So(err, ShouldBeNil)

	benchDB(b, db, minerCount > 0)

	_, err = client.Drop(dsn)
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

func benchOutsideMiner(b *testing.B, minerCount uint16, confDir string) {
	benchOutsideMinerWithTargetMinerList(b, minerCount, nil, confDir)
}

func benchOutsideMinerWithTargetMinerList(
	b *testing.B, minerCount uint16, targetMiners []proto.AccountAddress, confDir string,
) {
	log.Warnf("benchmark %v for %d Miners:", confDir, minerCount)

	// Create temp directory
	testDataDir, err := ioutil.TempDir(testWorkingDir, "covenantsql")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(testDataDir)
	clientConf := FJ(confDir, "config.yaml")
	tempConf := FJ(testDataDir, "config.yaml")
	clientKey := FJ(confDir, "private.key")
	tempKey := FJ(testDataDir, "private.key")
	utils.CopyFile(clientConf, tempConf)
	utils.CopyFile(clientKey, tempKey)

	err = client.Init(tempConf, []byte(""))
	So(err, ShouldBeNil)

	for _, node := range conf.GConf.KnownNodes {
		if node.Role == proto.Leader {
			log.Infof("Benching started on bp addr: %v", node.Addr)
			break
		}
	}

	dsnFile := FJ(baseDir, "./cmd/cql-minerd/.dsn")
	var dsn string
	if minerCount > 0 {
		// create
		meta := client.ResourceMeta{
			TargetMiners:           targetMiners,
			Node:                   minerCount,
			UseEventualConsistency: benchEventualConsistency,
			IsolationLevel:         int(sql.LevelReadUncommitted),
			AdvancePayment:         1000000000,
		}
		// wait for chain service
		var ctx1, cancel1 = context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel1()
		err = test.WaitBPChainService(ctx1, 3*time.Second)
		if err != nil {
			b.Fatalf("wait for chain service failed: %v", err)
		}

		_, dsn, err = client.Create(meta)
		So(err, ShouldBeNil)
		log.Infof("the created database dsn is %v", dsn)

		// wait for creation
		var ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		err = client.WaitDBCreation(ctx, dsn)
		So(err, ShouldBeNil)

		err = ioutil.WriteFile(dsnFile, []byte(dsn), 0666)
		if err != nil {
			log.Errorf("write .dsn failed: %v", err)
		}
		defer os.Remove(dsnFile)
		defer client.Drop(dsn)
	} else {
		dsn = os.Getenv("DSN")
	}

	db, err := sql.Open("covenantsql", dsn)
	So(err, ShouldBeNil)

	benchDB(b, db, minerCount > 0)
}

func BenchmarkClientOnly(b *testing.B) {
	Convey("bench three node", b, func() {
		benchMiner(b, 0)
	})
}

func BenchmarkMiner(b *testing.B) {
	Convey(fmt.Sprintf("bench %d node(s)", benchMinerCount), b, func() {
		benchMiner(b, uint16(benchMinerCount))
	})
}

func BenchmarkMinerGNTE(b *testing.B) {
	Convey(fmt.Sprintf("bench GNTE %d node(s)", benchMinerCount), b, func() {
		benchOutsideMiner(b, uint16(benchMinerCount), gnteConfDir)
	})
}

func BenchmarkTestnetMiner(b *testing.B) {
	Convey(fmt.Sprintf("bench testnet %d node(s)", benchMinerCount), b, func() {
		benchOutsideMiner(b, uint16(benchMinerCount), testnetConfDir)
	})
}

func BenchmarkCustomMiner(b *testing.B) {
	Convey(fmt.Sprintf("bench custom %d node(s)", benchMinerCount), b, func() {
		benchOutsideMiner(b, uint16(benchMinerCount), benchMinerConfigDir)
	})
}
