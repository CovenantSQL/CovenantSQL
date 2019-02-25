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
	"github.com/pkg/errors"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	dbQuerySuccCounter metrics.Meter
	dbQueryFailCounter metrics.Meter
)

// ObserverFetchBlockReq defines the request for observer to fetch block.
type ObserverFetchBlockReq struct {
	proto.Envelope
	proto.DatabaseID
	Count int32 // sqlchain block serial number since genesis block (0)
}

// ObserverFetchBlockResp defines the response for observer to fetch block.
type ObserverFetchBlockResp struct {
	Count int32 // sqlchain block serial number since genesis block (0)
	Block *types.Block
}

// DBMSRPCService is the rpc endpoint of database management.
type DBMSRPCService struct {
	DBMS *DBMS
}

// NewDBMSRPCService returns new dbms rpc service endpoint.
func NewDBMSRPCService(serviceName string, server *rpc.Server, dbms *DBMS) (service *DBMSRPCService) {
	service = &DBMSRPCService{
		DBMS: dbms,
	}
	server.RegisterService(serviceName, service)

	dbQuerySuccCounter = metrics.NewMeter()
	metrics.Register("db-query-succ", dbQuerySuccCounter)
	dbQueryFailCounter = metrics.NewMeter()
	metrics.Register("db-query-fail", dbQueryFailCounter)

	return
}

// Query rpc, called by client to issue read/write query.
func (rpc *DBMSRPCService) Query(req *types.Request, res *types.Response) (err error) {
	// Just need to verify signature in db.saveAck
	//if err = req.Verify(); err != nil {
	//	dbQueryFailCounter.Mark(1)
	//	return
	//}
	// verify query is sent from the request node
	if req.Envelope.NodeID.String() != string(req.Header.NodeID) {
		// node id mismatch
		err = errors.Wrap(ErrInvalidRequest, "request node id mismatch in query")
		dbQueryFailCounter.Mark(1)
		return
	}

	var r *types.Response
	if r, err = rpc.DBMS.Query(req); err != nil {
		dbQueryFailCounter.Mark(1)
		return
	}

	*res = *r
	dbQuerySuccCounter.Mark(1)

	return
}

// Ack rpc, called by client to confirm read request.
func (rpc *DBMSRPCService) Ack(ack *types.Ack, _ *types.AckResponse) (err error) {
	// Just need to verify signature in db.saveAck
	//if err = ack.Verify(); err != nil {
	//	return
	//}
	// verify if ack node is the original ack node
	if ack.Envelope.NodeID.String() != string(ack.Header.Response.Request.NodeID) {
		err = errors.Wrap(ErrInvalidRequest, "request node id mismatch in ack")
		return
	}

	// verification
	err = rpc.DBMS.Ack(ack)

	return
}

// Deploy rpc, called by BP to create/drop database and update peers.
func (rpc *DBMSRPCService) Deploy(req *types.UpdateService, _ *types.UpdateServiceResponse) (err error) {
	// verify request node is block producer
	if !route.IsPermitted(&req.Envelope, route.DBSDeploy) {
		err = errors.Wrap(ErrInvalidRequest, "node not permitted for deploy request")
		return
	}

	// verify signature
	if err = req.Verify(); err != nil {
		return
	}

	// create/drop/update
	switch req.Header.Op {
	case types.CreateDB:
		err = rpc.DBMS.Create(&req.Header.Instance, true)
	case types.UpdateDB:
		err = rpc.DBMS.Update(&req.Header.Instance)
	case types.DropDB:
		err = rpc.DBMS.Drop(req.Header.Instance.DatabaseID)
	}

	return
}
