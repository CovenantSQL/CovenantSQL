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

package blockproducer

import (
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/metric"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestService(t *testing.T) {
	Convey("test db service", t, func() {
		// init node
		var cleanup func()
		var dht *route.DHTService
		var metricService *metric.CollectServer
		var server *rpc.Server
		var err error

		cleanup, dht, metricService, server, err = initNode(
			"../test/node_standalone/config.yaml",
			"../test/node_standalone/private.key",
		)
		defer cleanup()

		// get keys
		var privateKey *asymmetric.PrivateKey
		privateKey, err = kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)

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
		err = server.RegisterService(route.BPDBRPCName, dbService)
		So(err, ShouldBeNil)

		// get database
		var nodeID proto.NodeID
		nodeID, err = kms.GetLocalNodeID()
		So(err, ShouldBeNil)

		// test get database
		getReq := new(GetDatabaseRequest)
		getReq.Header.DatabaseID = proto.DatabaseID("db")
		err = getReq.Sign(privateKey)
		So(err, ShouldBeNil)

		getRes := new(GetDatabaseResponse)
		err = rpc.NewCaller().CallNode(nodeID, route.BPDBGetDatabase.String(), getReq, getRes)
		So(err, ShouldBeNil)
		So(getReq.Verify(), ShouldBeNil)
		So(getRes.Header.InstanceMeta.DatabaseID, ShouldResemble, proto.DatabaseID("db"))

		// get node databases
		getAllReq := new(wt.InitService)
		getAllRes := new(wt.InitServiceResponse)
		err = rpc.NewCaller().CallNode(nodeID, route.BPDBGetNodeDatabases.String(), getAllReq, getAllRes)
		So(err, ShouldBeNil)
		So(getAllRes.Verify(), ShouldBeNil)
		So(getAllRes.Header.Instances, ShouldHaveLength, 1)
		So(getAllRes.Header.Instances[0].DatabaseID, ShouldResemble, proto.DatabaseID("db"))

		// create database, no metric received, should failed
		createDBReq := new(CreateDatabaseRequest)
		createDBReq.Header.ResourceMeta = wt.ResourceMeta{
			Node: 1,
		}
		err = createDBReq.Sign(privateKey)
		So(err, ShouldBeNil)
		createDBRes := new(CreateDatabaseResponse)
		err = rpc.NewCaller().CallNode(nodeID, route.BPDBCreateDatabase.String(), createDBReq, createDBRes)
		So(err, ShouldNotBeNil)

		// trigger metrics, but does not allow block producer to service as miner
		metric.NewCollectClient().UploadMetrics(nodeID)
		createDBRes = new(CreateDatabaseResponse)
		err = rpc.NewCaller().CallNode(nodeID, route.BPDBCreateDatabase.String(), createDBReq, createDBRes)
		So(err, ShouldNotBeNil)

		// allow block producer to service as miner, only use this in test case
		dbService.includeBPNodesForAllocation = true
		createDBRes = new(CreateDatabaseResponse)
		err = rpc.NewCaller().CallNode(nodeID, route.BPDBCreateDatabase.String(), createDBReq, createDBRes)
		So(err, ShouldBeNil)
		So(createDBRes.Verify(), ShouldBeNil)
		So(createDBRes.Header.InstanceMeta.DatabaseID, ShouldNotBeEmpty)

		// get all databases, this new database should exists
		err = rpc.NewCaller().CallNode(nodeID, route.BPDBGetNodeDatabases.String(), getAllReq, getAllRes)
		So(err, ShouldBeNil)
		So(getAllRes.Verify(), ShouldBeNil)
		So(getAllRes.Header.Instances, ShouldHaveLength, 2)
		So(getAllRes.Header.Instances[0].DatabaseID, ShouldBeIn, []proto.DatabaseID{
			proto.DatabaseID("db"),
			createDBRes.Header.InstanceMeta.DatabaseID,
		})
		So(getAllRes.Header.Instances[1].DatabaseID, ShouldBeIn, []proto.DatabaseID{
			proto.DatabaseID("db"),
			createDBRes.Header.InstanceMeta.DatabaseID,
		})

		// use the database
		serverID := createDBRes.Header.InstanceMeta.Peers.Leader
		dbID := createDBRes.Header.InstanceMeta.DatabaseID
		var queryReq *wt.Request
		queryReq, err = buildQuery(wt.WriteQuery, 1, 1, dbID, []string{
			"create table test (test int)",
			"insert into test values(1)",
		})
		So(err, ShouldBeNil)
		queryRes := new(wt.Response)
		err = rpc.NewCaller().CallNode(serverID, route.DBSQuery.String(), queryReq, queryRes)
		So(err, ShouldBeNil)
		queryReq, err = buildQuery(wt.ReadQuery, 1, 2, dbID, []string{
			"select * from test",
		})
		So(err, ShouldBeNil)
		err = rpc.NewCaller().CallNode(serverID, route.DBSQuery.String(), queryReq, queryRes)
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
		dropDBReq := new(DropDatabaseRequest)
		dropDBReq.Header.DatabaseID = createDBRes.Header.InstanceMeta.DatabaseID
		err = dropDBReq.Sign(privateKey)
		So(err, ShouldBeNil)
		dropDBRes := new(DropDatabaseResponse)
		err = rpc.NewCaller().CallNode(nodeID, route.BPDBDropDatabase.String(), dropDBReq, dropDBRes)
		So(err, ShouldBeNil)

		// get this database again to test if it is dropped
		getReq = new(GetDatabaseRequest)
		getReq.Header.DatabaseID = createDBRes.Header.InstanceMeta.DatabaseID
		err = getReq.Sign(privateKey)
		So(err, ShouldBeNil)
		err = rpc.NewCaller().CallNode(nodeID, route.BPDBGetDatabase.String(), getReq, getRes)
		So(err, ShouldNotBeNil)
	})
}

func buildQuery(queryType wt.QueryType, connID uint64, seqNo uint64, databaseID proto.DatabaseID, queries []string) (query *wt.Request, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get private key
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	tm := time.Now().UTC()

	// build queries
	realQueries := make([]wt.Query, len(queries))

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
		},
		Payload: wt.RequestPayload{
			Queries: realQueries,
		},
	}

	err = query.Sign(privateKey)

	return
}
