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

package transport

import (
	"context"
	"crypto/rand"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"
)

type mockRes struct {
	nodeID     proto.NodeID
	service    *ETLSTransportService
	transport  *ETLSTransport
	server     *rpc.Server
	listenAddr string
}

func testWithNewNode() (mock *mockRes, err error) {
	// mock etls transport without kms server
	mock = &mockRes{}
	addr := "127.0.0.1:0"

	// random node id
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	mock.nodeID = proto.NodeID(hash.THashH(randBytes).String())
	kms.SetLocalNodeIDNonce(mock.nodeID.ToRawNodeID().CloneBytes(), &cpuminer.Uint256{})
	mock.service = &ETLSTransportService{}
	mock.transport = NewETLSTransport(&ETLSTransportConfig{
		NodeID:           mock.nodeID,
		TransportID:      "test",
		TransportService: mock.service,
		ServiceName:      "Kayak",
	})
	mock.server, err = rpc.NewServerWithService(rpc.ServiceMap{"Kayak": mock.service})
	if err != nil {
		return
	}
	_, testFile, _, _ := runtime.Caller(0)
	privKeyPath := filepath.Join(filepath.Dir(testFile), "../../test/node_standalone/private.key")
	if err = mock.server.InitRPCServer(addr, privKeyPath, []byte("")); err != nil {
		return
	}
	mock.listenAddr = mock.server.Listener.Addr().String()
	route.SetNodeAddrCache(mock.nodeID.ToRawNodeID(), mock.listenAddr)
	var nonce *cpuminer.Uint256
	if nonce, err = kms.GetLocalNonce(); err != nil {
		return
	}
	var pubKey *asymmetric.PublicKey
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	if err = kms.SetPublicKey(mock.nodeID, *nonce, pubKey); err != nil {
		return
	}

	log.Infof("fake node with node id: %v", mock.nodeID)
	return
}

func initKMS() (err error) {
	var f *os.File
	f, err = ioutil.TempFile("", "keystore_")
	f.Close()
	os.Remove(f.Name())
	route.InitKMS(f.Name())

	// flag as test
	kms.Unittest = true

	return
}

func TestETLSTransport(t *testing.T) {
	Convey("full test", t, FailureContinues, func(c C) {
		var err error

		err = initKMS()
		So(err, ShouldBeNil)

		mock1, err := testWithNewNode()
		So(err, ShouldBeNil)
		mock2, err := testWithNewNode()
		So(err, ShouldBeNil)

		var wgServer, wgRequest sync.WaitGroup

		// start server
		wgServer.Add(1)
		go func() {
			defer wgServer.Done()
			mock1.server.Serve()
		}()

		wgServer.Add(1)
		go func() {
			defer wgServer.Done()
			mock2.server.Serve()
		}()

		// init transport
		err = mock1.transport.Init()
		So(err, ShouldBeNil)
		err = mock2.transport.Init()
		So(err, ShouldBeNil)

		testLog := testLogFixture([]byte("test request"))

		// make request issuer as node 1
		kms.SetLocalNodeIDNonce(mock1.nodeID.ToRawNodeID().CloneBytes(), &cpuminer.Uint256{})

		wgRequest.Add(1)
		go func() {
			defer wgRequest.Done()
			res, err := mock1.transport.Request(context.Background(), mock2.nodeID, "test method", testLog)
			c.So(err, ShouldBeNil)
			c.So(res, ShouldResemble, []byte("test response"))
		}()

		wgRequest.Add(1)
		go func() {
			defer wgRequest.Done()
			select {
			case req := <-mock2.transport.Process():
				c.So(req.GetLog(), ShouldResemble, testLog)
				c.So(req.GetMethod(), ShouldEqual, "test method")
				c.So(req.GetPeerNodeID(), ShouldEqual, mock1.nodeID)
				req.SendResponse([]byte("test response"), nil)
			}
		}()

		wgRequest.Wait()

		// shutdown transport
		err = mock1.transport.Shutdown()
		So(err, ShouldBeNil)
		err = mock2.transport.Shutdown()
		So(err, ShouldBeNil)

		// stop
		mock1.server.Listener.Close()
		mock1.server.Stop()
		mock2.server.Listener.Close()
		mock2.server.Stop()

		wgServer.Wait()
	})
}

func TestETLSIntegration(t *testing.T) {
	type createMockRes struct {
		runner    *kayak.TwoPCRunner
		transport *ETLSTransport
		worker    *MockWorker
		config    *kayak.TwoPCConfig
		runtime   *kayak.Runtime
		etlsMock  *mockRes
	}

	// create mock returns basic arguments to prepare for a server
	createMock := func(etlsMock *mockRes, peers *kayak.Peers) (res *createMockRes) {
		res = &createMockRes{}
		log.SetLevel(log.FatalLevel)
		d, _ := ioutil.TempDir("", "kayak_test")

		// etls mock res
		res.etlsMock = etlsMock
		// runner instance
		res.runner = kayak.NewTwoPCRunner()
		// transport for this instance
		res.transport = res.etlsMock.transport
		// underlying worker
		res.worker = &MockWorker{}
		// runner config including timeout settings, commit log storage, local server id
		res.config = &kayak.TwoPCConfig{
			RuntimeConfig: kayak.RuntimeConfig{
				RootDir:        d,
				LocalID:        etlsMock.nodeID,
				Runner:         res.runner,
				Transport:      res.transport,
				ProcessTimeout: time.Millisecond * 800,
			},
			Storage: res.worker,
		}
		res.runtime, _ = kayak.NewRuntime(res.config, peers)
		go func() {
			res.etlsMock.server.Serve()
		}()
		return
	}
	// cleanup log storage after execution
	cleanupDir := func(c *createMockRes) {
		os.RemoveAll(c.config.RuntimeConfig.RootDir)
	}

	Convey("integration test", t, FailureContinues, func(c C) {
		var err error

		err = initKMS()
		So(err, ShouldBeNil)

		lNodeEtls, err := testWithNewNode()
		So(err, ShouldBeNil)
		f1NodeEtls, err := testWithNewNode()
		So(err, ShouldBeNil)
		f2NodeEtls, err := testWithNewNode()
		So(err, ShouldBeNil)

		// peers is a simple 3-node peer configuration
		peers := testPeersFixture(1, []*kayak.Server{
			{
				Role: proto.Leader,
				ID:   lNodeEtls.nodeID,
			},
			{
				Role: proto.Follower,
				ID:   f1NodeEtls.nodeID,
			},
			{
				Role: proto.Follower,
				ID:   f2NodeEtls.nodeID,
			},
		})

		lMock := createMock(lNodeEtls, peers)
		f1Mock := createMock(f1NodeEtls, peers)
		f2Mock := createMock(f2NodeEtls, peers)
		defer cleanupDir(lMock)
		defer cleanupDir(f1Mock)
		defer cleanupDir(f2Mock)

		// init
		err = lMock.runtime.Init()
		So(err, ShouldBeNil)
		err = f1Mock.runtime.Init()
		So(err, ShouldBeNil)
		err = f2Mock.runtime.Init()
		So(err, ShouldBeNil)

		// payload to send
		testPayload := []byte("test data")

		// make request issuer as leader node
		kms.SetLocalNodeIDNonce(lMock.config.LocalID.ToRawNodeID().CloneBytes(), &cpuminer.Uint256{})

		// underlying worker mock, prepare/commit/rollback with be received the decoded data
		callOrder := &CallCollector{}
		f1Mock.worker.On("Prepare", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("prepare")
		})
		f2Mock.worker.On("Prepare", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("prepare")
		})
		f1Mock.worker.On("Commit", mock.Anything, testPayload).
			Return(nil, nil).Run(func(args mock.Arguments) {
			callOrder.Append("commit")
		})
		f2Mock.worker.On("Commit", mock.Anything, testPayload).
			Return(nil, nil).Run(func(args mock.Arguments) {
			callOrder.Append("commit")
		})
		lMock.worker.On("Prepare", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("prepare")
		})
		lMock.worker.On("Commit", mock.Anything, testPayload).
			Return(nil, nil).Run(func(args mock.Arguments) {
			callOrder.Append("commit")
		})

		// process the encoded data
		_, _, err = lMock.runtime.Apply(testPayload)
		So(err, ShouldBeNil)
		So(callOrder.Get(), ShouldResemble, []string{
			"prepare",
			"prepare",
			"prepare",
			"commit",
			"commit",
			"commit",
		})

		// process the encoded data again
		callOrder.Reset()
		_, _, err = lMock.runtime.Apply(testPayload)
		So(err, ShouldBeNil)
		So(callOrder.Get(), ShouldResemble, []string{
			"prepare",
			"prepare",
			"prepare",
			"commit",
			"commit",
			"commit",
		})

		// shutdown
		lMock.runtime.Shutdown()
		f1Mock.runtime.Shutdown()
		f2Mock.runtime.Shutdown()

		// stop server
		lNodeEtls.server.Listener.Close()
		f1NodeEtls.server.Listener.Close()
		f2NodeEtls.server.Listener.Close()
		lNodeEtls.server.Stop()
		f1NodeEtls.server.Stop()
		f2NodeEtls.server.Stop()
	})
}
