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

package mux

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func TestSessionPool_SessionBroken(t *testing.T) {
	log.SetLevel(log.FatalLevel)
	var err error

	conf.GConf, err = conf.LoadConfig(FJ(testWorkingDir, "./leak/client.yaml"))
	if err != nil {
		t.Errorf("load config from %s failed: %s", FJ(testWorkingDir, "./leak/client.yaml"), err)
	}
	log.Debugf("GConf: %##v", conf.GConf)
	utils.RemoveAll(conf.GConf.PubKeyStoreFile + "*")
	_ = os.Remove(FJ(testWorkingDir, "./leak/leader/dht.db"))
	_ = os.Remove(FJ(testWorkingDir, "./leak/leader/dht.db-shm"))
	_ = os.Remove(FJ(testWorkingDir, "./leak/leader/dht.db-wal"))
	_ = os.Remove(FJ(testWorkingDir, "./leak/kayak.db"))
	_ = os.RemoveAll(FJ(testWorkingDir, "./leak/kayak.ldb"))

	leader, err := utils.RunCommandNB(
		FJ(baseDir, "./bin/cqld"),
		[]string{"-config", FJ(testWorkingDir, "./leak/leader.yaml")},
		"leak", testWorkingDir, logDir, false,
	)

	defer func() {
		_ = leader.LogFD.Close()
		_ = leader.Cmd.Process.Signal(syscall.SIGKILL)
	}()

	log.Debugf("leader pid %d", leader.Cmd.Process.Pid)
	time.Sleep(5 * time.Second)

	route.InitKMS(conf.GConf.PubKeyStoreFile)
	var masterKey []byte

	err = kms.InitLocalKeyPair(conf.GConf.PrivateKeyFile, masterKey)
	if err != nil {
		t.Errorf("init local key pair failed: %s", err)
		return
	}

	leaderNodeID := kms.BP.NodeID
	thisClient, _ := kms.GetNodeInfo(conf.GConf.ThisNodeID)

	var reqType string
	caller := NewCaller()

	reqType = "Ping"
	reqPing := &proto.PingReq{
		Node: *thisClient,
	}
	respPing := new(proto.PingResp)
	err = caller.CallNode(leaderNodeID, "DHT."+reqType, reqPing, respPing)
	log.Debugf("respPing %s: %##v", reqType, respPing)
	if err != nil {
		t.Error(err)
	}

	reqType = "Ping"
	reqPing = &proto.PingReq{
		Node: *thisClient,
	}
	respPing = new(proto.PingResp)
	err = caller.CallNode(leaderNodeID, "DHT."+reqType, reqPing, respPing)
	log.Debugf("respPing %s: %##v", reqType, respPing)
	if err != nil {
		t.Error(err)
	}

	reqType = "FindNode"
	reqFN := &proto.FindNodeReq{
		ID: thisClient.ID,
	}
	respFN := new(proto.FindNodeResp)
	err = caller.CallNode(leaderNodeID, "DHT."+reqType, reqFN, respFN)
	log.Debugf("respFN %s: %##v", reqType, respFN.Node)
	if err != nil {
		t.Error(err)
	}

	pool := GetSessionPoolInstance()
	sess, _ := pool.getSession(leaderNodeID)
	log.Debugf("session for %s, %#v", leaderNodeID, sess)
	_ = sess.Close()

	reqType = "FindNode"
	reqFN = &proto.FindNodeReq{
		ID: thisClient.ID,
	}
	respFN = new(proto.FindNodeResp)
	err = caller.CallNode(leaderNodeID, "DHT."+reqType, reqFN, respFN)
	log.Debugf("respFN %s: %##v", reqType, respFN.Node)
	if err != nil {
		t.Error(err)
	}
}
