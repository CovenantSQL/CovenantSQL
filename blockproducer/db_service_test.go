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

package blockproducer

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/metric"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/route"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

func TestService(t *testing.T) {
	Convey("test db service", t, func() {
		// init node
		var cleanup func()
		var dht *route.DHTService
		var metricService *metric.CollectServer
		var server *rpc.Server
		var err error

		cleanup, dht, metricService, server, err = initNode()
		defer cleanup()

		// create service
		stubPersistence := &stubDBMetaPersistence{}
		svcMap, err := InitServiceMap(stubPersistence)
		So(err, ShouldBeNil)
		dbService := &DBService{
			AllocationRounds: DefaultAllocationRounds,
			ServiceMap:       svcMap,
			Consistent:       dht.Consistent,
			NodeMetrics:      &metricService.NodeMetric,
		}

		// register BPDB service to rpc
		err = server.RegisterService(DBServiceName, dbService)
		So(err, ShouldBeNil)

		// get database
		var nodeID proto.NodeID
		nodeID, err = kms.GetLocalNodeID()
		So(err, ShouldBeNil)

		// test get database
		getReq := &GetDatabaseRequest{
			DatabaseID: proto.DatabaseID("db"),
		}
		getRes := new(GetDatabaseResponse)
		err = rpc.NewCaller().CallNode(nodeID, DBServiceName+".GetDatabase", getReq, getRes)
		So(err, ShouldBeNil)
		So(getRes.InstanceMeta.DatabaseID, ShouldResemble, proto.DatabaseID("db"))

		// get node databases
		getAllReq := new(wt.InitService)
		getAllRes := new(wt.InitServiceResponse)
		err = rpc.NewCaller().CallNode(nodeID, DBServiceName+".GetNodeDatabases", getAllReq, getAllRes)
		So(err, ShouldBeNil)
		So(getAllRes.Instances, ShouldHaveLength, 1)
		So(getAllRes.Instances[0].DatabaseID, ShouldResemble, proto.DatabaseID("db"))

		// create database, no metric received, should failed
		createDBReq := &CreateDatabaseRequest{
			ResourceMeta: wt.ResourceMeta{
				Node: 1,
			},
		}
		createDBRes := new(CreateDatabaseResponse)
		err = rpc.NewCaller().CallNode(nodeID, DBServiceName+".CreateDatabase", createDBReq, createDBRes)
		So(err, ShouldNotBeNil)

		// trigger metrics
		metric.NewCollectClient().UploadMetrics(nodeID, nil)
		createDBRes = new(CreateDatabaseResponse)
		err = rpc.NewCaller().CallNode(nodeID, DBServiceName+".CreateDatabase", createDBReq, createDBRes)
		So(err, ShouldBeNil)
		So(createDBRes.InstanceMeta.DatabaseID, ShouldNotBeEmpty)

		// get all databases, this new database should exists
		err = rpc.NewCaller().CallNode(nodeID, DBServiceName+".GetNodeDatabases", getAllReq, getAllRes)
		So(err, ShouldBeNil)
		So(getAllRes.Instances, ShouldHaveLength, 2)
		So(getAllRes.Instances[0].DatabaseID, ShouldBeIn, []proto.DatabaseID{
			proto.DatabaseID("db"),
			createDBRes.InstanceMeta.DatabaseID,
		})
		So(getAllRes.Instances[1].DatabaseID, ShouldBeIn, []proto.DatabaseID{
			proto.DatabaseID("db"),
			createDBRes.InstanceMeta.DatabaseID,
		})

		// use the database
		serverID := createDBRes.InstanceMeta.Peers.Leader.ID
		dbID := createDBRes.InstanceMeta.DatabaseID
		var queryReq *wt.Request
		queryReq, err = buildQuery(wt.WriteQuery, 1, 1, dbID, []string{
			"create table test (test int)",
			"insert into test values(1)",
		})
		So(err, ShouldBeNil)
		queryRes := new(wt.Response)
		err = rpc.NewCaller().CallNode(serverID, "DBS.Query", queryReq, queryRes)
		So(err, ShouldBeNil)
		queryReq, err = buildQuery(wt.ReadQuery, 1, 2, dbID, []string{
			"select * from test",
		})
		So(err, ShouldBeNil)
		err = rpc.NewCaller().CallNode(serverID, "DBS.Query", queryReq, queryRes)
		So(err, ShouldBeNil)
		err = queryRes.Verify()
		So(err, ShouldBeNil)
		So(queryRes.Header.RowCount, ShouldEqual, uint64(1))
		So(queryRes.Payload.Columns, ShouldResemble, []string{"test"})
		So(queryRes.Payload.DeclTypes, ShouldResemble, []string{"int"})
		So(queryRes.Payload.Rows, ShouldNotBeEmpty)
		So(queryRes.Payload.Rows[0].Values, ShouldNotBeEmpty)
		So(queryRes.Payload.Rows[0].Values[0], ShouldEqual, 1)

		// drop database
		dropDBReq := &DropDatabaseRequest{
			DatabaseID: createDBRes.InstanceMeta.DatabaseID,
		}
		dropDBRes := new(DropDatabaseResponse)
		err = rpc.NewCaller().CallNode(nodeID, DBServiceName+".DropDatabase", dropDBReq, dropDBRes)
		So(err, ShouldBeNil)

		// get this database again to test if it is dropped
		getReq = &GetDatabaseRequest{
			DatabaseID: createDBRes.InstanceMeta.DatabaseID,
		}
		err = rpc.NewCaller().CallNode(nodeID, DBServiceName+".GetDatabase", getReq, getRes)
		So(err, ShouldNotBeNil)
	})
}

func buildQuery(queryType wt.QueryType, connID uint64, seqNo uint64, databaseID proto.DatabaseID, queries []string) (query *wt.Request, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get private/public key
	var pubKey *asymmetric.PublicKey
	var privateKey *asymmetric.PrivateKey
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	tm := time.Now().UTC()

	// build queries
	realQueries := make([]storage.Query, len(queries))

	for i, v := range queries {
		realQueries[i].Pattern = v
	}

	query = &wt.Request{
		Header: wt.SignedRequestHeader{
			RequestHeader: wt.RequestHeader{
				DatabaseID:   databaseID,
				QueryType:    queryType,
				NodeID:       nodeID,
				ConnectionID: connID,
				SeqNo:        seqNo,
				Timestamp:    tm,
			},
			Signee: pubKey,
		},
		Payload: wt.RequestPayload{
			Queries: realQueries,
		},
	}

	err = query.Sign(privateKey)

	return
}
