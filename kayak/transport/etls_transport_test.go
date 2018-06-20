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

package transport

import (
	"context"
	"crypto/rand"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/mock"
	"gitlab.com/thunderdb/ThunderDB/crypto/etls"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
)

var (
	pass = "DU>p~[/dd2iImUs*"
)

type mockRes struct {
	nodeID     proto.NodeID
	service    *ETLSTransportService
	transport  *ETLSTransport
	server     *rpc.Server
	listenAddr string
}

func cipherHandler(conn net.Conn) (cryptoConn *etls.CryptoConn, err error) {
	nodeIDBuf := make([]byte, hash.HashBSize)
	rCount, err := conn.Read(nodeIDBuf)
	if err != nil || rCount != hash.HashBSize {
		return
	}
	cipher := etls.NewCipher([]byte(pass))
	h, _ := hash.NewHash(nodeIDBuf)
	nodeID := &proto.RawNodeID{
		Hash: *h,
	}

	cryptoConn = etls.NewConn(conn, cipher, nodeID)
	return
}

func getNodeDialer(reqNodeID proto.NodeID, nodeMap *sync.Map) ETLSRPCClientBuilder {
	return func(ctx context.Context, nodeID proto.NodeID) (client *rpc.Client, err error) {
		cipher := etls.NewCipher([]byte(pass))

		var ok bool
		var addr interface{}

		if addr, ok = nodeMap.Load(nodeID); !ok {
			return nil, errors.New("could not connect to " + string(nodeID))
		}

		var conn net.Conn
		conn, err = net.Dial("tcp", addr.(string))
		if err != nil {
			return
		}

		// convert node id to raw node id
		h, _ := hash.NewHashFromStr(string(reqNodeID))
		rawNodeID := &proto.RawNodeID{Hash: *h}
		wCount, err := conn.Write(rawNodeID.Hash[:])
		if err != nil || wCount != hash.HashBSize {
			return
		}

		cryptConn := etls.NewConn(conn, cipher, rawNodeID)
		return rpc.InitClientConn(cryptConn)
	}
}

func testWithNewNode(nodeMap *sync.Map) (mock *mockRes, err error) {
	// mock etls transport without kms server
	mock = &mockRes{}
	addr := "127.0.0.1:0"
	var l *etls.CryptoListener
	l, err = etls.NewCryptoListener("tcp", addr, cipherHandler)
	if err != nil {
		return
	}
	mock.listenAddr = l.Addr().String()

	// random node id
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	mock.nodeID = proto.NodeID(hash.THashH(randBytes).String())
	mock.service = &ETLSTransportService{}
	mock.transport, _ = NewETLSTransport(&ETLSTransportConfig{
		NodeID:           mock.nodeID,
		TransportID:      "test",
		TransportService: mock.service,
		ServiceName:      "Kayak",
		ClientBuilder:    getNodeDialer(mock.nodeID, nodeMap),
	})
	mock.server, err = rpc.NewServerWithService(rpc.ServiceMap{"Kayak": mock.service})
	if err != nil {
		return
	}
	mock.server.SetListener(l)
	nodeMap.Store(mock.nodeID, mock.listenAddr)
	return
}

func TestETLSTransport(t *testing.T) {
	Convey("full test", t, FailureContinues, func(c C) {
		var nodeMap sync.Map
		var err error

		mock1, err := testWithNewNode(&nodeMap)
		So(err, ShouldBeNil)
		mock2, err := testWithNewNode(&nodeMap)
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

		testLog := testLogFixture([]byte("test request"))

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

	// test node map/routine map
	var nodeMap sync.Map

	// codec to encode/decode data in kayak.Log structure
	mockLogCodec := &MockLogCodec{}

	// create mock returns basic arguments to prepare for a server
	createMock := func(etlsMock *mockRes, peers *kayak.Peers) (res *createMockRes) {
		res = &createMockRes{}
		logger := log.New()
		logger.SetLevel(log.FatalLevel)
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
				Logger:         logger,
			},
			LogCodec:        mockLogCodec,
			Storage:         res.worker,
			PrepareTimeout:  time.Millisecond * 200,
			CommitTimeout:   time.Millisecond * 200,
			RollbackTimeout: time.Millisecond * 200,
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

		lNodeEtls, err := testWithNewNode(&nodeMap)
		So(err, ShouldBeNil)
		f1NodeEtls, err := testWithNewNode(&nodeMap)
		So(err, ShouldBeNil)
		f2NodeEtls, err := testWithNewNode(&nodeMap)
		So(err, ShouldBeNil)

		// peers is a simple 3-node peer configuration
		peers := testPeersFixture(1, []*kayak.Server{
			{
				Role: kayak.Leader,
				ID:   lNodeEtls.nodeID,
			},
			{
				Role: kayak.Follower,
				ID:   f1NodeEtls.nodeID,
			},
			{
				Role: kayak.Follower,
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
		testPayload := "test data"
		testData, _ := mockLogCodec.Encode(testPayload)

		// underlying worker mock, prepare/commit/rollback with be received the decoded data
		callOrder := &CallCollector{}
		f1Mock.worker.On("Prepare", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("f_prepare")
		})
		f2Mock.worker.On("Prepare", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("f_prepare")
		})
		f1Mock.worker.On("Commit", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("f_commit")
		})
		f2Mock.worker.On("Commit", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("f_commit")
		})
		lMock.worker.On("Prepare", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("l_prepare")
		})
		lMock.worker.On("Commit", mock.Anything, testPayload).
			Return(nil).Run(func(args mock.Arguments) {
			callOrder.Append("l_commit")
		})

		// process the encoded data
		err = lMock.runtime.Apply(testData)
		So(err, ShouldBeNil)
		So(callOrder.Get(), ShouldResemble, []string{
			"f_prepare",
			"f_prepare",
			"l_prepare",
			"f_commit",
			"f_commit",
			"l_commit",
		})

		// process the encoded data again
		callOrder.Reset()
		err = lMock.runtime.Apply(testData)
		So(err, ShouldBeNil)
		So(callOrder.Get(), ShouldResemble, []string{
			"f_prepare",
			"f_prepare",
			"l_prepare",
			"f_commit",
			"f_commit",
			"l_commit",
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
