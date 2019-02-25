/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Marc Berhault (marc@cockroachlabs.com)

package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/test"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	baseDir        = utils.GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/")
	logDir         = FJ(testWorkingDir, "./log/")
	db             *sql.DB
)

var nodeCmds []*utils.CMD

var FJ = filepath.Join

func TestMain(m *testing.M) {
	os.Exit(func() int {
		var stop func()
		db, stop = initTestDB()
		if db == nil {
			stop()
			log.Fatalf("init test DB failed")
		}
		defer stop()
		defer db.Close()
		return m.Run()
	}())
}

func startNodes() {
	ctx := context.Background()

	// wait for ports to be available
	var err error

	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		6122,
		6121,
		6120,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	// start 3bps
	var cmd *utils.CMD
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./fuse/node_0/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-fuse/leader.cover.out"),
		},
		"leader", testWorkingDir, logDir, true,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./fuse/node_1/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-fuse/follower1.cover.out"),
		},
		"follower1", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld.test"),
		[]string{"-config", FJ(testWorkingDir, "./fuse/node_2/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-fuse/follower2.cover.out"),
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
		6122,
		6121,
		6120,
	}, time.Second)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		3144,
		3145,
		3146,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	time.Sleep(10 * time.Second)

	// start 3miners
	os.RemoveAll(FJ(testWorkingDir, "./fuse/node_miner_0/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./fuse/node_miner_0/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-fuse/miner0.cover.out"),
		},
		"miner0", testWorkingDir, logDir, true,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	os.RemoveAll(FJ(testWorkingDir, "./fuse/node_miner_1/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./fuse/node_miner_1/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-fuse/miner1.cover.out"),
		},
		"miner1", testWorkingDir, logDir, false,
	); err == nil {
		nodeCmds = append(nodeCmds, cmd)
	} else {
		log.Errorf("start node failed: %v", err)
	}

	os.RemoveAll(FJ(testWorkingDir, "./fuse/node_miner_2/data"))
	if cmd, err = utils.RunCommandNB(
		FJ(baseDir, "./bin/cql-minerd.test"),
		[]string{"-config", FJ(testWorkingDir, "./fuse/node_miner_2/config.yaml"),
			"-test.coverprofile", FJ(baseDir, "./cmd/cql-fuse/miner2.cover.out"),
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
	testDir := FJ(testWorkingDir, "./fuse")
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
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf(`cd %s && find . -name '*.db' -exec rm -vf {} \;`, testDir))
	cmd.Run()
	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf(`cd %s && find . -name '*.db-shm' -exec rm -vf {} \;`, testDir))
	cmd.Run()
	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf(`cd %s && find . -name '*.db-wal' -exec rm -vf {} \;`, testDir))
	cmd.Run()
	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf(`cd %s && find . -name 'db.meta' -exec rm -vf {} \;`, testDir))
	cmd.Run()
	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf(`cd %s && find . -name 'public.keystore' -exec rm -vf {} \;`, testDir))
	cmd.Run()
	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf(`cd %s && find . -name '*.public.keystore' -exec rm -vf {} \;`, testDir))
	cmd.Run()
	cmd = exec.Command("/bin/sh", "-c", fmt.Sprintf(`cd %s && find . -name '*.ldb' -exec rm -vrf {} \;`, testDir))
	cmd.Run()
}

func initTestDB() (*sql.DB, func()) {

	startNodes()
	var err error

	time.Sleep(10 * time.Second)

	err = client.Init(FJ(testWorkingDir, "./fuse/node_c/config.yaml"), []byte(""))
	if err != nil {
		log.Errorf("init client failed: %v", err)
		return nil, stopNodes
	}

	// wait for chain service
	var ctx1, cancel1 = context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel1()
	err = test.WaitBPChainService(ctx1, 3*time.Second)
	if err != nil {
		log.Errorf("wait for chain service failed: %v", err)
		return nil, stopNodes
	}

	// create
	meta := client.ResourceMeta{}
	meta.Node = 1
	_, dsn, err := client.Create(meta)
	if err != nil {
		log.Errorf("create db failed: %v", err)
		return nil, stopNodes
	}

	db, err := sql.Open("covenantsql", dsn)
	if err != nil {
		log.Errorf("open db failed: %v", err)
		return nil, stopNodes
	}

	// wait for creation
	var ctx2, cancel2 = context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel2()
	err = client.WaitDBCreation(ctx2, dsn)
	if err != nil {
		log.Errorf("wait for creation failed: %v", err)
		return nil, stopNodes
	}

	log.Infof("the created database dsn is %v", dsn)

	if err := initSchema(db); err != nil {
		stopNodes()
		log.Fatal(err)
	}

	return db, stopNodes
}

func getAllBlocks(db *sql.DB, inode uint64) ([]byte, error) {
	blocks, err := getBlocks(db, inode)
	if err != nil {
		return nil, err
	}
	num := len(blocks)
	var data []byte
	for i, b := range blocks {
		if i != b.block {
			// We can't have missing blocks.
			return nil, fmt.Errorf("gap in block list, found block %d at index %d", b.block, i)
		}
		bl := uint64(len(b.data))
		if bl == 0 {
			return nil, fmt.Errorf("empty block found at %d (out of %d blocks)", i, num)
		}
		if i != (num-1) && bl != BlockSize {
			return nil, fmt.Errorf("non-blocksize %d at %d (out of %d blocks)", bl, i, num)
		}
		data = append(data, b.data...)
	}
	return data, nil
}

func TestBlockInfo(t *testing.T) {
	testCases := []struct {
		start, length uint64
		expected      blockRange
	}{
		{0, 0, blockRange{0, 0, 0, 0, 0}},
		{0, BlockSize * 4, blockRange{0, 0, BlockSize, 4, 0}},
		{0, BlockSize*4 + 500, blockRange{0, 0, BlockSize, 4, 500}},
		{500, BlockSize * 4, blockRange{0, 500, BlockSize - 500, 4, 500}},
		{BlockSize, BlockSize * 4, blockRange{1, 0, BlockSize, 5, 0}},
		{BlockSize, 500, blockRange{1, 0, 500, 1, 500}},
		{500, 1000, blockRange{0, 500, 1000, 0, 1500}},
	}

	for tcNum, tc := range testCases {
		actual := newBlockRange(tc.start, tc.length)
		if !reflect.DeepEqual(actual, tc.expected) {
			t.Errorf("#%d: expected:\n%+v\ngot:\n%+v", tcNum, tc.expected, actual)
		}
	}
}

func tryGrow(db *sql.DB, data []byte, id, newSize uint64) ([]byte, error) {
	originalSize := uint64(len(data))
	data = append(data, make([]byte, newSize-originalSize)...)
	if err := grow(db, id, originalSize, newSize); err != nil {
		return nil, err
	}
	newData, err := getAllBlocks(db, id)
	if err != nil {
		return nil, err
	}
	if uint64(len(newData)) != newSize {
		return nil, fmt.Errorf("getAllBlocks lengths don't match: got %d, expected %d", len(newData), newSize)
	}
	if !bytes.Equal(data, newData) {
		return nil, fmt.Errorf("getAllBlocks data doesn't match")
	}

	if newSize == 0 {
		return newData, nil
	}

	// Check the read as well.
	newData, err = read(db, id, 0, newSize)
	if err != nil {
		return nil, err
	}

	if uint64(len(newData)) != newSize {
		return nil, fmt.Errorf("read lengths don't match: got %d, expected %d", len(newData), newSize)
	}
	if !bytes.Equal(data, newData) {
		return nil, fmt.Errorf("read data doesn't match")
	}

	return newData, nil
}

func tryShrink(db *sql.DB, data []byte, id, newSize uint64) ([]byte, error) {
	originalSize := uint64(len(data))
	data = data[:newSize]
	if err := shrink(db, id, originalSize, newSize); err != nil {
		return nil, err
	}
	newData, err := getAllBlocks(db, id)
	if err != nil {
		return nil, err
	}
	if uint64(len(newData)) != newSize {
		return nil, fmt.Errorf("getAllData lengths don't match: got %d, expected %d", len(newData), newSize)
	}
	if !bytes.Equal(data, newData) {
		return nil, fmt.Errorf("getAllData data doesn't match")
	}

	if newSize == 0 {
		return newData, nil
	}

	// Check the read as well.
	newData, err = read(db, id, 0, newSize)
	if err != nil {
		return nil, err
	}

	if uint64(len(newData)) != newSize {
		return nil, fmt.Errorf("read lengths don't match: got %d, expected %d", len(newData), newSize)
	}
	if !bytes.Equal(data, newData) {
		return nil, fmt.Errorf("read data doesn't match")
	}

	return newData, nil
}

func TestShrinkGrow(t *testing.T) {

	id := uint64(10)

	var err error
	data := []byte{}

	if data, err = tryGrow(db, data, id, BlockSize*4+500); err != nil {
		log.Fatal(err)
	}
	if data, err = tryGrow(db, data, id, BlockSize*4+600); err != nil {
		log.Fatal(err)
	}
	if data, err = tryGrow(db, data, id, BlockSize*5); err != nil {
		log.Fatal(err)
	}
	if data, err = tryGrow(db, data, id, BlockSize*999); err != nil {
		log.Fatal(err)
	}

	// Shrink it down to 0.
	if data, err = tryShrink(db, data, id, 0); err != nil {
		log.Fatal(err)
	}
	if data, err = tryGrow(db, data, id, BlockSize*3+500); err != nil {
		log.Fatal(err)
	}
	if data, err = tryShrink(db, data, id, BlockSize*3+300); err != nil {
		log.Fatal(err)
	}
	if data, err = tryShrink(db, data, id, BlockSize*3); err != nil {
		log.Fatal(err)
	}
	if data, err = tryShrink(db, data, id, 0); err != nil {
		log.Fatal(err)
	}
	if data, err = tryGrow(db, data, id, BlockSize); err != nil {
		log.Fatal(err)
	}
	if data, err = tryShrink(db, data, id, BlockSize-200); err != nil {
		log.Fatal(err)
	}
	if data, err = tryShrink(db, data, id, BlockSize-500); err != nil {
		log.Fatal(err)
	}
	if data, err = tryShrink(db, data, id, 0); err != nil {
		log.Fatal(err)
	}
}

func TestReadWriteBlocks(t *testing.T) {

	id := uint64(10)
	rng, _ := NewPseudoRand()
	length := BlockSize*3 + 500
	part1 := RandBytes(rng, length)

	if err := write(db, id, 0, 0, part1); err != nil {
		log.Fatal(err)
	}

	readData, err := read(db, id, 0, uint64(length))
	if err != nil {
		log.Fatal(err)
	}
	if !bytes.Equal(part1, readData) {
		t.Errorf("bytes differ. lengths: %d, expected %d", len(readData), len(part1))
	}

	verboseData, err := getAllBlocks(db, id)
	if err != nil {
		log.Fatal(err)
	}
	if !bytes.Equal(verboseData, part1) {
		t.Errorf("bytes differ. lengths: %d, expected %d", len(verboseData), len(part1))
	}

	// Write with hole in the middle.
	part2 := make([]byte, BlockSize*2+250, BlockSize*2+250)
	fullData := append(part1, part2...)
	part3 := RandBytes(rng, BlockSize+123)
	if err := write(db, id, uint64(len(part1)), uint64(len(fullData)), part3); err != nil {
		log.Fatal(err)
	}
	fullData = append(fullData, part3...)
	readData, err = read(db, id, 0, uint64(len(fullData)))
	if err != nil {
		log.Fatal(err)
	}
	if !bytes.Equal(fullData, readData) {
		t.Errorf("bytes differ. lengths: %d, expected %d", len(readData), len(fullData))
	}

	verboseData, err = getAllBlocks(db, id)
	if err != nil {
		log.Fatal(err)
	}
	if !bytes.Equal(verboseData, fullData) {
		t.Errorf("bytes differ. lengths: %d, expected %d", len(verboseData), len(fullData))
	}

	// Now write into the middle of the file.
	part2 = RandBytes(rng, len(part2))
	if err := write(db, id, uint64(len(fullData)), uint64(len(part1)), part2); err != nil {
		log.Fatal(err)
	}
	fullData = append(part1, part2...)
	fullData = append(fullData, part3...)
	readData, err = read(db, id, 0, uint64(len(fullData)))
	if err != nil {
		log.Fatal(err)
	}
	if !bytes.Equal(fullData, readData) {
		t.Errorf("bytes differ. lengths: %d, expected %d", len(readData), len(fullData))
	}

	verboseData, err = getAllBlocks(db, id)
	if err != nil {
		log.Fatal(err)
	}
	if !bytes.Equal(verboseData, fullData) {
		t.Errorf("bytes differ. lengths: %d, expected %d", len(verboseData), len(fullData))
	}

	// New file.
	id2 := uint64(20)
	if err := write(db, id2, 0, 0, []byte("1")); err != nil {
		log.Fatal(err)
	}
	readData, err = read(db, id2, 0, 1)
	if err != nil {
		log.Fatal(err)
	}
	if string(readData) != "1" {
		log.Fatalf("mismatch: %s", readData)
	}

	if err := write(db, id2, 1, 0, []byte("22")); err != nil {
		log.Fatal(err)
	}
	readData, err = read(db, id2, 0, 2)
	if err != nil {
		log.Fatal(err)
	}
	if string(readData) != "22" {
		log.Fatalf("mismatch: %s", readData)
	}

	id3 := uint64(30)
	part1 = RandBytes(rng, BlockSize)
	// Write 5 blocks.
	var offset uint64
	for i := 0; i < 5; i++ {
		if err := write(db, id3, offset, offset, part1); err != nil {
			log.Fatal(err)
		}
		offset += BlockSize
	}
}
