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
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDBMS(t *testing.T) {
	Convey("test dbms", t, func() {
		var err error
		var server *rpc.Server
		var cleanup func()
		cleanup, server, err = initNode()
		So(err, ShouldBeNil)

		var privateKey *asymmetric.PrivateKey
		privateKey, err = kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)

		var rootDir string
		rootDir, err = ioutil.TempDir("", "dbms_test_")
		So(err, ShouldBeNil)

		cfg := &DBMSConfig{
			RootDir:       rootDir,
			Server:        server,
			MaxReqTimeGap: time.Second * 5,
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
		var peers *proto.Peers
		var block *ct.Block

		dbID := proto.DatabaseID("db")

		// create sqlchain block
		block, err = createRandomBlock(rootHash, true)
		So(err, ShouldBeNil)

		// get peers
		peers, err = getPeers(1)
		So(err, ShouldBeNil)

		// call with no BP privilege
		req = new(wt.UpdateService)
		req.Header.Op = wt.CreateDB
		req.Header.Instance = wt.ServiceInstance{
			DatabaseID:   dbID,
			Peers:        peers,
			GenesisBlock: block,
		}
		err = req.Sign(privateKey)
		So(err, ShouldBeNil)

		Convey("with bp privilege", func() {
			// send update again
			err = testRequest(route.DBSDeploy, req, &res)
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

				err = testRequest(route.DBSQuery, writeQuery, &queryRes)
				So(err, ShouldBeNil)
				err = queryRes.Verify()
				So(err, ShouldBeNil)
				So(queryRes.Header.RowCount, ShouldEqual, 0)
				So(queryRes.Header.LogOffset, ShouldEqual, 0)

				var reqGetRequest wt.GetRequestReq
				var respGetRequest *wt.GetRequestResp

				reqGetRequest.DatabaseID = dbID
				reqGetRequest.LogOffset = queryRes.Header.LogOffset
				err = testRequest(route.DBSGetRequest, reqGetRequest, &respGetRequest)
				So(err, ShouldBeNil)
				So(respGetRequest.Request.Header.Hash, ShouldResemble, writeQuery.Header.Hash)

				// sending read query
				var readQuery *wt.Request
				readQuery, err = buildQueryWithDatabaseID(wt.ReadQuery, 1, 2, dbID, []string{
					"select * from test",
				})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, readQuery, &queryRes)
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
				err = testRequest(route.DBSAck, ack, &ackRes)
				So(err, ShouldBeNil)
			})

			Convey("query non-existent database", func() {
				// sending write query
				var writeQuery *wt.Request
				var queryRes *wt.Response
				writeQuery, err = buildQueryWithDatabaseID(wt.WriteQuery, 1, 1,
					proto.DatabaseID("db_not_exists"), []string{
						"create table test (test int)",
						"insert into test values(1)",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, writeQuery, &queryRes)
				So(err, ShouldNotBeNil)
			})

			Convey("update peers", func() {
				// update database
				peers, err = getPeers(2)
				So(err, ShouldBeNil)

				req = new(wt.UpdateService)
				req.Header.Op = wt.UpdateDB
				req.Header.Instance = wt.ServiceInstance{
					DatabaseID: dbID,
					Peers:      peers,
				}
				err = req.Sign(privateKey)
				So(err, ShouldBeNil)

				err = testRequest(route.DBSDeploy, req, &res)
				So(err, ShouldBeNil)
			})

			Convey("drop database before shutdown", func() {
				// drop database
				req = new(wt.UpdateService)
				req.Header.Op = wt.DropDB
				req.Header.Instance = wt.ServiceInstance{
					DatabaseID: dbID,
				}
				err = req.Sign(privateKey)
				So(err, ShouldBeNil)

				err = testRequest(route.DBSDeploy, req, &res)
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

func testRequest(method route.RemoteFunc, req interface{}, response interface{}) (err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	return rpc.NewCaller().CallNode(nodeID, method.String(), req, response)
}
