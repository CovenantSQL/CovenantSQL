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
	"runtime/trace"

	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics"
)

var (
	dbQuerySuccCounter metrics.Meter
	dbQueryFailCounter metrics.Meter
)

// DBMSRPCService is the rpc endpoint of database management.
type DBMSRPCService struct {
	dbms *DBMS
}

// NewDBMSRPCService returns new dbms rpc service endpoint.
func NewDBMSRPCService(serviceName string, server *rpc.Server, dbms *DBMS) (service *DBMSRPCService) {
	service = &DBMSRPCService{
		dbms: dbms,
	}
	server.RegisterService(serviceName, service)

	dbQuerySuccCounter = metrics.NewMeter()
	metrics.Register("db-query-succ", dbQuerySuccCounter)
	dbQueryFailCounter = metrics.NewMeter()
	metrics.Register("db-query-fail", dbQueryFailCounter)

	return
}

// Query rpc, called by client to issue read/write query.
func (rpc *DBMSRPCService) Query(req *wt.Request, res *wt.Response) (err error) {
	// Just need to verify signature in db.saveAck
	//if err = req.Verify(); err != nil {
	//	dbQueryFailCounter.Mark(1)
	//	return
	//}
	ctx := context.Background()
	ctx, task := trace.NewTask(ctx, "Query")
	defer task.End()
	defer trace.StartRegion(ctx, "QueryRegion").End()
	// verify query is sent from the request node
	if req.Envelope.NodeID.String() != string(req.Header.NodeID) {
		// node id mismatch
		err = errors.Wrap(ErrInvalidRequest, "request node id mismatch in query")
		dbQueryFailCounter.Mark(1)
		return
	}

	var r *wt.Response
	if r, err = rpc.dbms.Query(req); err != nil {
		dbQueryFailCounter.Mark(1)
		return
	}

	*res = *r
	dbQuerySuccCounter.Mark(1)

	return
}

// Ack rpc, called by client to confirm read request.
func (rpc *DBMSRPCService) Ack(ack *wt.Ack, _ *wt.AckResponse) (err error) {
	// Just need to verify signature in db.saveAck
	//if err = ack.Verify(); err != nil {
	//	return
	//}
	ctx := context.Background()
	ctx, task := trace.NewTask(ctx, "Ack")
	defer task.End()
	defer trace.StartRegion(ctx, "AckRegion").End()

	// verify if ack node is the original ack node
	if ack.Envelope.NodeID.String() != string(ack.Header.Response.Request.NodeID) {
		err = errors.Wrap(ErrInvalidRequest, "request node id mismatch in ack")
		return
	}

	// verification
	err = rpc.dbms.Ack(ack)

	return
}

// Deploy rpc, called by BP to create/drop database and update peers.
func (rpc *DBMSRPCService) Deploy(req *wt.UpdateService, _ *wt.UpdateServiceResponse) (err error) {
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
	case wt.CreateDB:
		err = rpc.dbms.Create(&req.Header.Instance, true)
	case wt.UpdateDB:
		err = rpc.dbms.Update(&req.Header.Instance)
	case wt.DropDB:
		err = rpc.dbms.Drop(req.Header.Instance.DatabaseID)
	}

	return
}

// GetRequest rpc, called by observer to fetch original request by log offset.
func (rpc *DBMSRPCService) GetRequest(req *wt.GetRequestReq, resp *wt.GetRequestResp) (err error) {
	// TODO(xq262144), check permission
	resp.Request, err = rpc.dbms.GetRequest(req.DatabaseID, req.LogOffset)
	return
}
