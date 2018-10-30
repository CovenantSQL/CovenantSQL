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
	"github.com/CovenantSQL/CovenantSQL/kayak"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
)

// conn implements an interface sql.Conn.
type conn struct {
	dbID proto.DatabaseID

	queries     []wt.Query
	localNodeID proto.NodeID
	privKey     *asymmetric.PrivateKey

	ackCh         chan *wt.Ack
	inTransaction bool
	closed        int32
	pCaller       *rpc.PersistentCaller
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
		queries:     make([]wt.Query, 0),
	}

	// get peers from BP
	if _, err = cacheGetPeers(c.dbID, c.privKey); err != nil {
		log.Errorf("cacheGetPeers failed: %v", err)
		c = nil
		return
	}

	err = c.startAckWorkers(2)
	if err != nil {
		log.Errorf("startAckWorkers failed: %v", err)
		c = nil
		return
	}
	log.WithField("db", c.dbID).Debug("new connection to database")

	return
}

func (c *conn) startAckWorkers(workerCount int) (err error) {
	c.ackCh = make(chan *wt.Ack, workerCount*4)
	for i := 0; i < workerCount; i++ {
		go c.ackWorker()
	}
	return
}

func (c *conn) stopAckWorkers() {
	close(c.ackCh)
}

func (c *conn) ackWorker() {
	if rawPeers, ok := peerList.Load(c.dbID); ok {
		if peers, ok := rawPeers.(*kayak.Peers); ok {
			var (
				oneTime sync.Once
				pc      *rpc.PersistentCaller
				err     error
			)

		ackWorkerLoop:
			for {
				ack, got := <-c.ackCh
				if !got { //closed and empty
					break ackWorkerLoop
				}
				oneTime.Do(func() {
					pc = rpc.NewPersistentCaller(peers.Leader.ID)
				})
				if err = ack.Sign(c.privKey, false); err != nil {
					log.Errorf("failed to sign ack for %s: %v", pc.TargetID, err)
					continue
				}

				var ackRes wt.AckResponse
				// send ack back
				if err = pc.Call(route.DBSAck.String(), ack, &ackRes); err != nil {
					log.Warningf("send ack failed: %v", err)
					continue
				}
			}
			if pc != nil {
				pc.CloseStream()
			}
			log.Debug("ack worker quiting")
			return
		}
	}

	log.Fatal("must GetPeers first")
	return
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
	c.stopAckWorkers()
	c.pCaller.CloseStream()
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

	// prepare the statement
	return newStmt(c, query), nil
}

// ExecContext implements the driver.ExecerContext.ExecContext method.
func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (result driver.Result, err error) {
	if atomic.LoadInt32(&c.closed) != 0 {
		err = driver.ErrBadConn
		return
	}

	// TODO(xq262144): make use of the ctx argument
	sq := convertQuery(query, args)
	if _, err = c.addQuery(wt.WriteQuery, sq); err != nil {
		return
	}

	result = driver.ResultNoRows

	return
}

// QueryContext implements the driver.QueryerContext.QueryContext method.
func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (rows driver.Rows, err error) {
	if atomic.LoadInt32(&c.closed) != 0 {
		err = driver.ErrBadConn
		return
	}

	// TODO(xq262144): make use of the ctx argument
	sq := convertQuery(query, args)
	return c.addQuery(wt.ReadQuery, sq)
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
		if _, err = c.sendQuery(wt.WriteQuery, c.queries); err != nil {
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

func (c *conn) addQuery(queryType wt.QueryType, query *wt.Query) (rows driver.Rows, err error) {
	if c.inTransaction {
		// check query type, enqueue query
		if queryType == wt.ReadQuery {
			// read query is not supported in transaction
			err = ErrQueryInTransaction
			return
		}

		// append queries
		c.queries = append(c.queries, *query)
		return
	}

	return c.sendQuery(queryType, []wt.Query{*query})
}

func (c *conn) sendQuery(queryType wt.QueryType, queries []wt.Query) (rows driver.Rows, err error) {
	var peers *kayak.Peers
	if peers, err = cacheGetPeers(c.dbID, c.privKey); err != nil {
		return
	}

	// allocate sequence
	connID, seqNo := allocateConnAndSeq()
	defer putBackConn(connID)

	// build request
	req := &wt.Request{
		Header: wt.SignedRequestHeader{
			RequestHeader: wt.RequestHeader{
				QueryType:    queryType,
				NodeID:       c.localNodeID,
				DatabaseID:   c.dbID,
				ConnectionID: connID,
				SeqNo:        seqNo,
				Timestamp:    getLocalTime(),
			},
		},
		Payload: wt.RequestPayload{
			Queries: queries,
		},
	}

	if err = req.Sign(c.privKey); err != nil {
		return
	}

	c.pCaller = rpc.NewPersistentCaller(peers.Leader.ID)
	var response wt.Response
	if err = c.pCaller.Call(route.DBSQuery.String(), req, &response); err != nil {
		return
	}

	// verify response
	if err = response.Verify(); err != nil {
		return
	}
	rows = newRows(&response)

	// build ack
	c.ackCh <- &wt.Ack{
		Header: wt.SignedAckHeader{
			AckHeader: wt.AckHeader{
				Response:  response.Header,
				NodeID:    c.localNodeID,
				Timestamp: getLocalTime(),
			},
		},
	}

	return
}

func getLocalTime() time.Time {
	return time.Now().UTC()
}

func convertQuery(query string, args []driver.NamedValue) (sq *wt.Query) {
	// rebuild args to named args
	sq = &wt.Query{
		Pattern: query,
	}

	sq.Args = make([]sql.NamedArg, len(args))

	for i, v := range args {
		sq.Args[i] = sql.Named(v.Name, v.Value)
	}

	return
}
