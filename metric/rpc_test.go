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

package metric

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const PubKeyStorePath = "./public.keystore"

func TestCollectClient_UploadMetrics(t *testing.T) {
	defer utils.RemoveAll(PubKeyStorePath + "*")
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	masterKey := []byte("abc")

	cc := NewCollectClient()
	cs := NewCollectServer()

	server, err := rpc.NewServerWithService(rpc.ServiceMap{MetricServiceName: cs})
	if err != nil {
		log.Fatal(err)
	}

	route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), false)
	server.InitRPCServer(addr, "../keys/test.key", masterKey)
	go server.Serve()

	publicKey, err := kms.GetLocalPublicKey()
	nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	serverNodeID := proto.NodeID(nonce.Hash.String())
	kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)
	kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	route.SetNodeAddrCache(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	Convey("get metric and upload by RPC", t, func() {
		err = cc.UploadMetrics(serverNodeID)
		v, ok := cs.NodeMetric.Load(serverNodeID)
		So(ok, ShouldBeTrue)
		//log.Debugf("NodeMetric：%#v", v)

		m, _ := v.(SimpleMetricMap)
		mfb, err := cc.GatherMetricBytes()
		So(err, ShouldBeNil)
		So(len(m), ShouldEqual, len(mfb))
		So(len(m), ShouldBeGreaterThan, 2)
	})

	Convey("get metric and upload by simply called without node id", t, func() {
		req := &proto.UploadMetricsReq{
			MFBytes:  nil,
			Envelope: proto.Envelope{},
		}
		err = cs.UploadMetrics(req, &proto.UploadMetricsResp{})
		So(err, ShouldNotBeNil)
	})
}
