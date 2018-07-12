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
	"bytes"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	ct "gitlab.com/thunderdb/ThunderDB/sqlchain/types"
	"gitlab.com/thunderdb/ThunderDB/utils"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

func TestDBMS(t *testing.T) {
	Convey("test dbms", t, func() {
		var err error
		var server *rpc.Server
		var cleanup func()
		cleanup, server, err = initNode()
		So(err, ShouldBeNil)

		var rootDir string
		rootDir, err = ioutil.TempDir("", "dbms_test_")
		So(err, ShouldBeNil)

		cfg := &DBMSConfig{
			RootDir:         rootDir,
			Server:          server,
			MaxWriteTimeGap: time.Second * 5,
		}

		var dbms *DBMS
		dbms, err = NewDBMS(cfg)
		So(err, ShouldBeNil)

		// init
		err = dbms.Init()
		So(err, ShouldBeNil)

		// add database
		var req *wt.UpdateService
		var res wt.UpdateServiceResponse
		var peers *kayak.Peers
		var block *ct.Block

		dbID := proto.DatabaseID("db")

		// create sqlchain block
		block, err = createRandomBlock(rootHash, true)
		So(err, ShouldBeNil)

		var blockBuffer *bytes.Buffer
		blockBuffer, err = utils.EncodeMsgPack(block)
		So(err, ShouldBeNil)

		// get peers
		peers, err = getPeers(1)
		So(err, ShouldBeNil)

		// call with no BP privilege
		req, err = buildUpdateRequest(wt.CreateDB, &wt.ServiceInstance{
			DatabaseID:   dbID,
			Peers:        peers,
			GenesisBlock: blockBuffer.Bytes(),
		})
		So(err, ShouldBeNil)

		Convey("with bp privilege", func() {
			// send update again
			err = testRequest("Update", req, &res)
			So(err, ShouldBeNil)

			Convey("queries", func() {
				// sending write query
				var writeQuery *wt.Request
				var queryRes *wt.Response
				writeQuery, err = buildQueryWithDatabaseID(wt.WriteQuery, 1, 1, dbID, []string{
					"create table test (test int)",
					"insert into test values(1)",
				})
				So(err, ShouldBeNil)

				err = testRequest("Query", writeQuery, &queryRes)
				So(err, ShouldBeNil)
				err = queryRes.Verify()
				So(err, ShouldBeNil)
				So(queryRes.Header.RowCount, ShouldEqual, 0)

				// sending read query
				var readQuery *wt.Request
				readQuery, err = buildQueryWithDatabaseID(wt.ReadQuery, 1, 2, dbID, []string{
					"select * from test",
				})
				So(err, ShouldBeNil)

				err = testRequest("Query", readQuery, &queryRes)
				So(err, ShouldBeNil)
				err = queryRes.Verify()
				So(err, ShouldBeNil)
				So(queryRes.Header.RowCount, ShouldEqual, uint64(1))
				So(queryRes.Payload.Columns, ShouldResemble, []string{"test"})
				So(queryRes.Payload.DeclTypes, ShouldResemble, []string{"int"})
				So(queryRes.Payload.Rows, ShouldNotBeEmpty)
				So(queryRes.Payload.Rows[0].Values, ShouldNotBeEmpty)
				So(queryRes.Payload.Rows[0].Values[0], ShouldEqual, 1)

				// sending read ack
				var ack *wt.Ack
				ack, err = buildAck(queryRes)
				So(err, ShouldBeNil)

				var ackRes wt.AckResponse
				err = testRequest("Ack", ack, &ackRes)
				So(err, ShouldBeNil)
			})

			Convey("query non-existent database", func() {
				// sending write query
				var writeQuery *wt.Request
				var queryRes *wt.Response
				writeQuery, err = buildQueryWithDatabaseID(wt.WriteQuery, 1, 1,
					proto.DatabaseID("db2"), []string{
						"create table test (test int)",
						"insert into test values(1)",
					})
				So(err, ShouldBeNil)

				err = testRequest("Query", writeQuery, &queryRes)
				So(err, ShouldNotBeNil)
			})

			Convey("update peers", func() {
				// update database
				peers, err = getPeers(2)
				So(err, ShouldBeNil)

				req, err = buildUpdateRequest(wt.UpdateDB, &wt.ServiceInstance{
					DatabaseID: dbID,
					Peers:      peers,
				})
				So(err, ShouldBeNil)
				err = testRequest("Update", req, &res)
				So(err, ShouldBeNil)
			})

			Convey("drop database before shutdown", func() {
				// drop database
				req, err = buildUpdateRequest(wt.DropDB, &wt.ServiceInstance{
					DatabaseID: dbID,
				})
				So(err, ShouldBeNil)
				err = testRequest("Update", req, &res)
				So(err, ShouldBeNil)

				// shutdown
				err = dbms.Shutdown()
				So(err, ShouldBeNil)
			})
		})

		Reset(func() {
			// shutdown
			err = dbms.Shutdown()
			So(err, ShouldBeNil)

			// cleanup
			os.RemoveAll(rootDir)
			cleanup()
		})
	})
}

func buildUpdateRequest(opType wt.UpdateType, instance *wt.ServiceInstance) (req *wt.UpdateService, err error) {
	// get private/public key
	var pubKey *asymmetric.PublicKey
	var privateKey *asymmetric.PrivateKey

	if privateKey, pubKey, err = getKeys(); err != nil {
		return
	}

	req = &wt.UpdateService{
		Header: wt.SignedUpdateServiceHeader{
			UpdateServiceHeader: wt.UpdateServiceHeader{
				Op:       opType,
				Instance: *instance,
			},
			Signee: pubKey,
		},
	}

	err = req.Sign(privateKey)

	return
}

func testRequest(method string, req interface{}, response interface{}) (err error) {
	realMethod := DBServiceRPCName + "." + method

	// get node id
	var nodeID proto.NodeID
	if nodeID, err = getNodeID(); err != nil {
		return
	}

	var conn net.Conn
	if conn, err = rpc.DialToNode(nodeID, nil); err != nil {
		return
	}

	var client *rpc.Client
	if client, err = rpc.InitClientConn(conn); err != nil {
		conn.Close()
		return
	}

	defer client.Close()

	return client.Call(realMethod, req, response)
}
