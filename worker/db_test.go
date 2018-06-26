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

package worker

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/thunderdb/ThunderDB/consistent"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	ka "gitlab.com/thunderdb/ThunderDB/kayak/api"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
)

func TestSingleDatabase(t *testing.T) {
	// init as single node database
	Convey("test database", t, func() {
		var err error
		var server *rpc.Server
		var cleanup func()
		cleanup, server, err = initNode()
		So(err, ShouldBeNil)

		defer cleanup()

		var rootDir string
		rootDir, err = ioutil.TempDir("", "db_test_")
		So(err, ShouldBeNil)

		defer os.RemoveAll(rootDir)

		// create mux service
		service := ka.NewMuxService("DBKayak", server)

		// create peers
		var peers *kayak.Peers
		peers, err = getPeers()
		So(err, ShouldBeNil)

		// create file
		cfg := &Config{
			DatabaseID:      "TEST",
			DataDir:         rootDir,
			MuxService:      service,
			MaxWriteTimeGap: time.Duration(5 * time.Second),
		}

		// create storage
		storePath := filepath.Join(rootDir, "database.db")
		var st *storage.Storage
		st, err = storage.New(storePath)
		So(err, ShouldBeNil)

		// create database
		var db *Database
		db, err = NewDatabase(cfg, peers, st)
		So(err, ShouldBeNil)

		Convey("test read write", func() {
			// test write query
			var writeQuery *Request
			var res *Response
			writeQuery, err = buildQuery(WriteQuery, 1, 1, []string{
				"create table test (test int)",
				"insert into test values(1)",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)
			So(res.Header.RowCount, ShouldEqual, 0)

			// test select query
			var readQuery *Request
			readQuery, err = buildQuery(ReadQuery, 1, 2, []string{
				"select * from test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Columns, ShouldResemble, []string{"test"})
			So(res.Payload.DeclTypes, ShouldResemble, []string{"int"})
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values[0], ShouldEqual, 1)

			err = db.Shutdown()
			So(err, ShouldBeNil)
		})

		Convey("test invalid request", func() {
			var writeQuery *Request
			var res *Response
			writeQuery, err = buildQuery(WriteQuery, 1, 1, []string{
				"create table test (test int)",
				"insert into test values(1)",
			})
			So(err, ShouldBeNil)

			// request once
			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)
			So(res.Header.RowCount, ShouldEqual, 0)

			// request again with same sequence
			writeQuery, err = buildQuery(WriteQuery, 1, 1, []string{
				"insert into test values(2)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request again with low sequence
			writeQuery, err = buildQuery(WriteQuery, 1, 0, []string{
				"insert into test values(3)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request with invalid timestamp
			writeQuery, err = buildQueryWithTimeShift(WriteQuery, 1, 2, time.Second*100, []string{
				"insert into test values(4)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request with invalid timestamp
			writeQuery, err = buildQueryWithTimeShift(WriteQuery, 1, 2, -time.Second*100, []string{
				"insert into test values(5)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request with different connection id
			writeQuery, err = buildQuery(WriteQuery, 2, 1, []string{
				"insert into test values(6)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)

			// read query, test records
			var readQuery *Request
			readQuery, err = buildQuery(ReadQuery, 1, 2, []string{
				"select * from test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(2))
			So(res.Payload.Columns, ShouldResemble, []string{"test"})
			So(res.Payload.DeclTypes, ShouldResemble, []string{"int"})
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(len(res.Payload.Rows), ShouldEqual, 2)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values[0], ShouldEqual, 1)
			So(res.Payload.Rows[1].Values[0], ShouldEqual, 6)

			err = db.Shutdown()
			So(err, ShouldBeNil)
		})

		Convey("corner case", func() {
			var req *Request
			var err error
			req, err = buildQuery(-1, 1, 1, []string{
				"create table test (test int)",
			})
			So(err, ShouldBeNil)
			_, err = db.Query(req)
			So(err, ShouldNotBeNil)

			var writeQuery *Request
			var res *Response
			writeQuery, err = buildQuery(WriteQuery, 1, 1, []string{
				"create table test (test int)",
			})
			So(err, ShouldBeNil)
			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)

			// read query, test records
			var readQuery *Request
			readQuery, err = buildQuery(ReadQuery, 1, 2, []string{
				"select * from test",
			})
			So(err, ShouldBeNil)
			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(0))
			So(res.Payload.Columns, ShouldResemble, []string{"test"})
			So(res.Payload.DeclTypes, ShouldResemble, []string{"int"})
			So(res.Payload.Rows, ShouldBeEmpty)

			// write query, test failed
			writeQuery, err = buildQuery(WriteQuery, 1, 3, []string{
				"insert into test2 values(1)", // table should not exists
			})
			So(err, ShouldBeNil)
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// read query, test dynamic fields
			readQuery, err = buildQuery(ReadQuery, 1, 4, []string{
				"select 1 as test",
			})
			So(err, ShouldBeNil)
			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Columns, ShouldResemble, []string{"test"})
			So(res.Payload.DeclTypes, ShouldResemble, []string{""})
			So(res.Payload.Rows, ShouldNotBeEmpty)

			// test ack
			var ack *Ack
			ack, err = buildAck(res)
			So(err, ShouldBeNil)

			err = db.Ack(ack)
			So(err, ShouldBeNil)
		})
	})
}

func TestDatabaseRecycle(t *testing.T) {
	defer leaktest.Check(t)()

	// test init/shutdown/destroy
	// test goroutine status
	Convey("test init destroy", t, func() {
		var err error
		var server *rpc.Server
		var cleanup func()
		cleanup, server, err = initNode()
		So(err, ShouldBeNil)

		defer cleanup()

		var rootDir string
		rootDir, err = ioutil.TempDir("", "db_test_")
		So(err, ShouldBeNil)

		// create mux service
		service := ka.NewMuxService("DBKayak", server)

		// create peers
		var peers *kayak.Peers
		peers, err = getPeers()
		So(err, ShouldBeNil)

		// create file
		cfg := &Config{
			DatabaseID:      "TEST",
			DataDir:         rootDir,
			MuxService:      service,
			MaxWriteTimeGap: time.Duration(5 * time.Second),
		}

		// create storage
		storePath := filepath.Join(rootDir, "database.db")
		var st *storage.Storage
		st, err = storage.New(storePath)
		So(err, ShouldBeNil)

		defer st.Close()

		// create database
		var db *Database
		db, err = NewDatabase(cfg, peers, st)
		So(err, ShouldBeNil)

		// do some query
		var writeQuery *Request
		var res *Response
		writeQuery, err = buildQuery(WriteQuery, 1, 1, []string{
			"create table test (test int)",
			"insert into test values(1)",
		})
		So(err, ShouldBeNil)

		res, err = db.Query(writeQuery)
		So(err, ShouldBeNil)
		err = res.Verify()
		So(err, ShouldBeNil)
		So(res.Header.RowCount, ShouldEqual, 0)

		// test select query
		var readQuery *Request
		readQuery, err = buildQuery(ReadQuery, 1, 2, []string{
			"select * from test",
		})
		So(err, ShouldBeNil)

		res, err = db.Query(readQuery)
		So(err, ShouldBeNil)
		err = res.Verify()
		So(err, ShouldBeNil)

		So(res.Header.RowCount, ShouldEqual, uint64(1))
		So(res.Payload.Columns, ShouldResemble, []string{"test"})
		So(res.Payload.DeclTypes, ShouldResemble, []string{"int"})
		So(res.Payload.Rows, ShouldNotBeEmpty)
		So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
		So(res.Payload.Rows[0].Values[0], ShouldEqual, 1)

		// destroy
		err = db.Destroy()
		So(err, ShouldBeNil)
		_, err = os.Stat(rootDir)
		So(err, ShouldNotBeNil)
	})
}

func buildAck(res *Response) (ack *Ack, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = getNodeID(); err != nil {
		return
	}

	// get private/public key
	var pubKey *asymmetric.PublicKey
	var privateKey *asymmetric.PrivateKey

	if privateKey, pubKey, err = getKeys(); err != nil {
		return
	}

	ack = &Ack{
		Header: SignedAckHeader{
			AckHeader: AckHeader{
				Response:  res.Header,
				NodeID:    nodeID,
				Timestamp: getLocalTime(),
			},
			Signee: pubKey,
		},
	}

	err = ack.Sign(privateKey)

	return
}

func buildQuery(queryType QueryType, connID uint64, seqNo uint64, queries []string) (query *Request, err error) {
	return buildQueryWithTimeShift(queryType, connID, seqNo, time.Duration(0), queries)
}

func buildQueryWithTimeShift(queryType QueryType, connID uint64, seqNo uint64, timeShift time.Duration, queries []string) (query *Request, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = getNodeID(); err != nil {
		return
	}

	// get private/public key
	var pubKey *asymmetric.PublicKey
	var privateKey *asymmetric.PrivateKey

	if privateKey, pubKey, err = getKeys(); err != nil {
		return
	}

	tm := getLocalTime()
	tm = tm.Add(-timeShift)

	query = &Request{
		Header: SignedRequestHeader{
			RequestHeader: RequestHeader{
				QueryType:    queryType,
				NodeID:       nodeID,
				ConnectionID: connID,
				SeqNo:        seqNo,
				Timestamp:    tm,
			},
			Signee: pubKey,
		},
		Payload: RequestPayload{
			Queries: queries,
		},
	}

	err = query.Sign(privateKey)

	return
}

func getLocalTime() time.Time {
	return time.Now().UTC()
}

func getPeers() (peers *kayak.Peers, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = getNodeID(); err != nil {
		return
	}

	// get private/public key
	var pubKey *asymmetric.PublicKey
	var privateKey *asymmetric.PrivateKey

	if privateKey, pubKey, err = getKeys(); err != nil {
		return
	}

	// generate peers and sign
	server := &kayak.Server{
		Role:   kayak.Leader,
		ID:     nodeID,
		PubKey: pubKey,
	}
	peers = &kayak.Peers{
		Term:    1,
		Leader:  server,
		Servers: []*kayak.Server{server},
		PubKey:  pubKey,
	}
	err = peers.Sign(privateKey)
	return
}

func getNodeID() (nodeID proto.NodeID, err error) {
	var rawNodeID []byte
	if rawNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}
	var h *hash.Hash
	if h, err = hash.NewHash(rawNodeID); err != nil {
		return
	}
	nodeID = proto.NodeID(h.String())
	return
}

func getKeys() (privKey *asymmetric.PrivateKey, pubKey *asymmetric.PublicKey, err error) {
	// get public key
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	// get private key
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	return
}

func initNode() (cleanupFunc func(), server *rpc.Server, err error) {
	var d string
	if d, err = ioutil.TempDir("", "db_test_"); err != nil {
		return
	}

	var dht *route.DHTService

	// init dht
	pubKeyStoreFile := filepath.Join(d, "pubkey.store")
	dht, err = route.NewDHTService(pubKeyStoreFile, new(consistent.KMSStorage), true)
	if err != nil {
		return
	}

	// init rpc
	if server, err = rpc.NewServerWithService(rpc.ServiceMap{"DHT": dht}); err != nil {
		return
	}

	// init private key
	masterKey := []byte("abc")
	privateKeyFile := filepath.Join(d, "private.key")
	addr := "127.0.0.1:0"
	server.InitRPCServer(addr, privateKeyFile, masterKey)

	// get public key
	var pubKey *asymmetric.PublicKey
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	nonce := asymmetric.GetPubKeyNonce(pubKey, 10, 100*time.Millisecond, nil)
	serverNodeID := proto.NodeID(nonce.Hash.String())
	kms.SetPublicKey(serverNodeID, nonce.Nonce, pubKey)

	kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	route.SetNodeAddr(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	// start server
	go server.Serve()

	cleanupFunc = func() {
		os.RemoveAll(d)
		server.Listener.Close()
		server.Stop()
	}

	return
}
