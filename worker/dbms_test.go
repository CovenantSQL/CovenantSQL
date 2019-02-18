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
	"sync/atomic"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDBMS(t *testing.T) {
	Convey("test dbms", t, func() {
		var err error
		var server *rpc.Server
		var cleanup func()
		cleanup, server, err = initNode()
		So(err, ShouldBeNil)

		var (
			privateKey *asymmetric.PrivateKey
			publicKey  *asymmetric.PublicKey
		)
		privateKey, err = kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)
		publicKey = privateKey.PubKey()

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
		dbms.busService.Stop()

		// add database
		var req *types.UpdateService
		var res types.UpdateServiceResponse
		var peers *proto.Peers
		var block *types.Block
		var nodeID proto.NodeID

		nodeID, err = kms.GetLocalNodeID()

		dbAddr := proto.AccountAddress(hash.HashH([]byte{'d', 'b'}))
		dbID := dbAddr.DatabaseID()
		dbAddr2 := proto.AccountAddress(hash.HashH([]byte{'a', 'b'}))
		dbID2 := dbAddr2.DatabaseID()
		userAddr, err := crypto.PubKeyHash(publicKey)
		So(err, ShouldBeNil)

		// create sqlchain block
		block, err = createRandomBlock(rootHash, true)
		So(err, ShouldBeNil)

		// get peers
		peers, err = getPeers(1)
		So(err, ShouldBeNil)

		// call with no BP privilege
		req = new(types.UpdateService)
		req.Header.Op = types.CreateDB
		req.Header.Instance = types.ServiceInstance{
			DatabaseID:   dbID,
			Peers:        peers,
			GenesisBlock: block,
		}
		err = req.Sign(privateKey)
		So(err, ShouldBeNil)

		var seqNo uint64

		Convey("with bp privilege", func() {
			// send update again
			err = testRequest(route.DBSDeploy, req, &res)
			So(err, ShouldBeNil)

			Convey("query should fail", func() {
				// sending write query
				var writeQuery *types.Request
				var queryRes *types.Response
				writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"create table test (test int)",
						"insert into test values(1)",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, writeQuery, &queryRes)
				So(err.Error(), ShouldContainSubstring, ErrPermissionDeny.Error())

				// sending read query
				var readQuery *types.Request
				readQuery, err = buildQueryWithDatabaseID(types.ReadQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"select * from test",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, readQuery, &queryRes)
				So(err.Error(), ShouldContainSubstring, ErrPermissionDeny.Error())
			})

			// grant write and read permission
			err = dbms.UpdatePermission(dbAddr.DatabaseID(), userAddr,
				&types.PermStat{Permission: types.UserPermissionFromRole(types.ReadWrite), Status: types.Normal})
			So(err, ShouldBeNil)
			userState, ok := dbms.busService.RequestPermStat(dbAddr.DatabaseID(), userAddr)
			So(ok, ShouldBeTrue)
			So(userState.Permission, ShouldNotBeNil)
			So(userState.Permission.Role, ShouldEqual, types.ReadWrite)
			So(userState.Status, ShouldEqual, types.Normal)

			Convey("success write and read", func() {
				// sending write query
				var writeQuery *types.Request
				var queryRes *types.Response
				writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"create table test (test int)",
						"insert into test values(1)",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, writeQuery, &queryRes)
				So(err, ShouldBeNil)
				So(queryRes.Header.RowCount, ShouldEqual, 0)

				// sending read query
				var readQuery *types.Request
				readQuery, err = buildQueryWithDatabaseID(types.ReadQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"select * from test",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, readQuery, &queryRes)
				So(err, ShouldBeNil)
				So(queryRes.Header.RowCount, ShouldEqual, uint64(1))
				So(queryRes.Payload.Columns, ShouldResemble, []string{"test"})
				So(queryRes.Payload.DeclTypes, ShouldResemble, []string{"int"})
				So(queryRes.Payload.Rows, ShouldNotBeEmpty)
				So(queryRes.Payload.Rows[0].Values, ShouldNotBeEmpty)
				So(queryRes.Payload.Rows[0].Values[0], ShouldEqual, 1)

				// sending read ack
				var ack *types.Ack
				ack, err = buildAck(queryRes)
				So(err, ShouldBeNil)

				var ackRes types.AckResponse
				err = testRequest(route.DBSAck, ack, &ackRes)
				So(err, ShouldBeNil)

				_, _, err = dbms.observerFetchBlock(dbID2, nodeID, 1)
				So(err.Error(), ShouldContainSubstring, ErrPermissionDeny.Error())
				_, _, err = dbms.observerFetchBlock(dbID, nodeID, 1)
				So(err, ShouldBeNil)

				// revoke write permission
				err = dbms.UpdatePermission(dbAddr.DatabaseID(), userAddr,
					&types.PermStat{Permission: types.UserPermissionFromRole(types.Read), Status: types.Normal})
				userState, ok := dbms.busService.RequestPermStat(dbAddr.DatabaseID(), userAddr)
				So(ok, ShouldBeTrue)
				So(userState.Permission, ShouldNotBeNil)
				So(userState.Permission.Role, ShouldEqual, types.Read)
				So(userState.Status, ShouldEqual, types.Normal)

				Convey("success reading and fail to write", func() {
					// sending write query
					var writeQuery *types.Request
					var queryRes *types.Response
					writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
						1, atomic.AddUint64(&seqNo, 1),
						dbID, []string{
							"insert into test values(1)",
						})
					So(err, ShouldBeNil)

					err = testRequest(route.DBSQuery, writeQuery, &queryRes)
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, ErrPermissionDeny.Error())

					// sending read query
					var readQuery *types.Request
					readQuery, err = buildQueryWithDatabaseID(types.ReadQuery,
						1, atomic.AddUint64(&seqNo, 1),
						dbID, []string{
							"select * from test",
						})
					So(err, ShouldBeNil)

					err = testRequest(route.DBSQuery, readQuery, &queryRes)
					So(err, ShouldBeNil)

					_, _, err = dbms.observerFetchBlock(dbID, nodeID, 1)
					So(err, ShouldBeNil)
				})

				// grant write only permission
				err = dbms.UpdatePermission(dbAddr.DatabaseID(), userAddr,
					&types.PermStat{Permission: types.UserPermissionFromRole(types.Write), Status: types.Normal})
				userState, ok = dbms.busService.RequestPermStat(dbAddr.DatabaseID(), userAddr)
				So(ok, ShouldBeTrue)
				So(userState.Permission, ShouldNotBeNil)
				So(userState.Permission.Role, ShouldEqual, types.Write)
				So(userState.Status, ShouldEqual, types.Normal)

				Convey("success writing and failed to read", func() {
					// sending read query
					var readQuery *types.Request
					var queryRes *types.Response
					readQuery, err = buildQueryWithDatabaseID(types.ReadQuery,
						1, atomic.AddUint64(&seqNo, 1),
						dbID, []string{
							"select * from test",
						})
					So(err, ShouldBeNil)

					err = testRequest(route.DBSQuery, readQuery, &queryRes)
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, ErrPermissionDeny.Error())

					// sending write query
					var writeQuery *types.Request
					writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
						1, atomic.AddUint64(&seqNo, 1),
						dbID, []string{
							"insert into test values(1)",
						})
					So(err, ShouldBeNil)

					err = testRequest(route.DBSQuery, writeQuery, &queryRes)
					So(err, ShouldBeNil)
					So(queryRes.Header.RowCount, ShouldEqual, 0)
				})
			})

			// grant invalid permission
			err = dbms.UpdatePermission(dbAddr.DatabaseID(), userAddr,
				&types.PermStat{Permission: types.UserPermissionFromRole(types.Void), Status: types.Normal})
			So(err, ShouldBeNil)
			userState, ok = dbms.busService.RequestPermStat(dbAddr.DatabaseID(), userAddr)
			So(ok, ShouldBeTrue)
			So(userState.Permission, ShouldNotBeNil)
			So(userState.Permission.Role, ShouldEqual, types.Void)
			So(userState.Status, ShouldEqual, types.Normal)

			Convey("invalid permission query should fail", func() {
				// sending write query
				var writeQuery *types.Request
				var queryRes *types.Response
				writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"create table test (test int)",
						"insert into test values(1)",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, writeQuery, &queryRes)
				So(err.Error(), ShouldContainSubstring, ErrPermissionDeny.Error())

				// sending read query
				var readQuery *types.Request
				readQuery, err = buildQueryWithDatabaseID(types.ReadQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"select * from test",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, readQuery, &queryRes)
				So(err.Error(), ShouldContainSubstring, ErrPermissionDeny.Error())

				_, _, err = dbms.observerFetchBlock(dbID, nodeID, 1)
				So(err.Error(), ShouldContainSubstring, ErrPermissionDeny.Error())
			})

			// grant admin permission but in arrears
			err = dbms.UpdatePermission(dbAddr.DatabaseID(), userAddr,
				&types.PermStat{Permission: types.UserPermissionFromRole(types.Admin), Status: types.Arrears})
			So(err, ShouldBeNil)
			userState, ok = dbms.busService.RequestPermStat(dbAddr.DatabaseID(), userAddr)
			So(ok, ShouldBeTrue)
			So(userState.Permission, ShouldNotBeNil)
			So(userState.Permission.Role, ShouldEqual, types.Admin)
			So(userState.Status, ShouldEqual, types.Arrears)

			Convey("arrears query should fail", func() {
				// sending write query
				var writeQuery *types.Request
				var queryRes *types.Response
				writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"create table test (test int)",
						"insert into test values(1)",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, writeQuery, &queryRes)
				So(err.Error(), ShouldContainSubstring, ErrPermissionDeny.Error())

				// sending read query
				var readQuery *types.Request
				readQuery, err = buildQueryWithDatabaseID(types.ReadQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"select * from test",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, readQuery, &queryRes)
				So(err.Error(), ShouldContainSubstring, ErrPermissionDeny.Error())
			})

			// switch user to normal
			err = dbms.UpdatePermission(dbAddr.DatabaseID(), userAddr,
				&types.PermStat{Permission: types.UserPermissionFromRole(types.Admin), Status: types.Normal})
			So(err, ShouldBeNil)
			userState, ok = dbms.busService.RequestPermStat(dbAddr.DatabaseID(), userAddr)
			So(ok, ShouldBeTrue)
			So(userState.Permission, ShouldNotBeNil)
			So(userState.Permission.Role, ShouldEqual, types.Admin)
			So(userState.Status, ShouldEqual, types.Normal)

			Convey("can send read and write queries", func() {
				// sending write query
				var writeQuery *types.Request
				var queryRes *types.Response
				writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"create table test (test int)",
						"insert into test values(1)",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, writeQuery, &queryRes)
				So(err, ShouldBeNil)
				So(queryRes.Header.RowCount, ShouldEqual, 0)

				// sending read query
				var readQuery *types.Request
				readQuery, err = buildQueryWithDatabaseID(types.ReadQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"select * from test",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, readQuery, &queryRes)
				So(err, ShouldBeNil)
				So(queryRes.Header.RowCount, ShouldEqual, uint64(1))
				So(queryRes.Payload.Columns, ShouldResemble, []string{"test"})
				So(queryRes.Payload.DeclTypes, ShouldResemble, []string{"int"})
				So(queryRes.Payload.Rows, ShouldNotBeEmpty)
				So(queryRes.Payload.Rows[0].Values, ShouldNotBeEmpty)
				So(queryRes.Payload.Rows[0].Values[0], ShouldEqual, 1)

				// sending read ack
				var ack *types.Ack
				ack, err = buildAck(queryRes)
				So(err, ShouldBeNil)

				var ackRes types.AckResponse
				err = testRequest(route.DBSAck, ack, &ackRes)
				So(err, ShouldBeNil)
			})

			// enforce query pattern regulations
			err = dbms.UpdatePermission(dbAddr.DatabaseID(), userAddr,
				&types.PermStat{Permission: &types.UserPermission{
					Role: types.Admin,
					Patterns: []string{
						"create table test (test int)",
						"SELECT 1",
						"INSERT INTO TEST VALUES(1)",
					},
				}, Status: types.Normal})
			So(err, ShouldBeNil)
			userState, ok = dbms.busService.RequestPermStat(dbAddr.DatabaseID(), userAddr)
			So(ok, ShouldBeTrue)
			So(userState.Permission, ShouldNotBeNil)
			So(userState.Permission.Role, ShouldEqual, types.Admin)
			So(userState.Permission.Patterns, ShouldHaveLength, 3)

			Convey("query patterns restrictions", func() {
				var writeQuery *types.Request
				var queryRes *types.Response

				// sending allowed write query
				writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"create table test (test int)",
						"INSERT INTO TEST VALUES(1)",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, writeQuery, &queryRes)
				So(err, ShouldBeNil)
				So(queryRes.Header.RowCount, ShouldEqual, 0)

				// sending allowed read query
				var readQuery *types.Request
				readQuery, err = buildQueryWithDatabaseID(types.ReadQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"SELECT 1",
					})
				So(err, ShouldBeNil)

				err = testRequest(route.DBSQuery, readQuery, &queryRes)
				So(err, ShouldBeNil)
				So(queryRes.Header.RowCount, ShouldEqual, uint64(1))
				So(queryRes.Payload.Rows, ShouldHaveLength, 1)
				So(queryRes.Payload.Rows[0].Values, ShouldHaveLength, 1)
				So(queryRes.Payload.Rows[0].Values[0], ShouldEqual, 1)

				// sending disallowed write query
				writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"insert into test values(1)",
					})
				So(err, ShouldBeNil)
				err = testRequest(route.DBSQuery, writeQuery, &queryRes)
				So(err, ShouldNotBeNil)

				// sending disallowed write query mixed with valid write query
				writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"INSERT INTO TEST VALUES(1)",
						"insert into test values(1)",
					})
				So(err, ShouldBeNil)
				err = testRequest(route.DBSQuery, writeQuery, &queryRes)
				So(err, ShouldNotBeNil)

				// sending disallowed read query
				readQuery, err = buildQueryWithDatabaseID(types.ReadQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"select * from test",
					})
				So(err, ShouldBeNil)
				err = testRequest(route.DBSQuery, readQuery, &queryRes)
				So(err, ShouldNotBeNil)

				// sending disallowed read query
				readQuery, err = buildQueryWithDatabaseID(types.ReadQuery,
					1, atomic.AddUint64(&seqNo, 1),
					dbID, []string{
						"SELECT 1",
						"select * from test",
					})
				So(err, ShouldBeNil)
				err = testRequest(route.DBSQuery, readQuery, &queryRes)
				So(err, ShouldNotBeNil)
			})

			// set back permission object
			err = dbms.UpdatePermission(dbAddr.DatabaseID(), userAddr,
				&types.PermStat{Permission: types.UserPermissionFromRole(types.Admin), Status: types.Normal})
			So(err, ShouldBeNil)
			userState, ok = dbms.busService.RequestPermStat(dbAddr.DatabaseID(), userAddr)
			So(ok, ShouldBeTrue)
			So(userState.Permission, ShouldNotBeNil)
			So(userState.Permission.Role, ShouldEqual, types.Admin)
			So(userState.Status, ShouldEqual, types.Normal)

			Convey("query non-existent database", func() {
				// sending write query
				var writeQuery *types.Request
				var queryRes *types.Response
				writeQuery, err = buildQueryWithDatabaseID(types.WriteQuery,
					1, atomic.AddUint64(&seqNo, 1),
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

				req = new(types.UpdateService)
				req.Header.Op = types.UpdateDB
				req.Header.Instance = types.ServiceInstance{
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
				req = new(types.UpdateService)
				req.Header.Op = types.DropDB
				req.Header.Instance = types.ServiceInstance{
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
