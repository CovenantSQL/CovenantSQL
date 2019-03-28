/*
 * Copyright 2019 The CovenantSQL Authors.
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

package mirror

import (
	"context"
	"database/sql"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/test"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	baseDir        = utils.GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/")
	logDir         = FJ(testWorkingDir, "./log/")
)

var nodeCmds []*utils.CMD

var FJ = filepath.Join

func startNodes() {
	// wait for ports to be available
	var err error
	ctx := context.Background()
	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		5120,
		5121,
		5122,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	var cmd *utils.CMD
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./mirror/node_0/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql/leader-mirror.cover.out"),
		},
		"leader", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./mirror/node_1/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql/follower1-mirror.cover.out"),
		},
		"follower1", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./mirror/node_2/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql/follower2-mirror.cover.out"),
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
		5120,
		5121,
		5122,
	}, time.Second)
	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		5144,
		5145,
		5146,
	}, time.Millisecond*200)
	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	time.Sleep(10 * time.Second)
	// start 3miners
	os.RemoveAll(FJ(testWorkingDir, "./mirror/node_miner_0/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./mirror/node_miner_0/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql/miner0-mirror.cover.out"),
		},
		"miner0", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	os.RemoveAll(FJ(testWorkingDir, "./mirror/node_miner_1/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./mirror/node_miner_1/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql/miner1-mirror.cover.out"),
		},
		"miner1", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	os.RemoveAll(FJ(testWorkingDir, "./mirror/node_miner_2/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./mirror/node_miner_2/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql/miner2-mirror.cover.out"),
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

func waitForMirrorComplete(ctx context.Context, dbID string, tick time.Duration, stableDuration time.Duration) (err error) {
	progressFile := FJ(testWorkingDir, "./mirror/node_mirror/"+dbID+progressFileSuffix)
	lastProgress := 0
	lastUpdate := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(tick):
			progressData, _ := ioutil.ReadFile(progressFile)
			progressCount, _ := strconv.Atoi(string(progressData))
			log.WithFields(log.Fields{
				"lastUpdate":     lastUpdate.String(),
				"stableDuration": stableDuration,
				"progressCount":  progressCount,
			}).Infof("current mirror count progress")
			if progressCount > lastProgress {
				lastProgress = progressCount
				lastUpdate = time.Now()
			}
			if progressCount > 5 || (progressCount > 0 && time.Now().Sub(lastUpdate) > stableDuration) {
				// mirror synced
				return
			}
		}
	}
}

func TestFullProcess(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	Convey("test mirror full process", t, func() {
		var (
			err error
		)

		startNodes()
		defer stopNodes()

		err = client.Init(FJ(testWorkingDir, "./mirror/node_c/config.yaml"), []byte(""))
		So(err, ShouldBeNil)

		// wait bp chain service to start
		ctx, ccl1 := context.WithTimeout(context.Background(), 1*time.Minute)
		defer ccl1()
		err = test.WaitBPChainService(ctx, 3*time.Second)
		So(err, ShouldBeNil)

		// create
		meta := client.ResourceMeta{}
		meta.Node = 1
		_, dsn, err := client.Create(meta)
		So(err, ShouldBeNil)
		dsnCfg, err := client.ParseDSN(dsn)
		So(err, ShouldBeNil)

		log.Infof("the created database dsn is %v", dsn)

		db, err := sql.Open("covenantsql", dsn)
		So(err, ShouldBeNil)
		defer db.Close()

		// wait for creation
		ctx, ccl2 := context.WithTimeout(context.Background(), 5*time.Minute)
		defer ccl2()
		err = client.WaitDBCreation(ctx, dsn)
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

		// run mirror node
		utils.RemoveAll(FJ(testWorkingDir, "./mirror/node_mirror/"+dsnCfg.DatabaseID+"*"))
		defer utils.RemoveAll(FJ(testWorkingDir, "./mirror/node_mirror/"+dsnCfg.DatabaseID+"*"))

		var mirrorCmd *utils.CMD
		mirrorCmd, err = utils.RunCommandNB(
			FJ(baseDir, "./bin/cql.test"),
			[]string{"-test.coverprofile", FJ(baseDir, "./cmd/cql/mirror.cover.out"),
				"mirror",
				"-config", FJ(testWorkingDir, "./mirror/node_mirror/config.yaml"),
				"-no-password",
				"-bg-log-level", "debug",
				dsnCfg.DatabaseID,
				"127.0.0.1:5663",
			},
			"mirror", testWorkingDir, logDir, true,
		)
		So(err, ShouldBeNil)
		defer func() {
			mirrorCmd.Cmd.Process.Signal(syscall.SIGTERM)
			mirrorCmd.Cmd.Wait()
		}()

		defer func() {
			_ = mirrorCmd.Cmd.Process.Signal(syscall.SIGTERM)
			_ = mirrorCmd.Cmd.Wait()
		}()

		err = utils.WaitToConnect(context.Background(), "127.0.0.1", []int{5663}, 200*time.Millisecond)
		So(err, ShouldBeNil)

		time.Sleep(time.Second)

		// check subscription status, wait for 5 period
		ctx, ccl3 := context.WithTimeout(context.Background(), 10*conf.GConf.SQLChainPeriod)
		defer ccl3()
		err = waitForMirrorComplete(ctx, dsnCfg.DatabaseID, conf.GConf.SQLChainPeriod, 2*conf.GConf.SQLChainPeriod)
		So(err, ShouldBeNil)

		// mirror synced, query using mirror
		dsnCfg.Mirror = "127.0.0.1:5663"
		mirrorDSN := dsnCfg.FormatDSN()
		log.Infof("the mirror dsn is %v", mirrorDSN)

		dbMirror, err := sql.Open("covenantsql", mirrorDSN)
		So(err, ShouldBeNil)
		defer dbMirror.Close()

		// test read query
		row = dbMirror.QueryRow("SELECT * FROM test LIMIT 1")
		err = row.Scan(&result)
		So(err, ShouldBeNil)
		So(result, ShouldEqual, 4)

		// test write query, must not success
		_, err = dbMirror.Exec("INSERT INTO test VALUES(?)", 5)
		So(err, ShouldNotBeNil)
	})
}
