/*
 * Copyright 2018 The ThunderDB Authors.
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
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"gitlab.com/thunderdb/ThunderDB/conf"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

var (
	baseDir        = GetProjectSrcDir()
	testWorkingDir = FJ(baseDir, "./test/")
	logDir         = FJ(testWorkingDir, "./log/")
)

var FJ = filepath.Join

func GetProjectSrcDir() string {
	_, testFile, _, _ := runtime.Caller(0)
	return FJ(filepath.Dir(testFile), "../../")
}

func Build() (err error) {
	wd := GetProjectSrcDir()
	err = os.Chdir(wd)
	if err != nil {
		log.Errorf("change working dir failed: %s", err)
	}
	cmd := exec.Command("./build.sh")

	err = cmd.Run()
	if err != nil {
		log.Errorf("build failed: %s", err)
	}
	return
}

func RunServer(bin, conf, name string, toStd bool) (err error) {
	logFD, err := os.Create(FJ(logDir, name+".log"))
	if err != nil {
		log.Errorf("create log file failed: %s", err)
		return
	}

	os.Chdir(testWorkingDir)
	cmd := exec.Command(bin, "-config", conf)
	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	var stdout, stderr io.Writer
	if toStd {
		stdout = io.MultiWriter(os.Stdout, logFD)
		stderr = io.MultiWriter(os.Stderr, logFD)
	} else {
		stdout = logFD
		stderr = logFD
	}

	err = cmd.Start()
	if err != nil {
		log.Errorf("cmd.Start() failed with '%s'", err)
		return
	}

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()

	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()

	err = cmd.Wait()
	if err != nil {
		log.Errorf("cmd %s args %s failed with %s", cmd.Path, cmd.Args, err)
		return
	}
	if errStdout != nil {
		log.Errorf("failed to capture stdout %s", errStdout)
		return
	}
	if errStderr != nil {
		log.Errorf("failed to capture stderr %s", errStderr)
		return
	}
	return
}

func TestBuild(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	Build()
}

func TestStartBPs(t *testing.T) {
	go RunServer(FJ(baseDir, "./bin/thunderdbd"), FJ(testWorkingDir, "./node_0/config.yaml"), "leader", false)
	go RunServer(FJ(baseDir, "./bin/thunderdbd"), FJ(testWorkingDir, "./node_1/config.yaml"), "follower1", false)
	go RunServer(FJ(baseDir, "./bin/thunderdbd"), FJ(testWorkingDir, "./node_2/config.yaml"), "follower2", false)
}

func BenchmarkKayakKVServer_GetAllNodeInfo(b *testing.B) {
	log.SetLevel(log.DebugLevel)

	go RunServer(FJ(baseDir, "./bin/thunderdbd"), FJ(testWorkingDir, "./node_0/config.yaml"), "leader", false)
	go RunServer(FJ(baseDir, "./bin/thunderdbd"), FJ(testWorkingDir, "./node_1/config.yaml"), "follower1", false)
	go RunServer(FJ(baseDir, "./bin/thunderdbd"), FJ(testWorkingDir, "./node_2/config.yaml"), "follower2", false)

	time.Sleep(5 * time.Second)

	var err error
	conf.GConf, err = conf.LoadConfig(FJ(testWorkingDir, "./node_c/config.yaml"))
	if err != nil {
		log.Fatalf("load config from %s failed: %s", configFile, err)
	}
	kms.InitBP()
	log.Debugf("config:\n%#v", conf.GConf)
	conf.GConf.GenerateKeyPair = false

	nodeID := conf.GConf.ThisNodeID

	var idx int
	for i, n := range (*conf.GConf.KnownNodes)[:] {
		if n.ID == nodeID {
			idx = i
			break
		}
	}

	rootPath := conf.GConf.WorkingRoot
	pubKeyStorePath := filepath.Join(rootPath, conf.GConf.PubKeyStoreFile)
	privateKeyPath := filepath.Join(rootPath, conf.GConf.PrivateKeyFile)

	// read master key
	var masterKey []byte

	err = kms.InitLocalKeyPair(privateKeyPath, masterKey)
	if err != nil {
		log.Errorf("init local key pair failed: %s", err)
		return
	}

	(*conf.GConf.KnownNodes)[idx].PublicKey, err = kms.GetLocalPublicKey()
	if err != nil {
		log.Errorf("get local public key failed: %s", err)
		return
	}

	// init nodes
	log.Infof("init peers")
	_, _, _, err = initNodePeers(nodeID, pubKeyStorePath)
	if err != nil {
		return
	}

	//connPool := rpc.newSessionPool(rpc.DefaultDialer)
	//// do client request
	//if err = clientRequest(connPool, clientOperation, ""); err != nil {
	//	return
	//}

	leaderNodeID := kms.BP.NodeID
	var conn net.Conn
	var RPCClient *rpc.Client

	if conn, err = rpc.DialToNode(leaderNodeID, rpc.GetSessionPoolInstance()); err != nil {
		return
	}
	if RPCClient, err = rpc.InitClientConn(conn); err != nil {
		return
	}
	client := rpc.NewCaller()

	var reqType = "FindNeighbor"
	nodePayload := proto.NewNode()
	nodePayload.InitNodeCryptoInfo(100 * time.Millisecond)
	nodePayload.Addr = "nodePayloadAddr"

	reqFindNeighbor := &proto.FindNeighborReq{
		NodeID: proto.NodeID(nodePayload.ID),
		Count:  1,
	}
	respFindNeighbor := new(proto.FindNeighborResp)
	log.Debugf("req %s: %v", reqType, reqFindNeighbor)
	b.Run("benchmark "+reqType, func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err = client.CallNode(leaderNodeID, "DHT."+reqType, reqFindNeighbor, respFindNeighbor)
			if err != nil {
				log.Fatal(err)
			}
			//log.Debugf("resp2 %s: %v", reqType, respA)
		}
	})

	reqType = "Ping"
	reqPing := &proto.PingReq{
		Node: *nodePayload,
	}
	respPing := new(proto.PingResp)
	b.Run("benchmark "+reqType, func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err = client.CallNode(leaderNodeID, "DHT."+reqType, reqPing, respPing)
			if err != nil {
				log.Fatal(err)
			}
		}
	})
	log.Debugf("respPing %s: %##v", reqType, respPing)

	reqType = "FindNode"
	reqFN := &proto.FindNodeReq{
		NodeID: nodePayload.ID,
	}
	respFN := new(proto.FindNodeResp)
	b.Run("benchmark "+reqType, func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err = RPCClient.Call("DHT."+reqType, reqFN, respFN)
			if err != nil || respFN.Node.Addr != "nodePayloadAddr" {
				log.Fatal(err)
			}
		}
	})
	log.Debugf("respFN %s: %##v", reqType, respFN.Node)
}
