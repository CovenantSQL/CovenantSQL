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

package worker

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var rootHash = hash.Hash{}

const PubKeyStorePath = "./public.keystore"

func TestSingleDatabase(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	// init as single node database
	Convey("test database", t, func() {
		var err error
		var server *rpc.Server
		var cleanup func()
		cleanup, server, err = initNode()
		So(err, ShouldBeNil)

		var rootDir string
		rootDir, err = ioutil.TempDir("", "db_test_")
		So(err, ShouldBeNil)

		// create mux service
		kayakMuxService, err := NewDBKayakMuxService("DBKayak", server)
		So(err, ShouldBeNil)

		chainMuxService, err := sqlchain.NewMuxService("sqlchain", server)
		So(err, ShouldBeNil)

		// create peers
		var peers *proto.Peers
		peers, err = getPeers(1)
		So(err, ShouldBeNil)

		// create file
		cfg := &DBConfig{
			DatabaseID:       "00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9",
			DataDir:          rootDir,
			KayakMux:         kayakMuxService,
			ChainMux:         chainMuxService,
			MaxWriteTimeGap:  time.Second * 5,
			UpdateBlockCount: 2,
		}

		// create genesis block
		var block *types.Block
		block, err = types.CreateRandomBlock(rootHash, true)
		So(err, ShouldBeNil)

		// create database
		var db *Database
		db, err = NewDatabase(cfg, peers, block)
		So(err, ShouldBeNil)

		Convey("test query rewrite", func() {
			// test query rewrite
			var writeQuery *types.Request
			var res *types.Response
			writeQuery, err = buildQuery(types.WriteQuery, 1, 1, []string{
				"create table test (col1 int, col2 string)",
				"create index test_index on test (col1)",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)

			// test show tables query
			var readQuery *types.Request
			readQuery, err = buildQuery(types.ReadQuery, 1, 2, []string{
				"show tables",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values[0], ShouldResemble, "test")

			// test show full tables query
			readQuery, err = buildQuery(types.ReadQuery, 1, 3, []string{
				"show full tables",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values[0], ShouldResemble, "test")

			// test show create table
			readQuery, err = buildQuery(types.ReadQuery, 1, 4, []string{
				"show create table test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			byteStr, isStr := res.Payload.Rows[0].Values[0].(string)
			So(isStr, ShouldBeTrue)
			So(strings.ToUpper(string(byteStr)), ShouldContainSubstring, "CREATE")

			// test show table
			readQuery, err = buildQuery(types.ReadQuery, 1, 5, []string{
				"show table test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(2))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldHaveLength, 6)
			So(res.Payload.Rows[0].Values[1], ShouldResemble, "col1")
			So(res.Payload.Rows[1].Values, ShouldHaveLength, 6)
			So(res.Payload.Rows[1].Values[1], ShouldResemble, "col2")

			// test desc table
			readQuery, err = buildQuery(types.ReadQuery, 1, 6, []string{
				"desc test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(2))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldHaveLength, 6)
			So(res.Payload.Rows[0].Values[1], ShouldResemble, "col1")
			So(res.Payload.Rows[1].Values, ShouldHaveLength, 6)
			So(res.Payload.Rows[1].Values[1], ShouldResemble, "col2")

			// test show index from table
			readQuery, err = buildQuery(types.ReadQuery, 1, 7, []string{
				"show index from table test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values[0], ShouldResemble, "test_index")
		})

		Convey("test read write", func() {
			// test write query
			var writeQuery *types.Request
			var res *types.Response
			writeQuery, err = buildQuery(types.WriteQuery, 1, 1, []string{
				"create table test (test int)",
				"insert into test values(1)",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)
			So(res.Header.RowCount, ShouldEqual, 0)

			// test select query
			var readQuery *types.Request
			readQuery, err = buildQuery(types.ReadQuery, 1, 2, []string{
				"select * from test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
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
			var writeQuery *types.Request
			var res *types.Response
			writeQuery, err = buildQuery(types.WriteQuery, 1, 1, []string{
				"create table test (test int)",
				"insert into test values(1)",
			})
			So(err, ShouldBeNil)

			// request once
			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)
			So(res.Header.RowCount, ShouldEqual, 0)

			// request again with same sequence
			writeQuery, err = buildQuery(types.WriteQuery, 1, 1, []string{
				"insert into test values(2)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request again with low sequence
			writeQuery, err = buildQuery(types.WriteQuery, 1, 0, []string{
				"insert into test values(3)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request with invalid timestamp
			writeQuery, err = buildQueryWithTimeShift(types.WriteQuery, 1, 2, time.Second*100, []string{
				"insert into test values(4)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request with invalid timestamp
			writeQuery, err = buildQueryWithTimeShift(types.WriteQuery, 1, 2, -time.Second*100, []string{
				"insert into test values(5)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request with different connection id
			writeQuery, err = buildQuery(types.WriteQuery, 2, 1, []string{
				"insert into test values(6)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)

			// read query, test records
			var readQuery *types.Request
			readQuery, err = buildQuery(types.ReadQuery, 1, 2, []string{
				"select * from test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)

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
			var req *types.Request
			var err error
			req, err = buildQuery(-1, 1, 1, []string{
				"create table test (test int)",
			})
			So(err, ShouldBeNil)
			_, err = db.Query(req)
			So(err, ShouldNotBeNil)

			var writeQuery *types.Request
			var res *types.Response
			writeQuery, err = buildQuery(types.WriteQuery, 1, 1, []string{
				"create table test (test int)",
			})
			So(err, ShouldBeNil)
			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)

			// read query, test records
			var readQuery *types.Request
			readQuery, err = buildQuery(types.ReadQuery, 1, 2, []string{
				"select * from test",
			})
			So(err, ShouldBeNil)
			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(0))
			So(res.Payload.Columns, ShouldResemble, []string{"test"})
			So(res.Payload.DeclTypes, ShouldResemble, []string{"int"})
			So(res.Payload.Rows, ShouldBeEmpty)

			// write query, test failed
			writeQuery, err = buildQuery(types.WriteQuery, 1, 3, []string{
				"insert into test2 values(1)", // table should not exists
			})
			So(err, ShouldBeNil)
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// read query, test dynamic fields
			readQuery, err = buildQuery(types.ReadQuery, 1, 4, []string{
				"select 1 as test",
			})
			So(err, ShouldBeNil)
			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)

			// wait for callback to sign signature
			time.Sleep(time.Millisecond * 10)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Columns, ShouldResemble, []string{"test"})
			So(res.Payload.DeclTypes, ShouldResemble, []string{""})
			So(res.Payload.Rows, ShouldNotBeEmpty)

			// test ack
			var ack *types.Ack
			ack, err = buildAck(res)
			So(err, ShouldBeNil)

			err = db.Ack(ack)
			So(err, ShouldBeNil)

			// test update peers
			peers, err = getPeers(2)
			So(err, ShouldBeNil)
		})

		Reset(func() {
			db.Shutdown()
			os.RemoveAll(rootDir)
			cleanup()
		})
	})
}

func TestInitFailed(t *testing.T) {
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
		kayakMuxService, err := NewDBKayakMuxService("DBKayak", server)
		So(err, ShouldBeNil)

		chainMuxService, err := sqlchain.NewMuxService("sqlchain", server)
		So(err, ShouldBeNil)

		// create peers
		var peers *proto.Peers
		peers, err = getPeers(1)
		So(err, ShouldBeNil)

		// create file
		cfg := &DBConfig{
			DatabaseID:       "00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9",
			DataDir:          rootDir,
			KayakMux:         kayakMuxService,
			ChainMux:         chainMuxService,
			MaxWriteTimeGap:  time.Duration(5 * time.Second),
			UpdateBlockCount: 2,
		}

		// create genesis block
		var block *types.Block
		block, err = types.CreateRandomBlock(rootHash, true)
		So(err, ShouldBeNil)

		// broken peers configuration
		peers.Term = 2

		// create database
		_, err = NewDatabase(cfg, peers, block)
		So(err, ShouldNotBeNil)
	})
}

func TestDatabaseRecycle(t *testing.T) {
	defer leaktest.Check(t)()
	defer kms.ClosePublicKeyStore()

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
		kayakMuxService, err := NewDBKayakMuxService("DBKayak", server)
		So(err, ShouldBeNil)

		chainMuxService, err := sqlchain.NewMuxService("sqlchain", server)
		So(err, ShouldBeNil)

		// create peers
		var peers *proto.Peers
		peers, err = getPeers(1)
		So(err, ShouldBeNil)

		// create file
		cfg := &DBConfig{
			DatabaseID:       "00000bef611d346c0cbe1beaa76e7f0ed705a194fdf9ac3a248ec70e9c198bf9",
			DataDir:          rootDir,
			KayakMux:         kayakMuxService,
			ChainMux:         chainMuxService,
			MaxWriteTimeGap:  time.Duration(5 * time.Second),
			UpdateBlockCount: 2,
		}

		// create genesis block
		var block *types.Block
		block, err = types.CreateRandomBlock(rootHash, true)
		So(err, ShouldBeNil)

		// create database
		var db *Database
		db, err = NewDatabase(cfg, peers, block)
		So(err, ShouldBeNil)

		// do some query
		var writeQuery *types.Request
		var res *types.Response
		writeQuery, err = buildQuery(types.WriteQuery, 1, 1, []string{
			"create table test (test int)",
			"insert into test values(1)",
		})
		So(err, ShouldBeNil)

		res, err = db.Query(writeQuery)
		So(err, ShouldBeNil)
		So(res.Header.RowCount, ShouldEqual, 0)

		// test select query
		var readQuery *types.Request
		readQuery, err = buildQuery(types.ReadQuery, 1, 2, []string{
			"select * from test",
		})
		So(err, ShouldBeNil)

		res, err = db.Query(readQuery)
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

func TestDatabase_EncodePayload(t *testing.T) {
	Convey("encode payload cache", t, func() {
		db := &Database{}
		req := &types.Request{
			Envelope: proto.Envelope{
				Version: "",
				TTL:     0,
				Expire:  0,
				NodeID: &proto.RawNodeID{
					Hash: hash.Hash{},
				},
			},
			Header: types.SignedRequestHeader{
				RequestHeader: types.RequestHeader{
					QueryType:    1,
					NodeID:       "0000000000000000000000000000000000000000000000000000000000000001",
					DatabaseID:   "1",
					ConnectionID: 1,
					SeqNo:        1,
					Timestamp:    time.Now().UTC(),
					BatchCount:   1,
					QueriesHash:  hash.Hash{},
				},
			},
			Payload: types.RequestPayload{
				Queries: []types.Query{
					{
						Pattern: "xxx",
						Args:    nil,
					},
				},
			},
		}
		encoded, err := db.EncodePayload(req)
		So(err, ShouldBeNil)
		req2, err := db.DecodePayload(encoded)
		So(err, ShouldBeNil)
		So(req.Header, ShouldResemble, req2.(*types.Request).Header)
		So(reflect.DeepEqual(req.Header, req2.(*types.Request).Header), ShouldBeTrue)
		So(reflect.DeepEqual(req.Payload, req2.(*types.Request).Payload), ShouldBeTrue)
		encoded2, err := db.EncodePayload(req)
		So(err, ShouldBeNil)
		So(encoded2, ShouldResemble, encoded)
	})
}

func buildAck(res *types.Response) (ack *types.Ack, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get private/public key
	var privateKey *asymmetric.PrivateKey

	if privateKey, _, err = getKeys(); err != nil {
		return
	}

	ack = &types.Ack{
		Header: types.SignedAckHeader{
			AckHeader: types.AckHeader{
				Response:     res.Header.ResponseHeader,
				ResponseHash: res.Header.Hash(),
				NodeID:       nodeID,
				Timestamp:    getLocalTime(),
			},
		},
	}

	err = ack.Sign(privateKey)

	return
}

func buildQuery(queryType types.QueryType, connID uint64, seqNo uint64, queries []string) (query *types.Request, err error) {
	return buildQueryEx(queryType, connID, seqNo, time.Duration(0), proto.DatabaseID(""), queries)
}

func buildQueryWithDatabaseID(queryType types.QueryType, connID uint64, seqNo uint64, databaseID proto.DatabaseID, queries []string) (query *types.Request, err error) {
	return buildQueryEx(queryType, connID, seqNo, time.Duration(0), databaseID, queries)
}

func buildQueryWithTimeShift(queryType types.QueryType, connID uint64, seqNo uint64, timeShift time.Duration, queries []string) (query *types.Request, err error) {
	return buildQueryEx(queryType, connID, seqNo, timeShift, proto.DatabaseID(""), queries)
}

func buildQueryEx(queryType types.QueryType, connID uint64, seqNo uint64, timeShift time.Duration, databaseID proto.DatabaseID, queries []string) (query *types.Request, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get private/public key
	var privateKey *asymmetric.PrivateKey

	if privateKey, _, err = getKeys(); err != nil {
		return
	}

	tm := getLocalTime()
	tm = tm.Add(-timeShift)

	// build queries
	realQueries := make([]types.Query, len(queries))

	for i, v := range queries {
		realQueries[i].Pattern = v
	}

	query = &types.Request{
		Header: types.SignedRequestHeader{
			RequestHeader: types.RequestHeader{
				DatabaseID:   databaseID,
				QueryType:    queryType,
				NodeID:       nodeID,
				ConnectionID: connID,
				SeqNo:        seqNo,
				Timestamp:    tm,
			},
		},
		Payload: types.RequestPayload{
			Queries: realQueries,
		},
	}
	query.SetContext(context.Background())

	err = query.Sign(privateKey)

	return
}

func getPeers(term uint64) (peers *proto.Peers, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get private/public key
	var privateKey *asymmetric.PrivateKey

	if privateKey, _, err = getKeys(); err != nil {
		return
	}

	// generate peers and sign
	peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Term:    term,
			Leader:  nodeID,
			Servers: []proto.NodeID{nodeID},
		},
	}
	err = peers.Sign(privateKey)
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
