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

package metric

import (
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/consistent"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
)

const PubKeyStorePath = "./public.keystore"

func TestCollectClient_UploadMetrics(t *testing.T) {
	defer os.Remove(PubKeyStorePath)
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	masterKey := []byte("abc")

	cc := NewCollectClient()

	cs := NewCollectServer()

	server, err := rpc.NewServerWithService(rpc.ServiceMap{MetricServiceName: cs})
	if err != nil {
		log.Fatal(err)
	}

	route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)
	server.InitRPCServer(addr, "../keys/test.key", masterKey)
	go server.Serve()

	publicKey, err := kms.GetLocalPublicKey()
	nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	serverNodeID := proto.NodeID(nonce.Hash.String())
	kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)
	kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	route.SetNodeAddr(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	time.Sleep(time.Second)
	err = cc.UploadMetrics(serverNodeID, nil)
	v, _ := cs.NodeMetric.Load(serverNodeID)
	log.Debugf("NodeMetric：%#v", v)

	m, _ := v.(metricMap)
	log.Debugf("NodeMetric：%#v", m["node_scrape_collector_success"].Metric)
}
