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

package client

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/CovenantSQL/utils/trace"
	"github.com/pkg/errors"
)

// conn implements an interface sql.Conn.
type conn struct {
	dbID proto.DatabaseID

	queries     []types.Query
	localNodeID proto.NodeID
	privKey     *asymmetric.PrivateKey

	inTransaction bool
	closed        int32

	leader   *pconn
	follower *pconn
}

// pconn represents a connection to a peer
type pconn struct {
	parent  *conn
	ackCh   chan *types.Ack
	pCaller *rpc.PersistentCaller
}

func newConn(cfg *Config) (c *conn, err error) {
	// get local node id
	var localNodeID proto.NodeID
	if localNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get local private key
	var privKey *asymmetric.PrivateKey
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	c = &conn{
		dbID:        proto.DatabaseID(cfg.DatabaseID),
		localNodeID: localNodeID,
		privKey:     privKey,
		queries:     make([]types.Query, 0),
	}

	// get peers from BP
	var peers *proto.Peers
	if peers, err = cacheGetPeers(c.dbID, c.privKey); err != nil {
		return nil, errors.WithMessage(err, "cacheGetPeers failed")
	}

	if cfg.UseLeader {
		c.leader = &pconn{
			parent:  c,
			pCaller: rpc.NewPersistentCaller(peers.Leader),
		}
	}

	// choose a random follower node
	if cfg.UseFollower && len(peers.Servers) > 1 {
		for {
			node := peers.Servers[randSource.Intn(len(peers.Servers))]
			if node != peers.Leader {
				c.follower = &pconn{
					parent:  c,
					pCaller: rpc.NewPersistentCaller(node),
				}
				break
			}
		}
	}

	if c.leader == nil && c.follower == nil {
		return nil, errors.New("no follower peers found")
	}

	if c.leader != nil {
		if err := c.leader.startAckWorkers(2); err != nil {
			return nil, errors.WithMessage(err, "leader startAckWorkers failed")
		}
	}
	if c.follower != nil {
		if err := c.follower.startAckWorkers(2); err != nil {
			return nil, errors.WithMessage(err, "follower startAckWorkers failed")
		}
	}

	log.WithField("db", c.dbID).Debug("new connection to database")
	return
}

func (c *pconn) startAckWorkers(workerCount int) (err error) {
	c.ackCh = make(chan *types.Ack, workerCount*4)
	for i := 0; i < workerCount; i++ {
		go c.ackWorker()
	}
	return
}

func (c *pconn) stopAckWorkers() {
	close(c.ackCh)
}

func (c *pconn) ackWorker() {
	var (
		oneTime sync.Once
		pc      *rpc.PersistentCaller
		err     error
	)

ackWorkerLoop:
	for {
		ack, got := <-c.ackCh
		if !got { // closed and empty
			break ackWorkerLoop
		}
		oneTime.Do(func() {
			pc = rpc.NewPersistentCaller(c.pCaller.TargetID)
		})
		if err = ack.Sign(c.parent.privKey); err != nil {
			log.WithField("target", pc.TargetID).WithError(err).Error("failed to sign ack")
			continue
		}

		var ackRes types.AckResponse
		// send ack back
		if err = pc.Call(route.DBSAck.String(), ack, &ackRes); err != nil {
			log.WithError(err).Debug("send ack failed")
			continue
		}
	}

	if pc != nil {
		pc.Close()
	}

	log.Debug("ack worker quiting")
}

func (c *pconn) close() error {
	c.stopAckWorkers()
	if c.pCaller != nil {
		c.pCaller.Close()
	}
	return nil
}

// Prepare implements the driver.Conn.Prepare method.
func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

// Close implements the driver.Conn.Close method.
func (c *conn) Close() error {
	// close the meta connection
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		log.WithField("db", c.dbID).Debug("closed connection")
	}
	if c.leader != nil {
		c.leader.close()
	}
	if c.follower != nil {
		c.follower.close()
	}
	return nil
}

// Begin implements the driver.Conn.Begin method.
func (c *conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

// BeginTx implements the driver.ConnBeginTx.BeginTx method.
func (c *conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if atomic.LoadInt32(&c.closed) != 0 {
		return nil, driver.ErrBadConn
	}

	// start transaction
	log.WithField("inTx", c.inTransaction).Debug("begin transaction")

	if c.inTransaction {
		return nil, sql.ErrTxDone
	}

	// TODO(xq262144): make use of the ctx argument
	c.inTransaction = true
	c.queries = c.queries[:0]

	return c, nil
}

// PrepareContext implements the driver.ConnPrepareContext.ConnPrepareContext method.
func (c *conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if atomic.LoadInt32(&c.closed) != 0 {
		return nil, driver.ErrBadConn
	}

	log.WithField("query", query).Debug("prepared statement")

	// prepare the statement
	return newStmt(c, query), nil
}

// ExecContext implements the driver.ExecerContext.ExecContext method.
func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (result driver.Result, err error) {
	defer trace.StartRegion(ctx, "dbExec").End()

	if atomic.LoadInt32(&c.closed) != 0 {
		err = driver.ErrBadConn
		return
	}

	// TODO(xq262144): make use of the ctx argument
	sq := convertQuery(query, args)

	var affectedRows, lastInsertID int64
	if affectedRows, lastInsertID, _, err = c.addQuery(ctx, types.WriteQuery, sq); err != nil {
		return
	}

	result = &execResult{
		affectedRows: affectedRows,
		lastInsertID: lastInsertID,
	}

	return
}

// QueryContext implements the driver.QueryerContext.QueryContext method.
func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	defer trace.StartRegion(ctx, "dbQuery").End()

	if atomic.LoadInt32(&c.closed) != 0 {
		err = driver.ErrBadConn
		return
	}

	// TODO(xq262144): make use of the ctx argument
	sq := convertQuery(query, args)
	_, _, rows, err = c.addQuery(ctx, types.ReadQuery, sq)

	return
}

// Commit implements the driver.Tx.Commit method.
func (c *conn) Commit() (err error) {
	if atomic.LoadInt32(&c.closed) != 0 {
		return driver.ErrBadConn
	}

	if !c.inTransaction {
		return sql.ErrTxDone
	}

	defer func() {
		c.queries = c.queries[:0]
		c.inTransaction = false
	}()

	if len(c.queries) > 0 {
		// send query
		if _, _, _, err = c.sendQuery(context.Background(), types.WriteQuery, c.queries); err != nil {
			return
		}
	}

	return
}

// Rollback implements the driver.Tx.Rollback method.
func (c *conn) Rollback() error {
	if atomic.LoadInt32(&c.closed) != 0 {
		return driver.ErrBadConn
	}

	if !c.inTransaction {
		return sql.ErrTxDone
	}

	defer func() {
		c.queries = c.queries[:0]
		c.inTransaction = false
	}()

	if len(c.queries) == 0 {
		return sql.ErrTxDone
	}

	return nil
}

func (c *conn) addQuery(ctx context.Context, queryType types.QueryType, query *types.Query) (affectedRows int64, lastInsertID int64, rows driver.Rows, err error) {
	if c.inTransaction {
		// check query type, enqueue query
		if queryType == types.ReadQuery {
			// read query is not supported in transaction
			err = ErrQueryInTransaction
			return
		}

		// append queries
		c.queries = append(c.queries, *query)

		log.WithFields(log.Fields{
			"pattern": query.Pattern,
			"args":    query.Args,
		}).Debug("add query to tx")

		return
	}

	log.WithFields(log.Fields{
		"pattern": query.Pattern,
		"args":    query.Args,
	}).Debug("execute query")

	return c.sendQuery(ctx, queryType, []types.Query{*query})
}

func (c *conn) sendQuery(ctx context.Context, queryType types.QueryType, queries []types.Query) (affectedRows int64, lastInsertID int64, rows driver.Rows, err error) {
	var uc *pconn // peer connection used to execute the queries

	uc = c.leader
	// use follower pconn only when the query is readonly
	if queryType == types.ReadQuery && c.follower != nil {
		uc = c.follower
	}
	if uc == nil {
		uc = c.follower
	}

	// allocate sequence
	connID, seqNo := allocateConnAndSeq()
	defer putBackConn(connID)

	defer func() {
		log.WithFields(log.Fields{
			"count":  len(queries),
			"type":   queryType.String(),
			"connID": connID,
			"seqNo":  seqNo,
			"target": uc.pCaller.TargetID,
			"source": c.localNodeID,
		}).WithError(err).Debug("send query")
	}()

	// build request
	req := &types.Request{
		Header: types.SignedRequestHeader{
			RequestHeader: types.RequestHeader{
				QueryType:    queryType,
				NodeID:       c.localNodeID,
				DatabaseID:   c.dbID,
				ConnectionID: connID,
				SeqNo:        seqNo,
				Timestamp:    getLocalTime(),
			},
		},
		Payload: types.RequestPayload{
			Queries: queries,
		},
	}

	if err = req.Sign(c.privKey); err != nil {
		return
	}

	var response types.Response
	if err = uc.pCaller.Call(route.DBSQuery.String(), req, &response); err != nil {
		return
	}
	rows = newRows(&response)

	if queryType == types.WriteQuery {
		affectedRows = response.Header.AffectedRows
		lastInsertID = response.Header.LastInsertID
	}

	// build ack
	func() {
		defer trace.StartRegion(ctx, "ackEnqueue").End()
		uc.ackCh <- &types.Ack{
			Header: types.SignedAckHeader{
				AckHeader: types.AckHeader{
					Response:     response.Header.ResponseHeader,
					ResponseHash: response.Header.Hash(),
					NodeID:       c.localNodeID,
					Timestamp:    getLocalTime(),
				},
			},
		}
	}()

	return
}

func getLocalTime() time.Time {
	return time.Now().UTC()
}

func convertQuery(query string, args []driver.NamedValue) (sq *types.Query) {
	// rebuild args to named args
	sq = &types.Query{
		Pattern: query,
	}

	sq.Args = make([]types.NamedArg, len(args))

	for i, v := range args {
		sq.Args[i].Name = v.Name
		sq.Args[i].Value = v.Value
	}

	return
}
