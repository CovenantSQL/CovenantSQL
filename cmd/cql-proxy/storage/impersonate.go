/*
 * Copyright 2019 The CovenantSQL Authors.
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

package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	gorp "gopkg.in/gorp.v2"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
)

var (
	connIDLock  sync.Mutex
	randSource  = rand.New(rand.NewSource(time.Now().UnixNano()))
	connIDAvail []uint64
	globalSeqNo uint64
)

type impersonatedRows struct {
	columns []string
	types   []string
	data    []types.ResponseRow
}

func (r *impersonatedRows) Columns() []string {
	return r.columns[:]
}

func (r *impersonatedRows) Close() error {
	r.data = nil
	return nil
}

func (r *impersonatedRows) Next(dest []driver.Value) error {
	if len(r.data) == 0 {
		return io.EOF
	}

	for i, d := range r.data[0].Values {
		dest[i] = d
	}

	// unshift data
	r.data = r.data[1:]

	return nil
}

func (r *impersonatedRows) ColumnTypeDatabaseTypeName(index int) string {
	return strings.ToUpper(r.types[index])
}

type impersonatedResult struct {
	affectedRows int64
	lastInsertID int64
}

func (r *impersonatedResult) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

func (r *impersonatedResult) RowsAffected() (int64, error) {
	return r.affectedRows, nil
}

type impersonatedDB struct {
	rpc    rpc.PCaller
	nodeID proto.NodeID
	db     proto.DatabaseID
	key    *asymmetric.PrivateKey
}

func (d *impersonatedDB) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (result driver.Result, err error) {
	resp, err := d.sendQuery(query, args, types.WriteQuery)
	if err != nil {
		return
	}

	result = &impersonatedResult{
		affectedRows: resp.Header.AffectedRows,
		lastInsertID: resp.Header.LastInsertID,
	}

	return
}

func (d *impersonatedDB) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	resp, err := d.sendQuery(query, args, types.ReadQuery)
	if err != nil {
		return
	}

	rows = &impersonatedRows{
		columns: resp.Payload.Columns,
		types:   resp.Payload.DeclTypes,
		data:    resp.Payload.Rows,
	}

	return
}

func (d *impersonatedDB) sendQuery(query string, args []driver.NamedValue, queryType types.QueryType) (
	resp *types.Response, err error) {
	var (
		connID, seqNo = allocateConnAndSeq()
		dArgs         []types.NamedArg
	)

	defer putBackConn(connID)

	for _, arg := range args {
		dArgs = append(dArgs, types.NamedArg{
			Name:  arg.Name,
			Value: arg.Value,
		})
	}

	req := &types.Request{
		Header: types.SignedRequestHeader{
			RequestHeader: types.RequestHeader{
				QueryType:    queryType,
				NodeID:       d.nodeID,
				DatabaseID:   d.db,
				ConnectionID: connID,
				SeqNo:        seqNo,
				Timestamp:    time.Now().UTC(),
			},
		},
		Payload: types.RequestPayload{
			Queries: []types.Query{
				{
					Pattern: query,
					Args:    dArgs,
				},
			},
		},
	}
	resp = &types.Response{}

	if err = req.Sign(d.key); err != nil {
		return
	}

	err = d.rpc.Call(route.DBSQuery.String(), req, resp)
	if err != nil {
		return
	}

	// add ack
	go d.sendAck(&types.Ack{
		Header: types.SignedAckHeader{
			AckHeader: types.AckHeader{
				Response:     resp.Header.ResponseHeader,
				ResponseHash: resp.Header.Hash(),
				NodeID:       d.nodeID,
				Timestamp:    time.Now().UTC(),
			},
		},
	})

	return
}

func (d *impersonatedDB) sendAck(ack *types.Ack) {
	if err := ack.Sign(d.key); err != nil {
		return
	}

	_ = d.rpc.Call(route.DBSAck.String(), ack, nil)
}

func (d *impersonatedDB) Open(name string) (driver.Conn, error) {
	return d, nil
}

func (d *impersonatedDB) Prepare(query string) (driver.Stmt, error) {
	return &impersonatedStmt{
		db:    d,
		query: query,
	}, nil
}

func (d *impersonatedDB) Close() error {
	return nil
}

func (d *impersonatedDB) Begin() (driver.Tx, error) {
	return nil, errors.New("transaction not supported")
}

func (d *impersonatedDB) Connect(context.Context) (driver.Conn, error) {
	return d, nil
}

func (d *impersonatedDB) Driver() driver.Driver {
	return d
}

type impersonatedStmt struct {
	db    *impersonatedDB
	query string
}

func (s *impersonatedStmt) Close() error {
	return nil
}

func (s *impersonatedStmt) NumInput() int {
	return -1
}

func (s *impersonatedStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.db.ExecContext(context.Background(), s.query, convertOldArgs(args))
}

func (s *impersonatedStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.db.QueryContext(context.Background(), s.query, convertOldArgs(args))
}

func (s *impersonatedStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.db.ExecContext(ctx, s.query, args)
}

func (s *impersonatedStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.db.QueryContext(ctx, s.query, args)
}

func allocateConnAndSeq() (connID uint64, seqNo uint64) {
	connIDLock.Lock()
	defer connIDLock.Unlock()

	if len(connIDAvail) == 0 {
		// generate one
		connID = randSource.Uint64()
		seqNo = atomic.AddUint64(&globalSeqNo, 1)
		return
	}

	// pop one conn
	connID = connIDAvail[0]
	connIDAvail = connIDAvail[1:]
	seqNo = atomic.AddUint64(&globalSeqNo, 1)

	return
}

func putBackConn(connID uint64) {
	connIDLock.Lock()
	defer connIDLock.Unlock()

	connIDAvail = append(connIDAvail, connID)
}

func convertOldArgs(args []driver.Value) (dargs []driver.NamedValue) {
	dargs = make([]driver.NamedValue, len(args))

	for i, v := range args {
		dargs[i] = driver.NamedValue{
			Ordinal: i + 1,
			Value:   v,
		}
	}

	return
}

func NewImpersonatedDB(nodeID proto.NodeID, rpc rpc.PCaller, dbID proto.DatabaseID,
	key *asymmetric.PrivateKey) *gorp.DbMap {
	return &gorp.DbMap{
		Db: sql.OpenDB(&impersonatedDB{
			nodeID: nodeID,
			db:     dbID,
			key:    key,
			rpc:    rpc,
		}),
		Dialect: gorp.SqliteDialect{},
	}
}
