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

package client

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/crypto/asymmetric"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/kayak"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/sqlchain/storage"
	wt "gitlab.com/thunderdb/ThunderDB/worker/types"
)

// conn implements an interface sql.Conn.
type conn struct {
	dbID         proto.DatabaseID
	connectionID uint64
	seqNo        uint64

	queries []storage.Query
	peers   *kayak.Peers
	nodeID  proto.NodeID
	privKey *asymmetric.PrivateKey
	pubKey  *asymmetric.PublicKey

	inTransaction bool
	closed        int32
	logger        *log.Logger
}

func newConn(cfg *Config) (c *conn, err error) {
	var logger *log.Logger

	if cfg.Debug {
		logger = log.New()
		logger.SetLevel(log.DebugLevel)
	}

	// generate random connection id
	connID := rand.Int63()
	if connID < 0 {
		connID = -connID
	}

	// get local node id
	var nodeID proto.NodeID
	if nodeID, err = getLocalNodeID(); err != nil {
		return
	}

	// get local private key
	var privKey *asymmetric.PrivateKey
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	// get local public key
	var pubKey *asymmetric.PublicKey
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	c = &conn{
		dbID:         cfg.DatabaseID,
		logger:       logger,
		connectionID: uint64(connID),
		nodeID:       nodeID,
		privKey:      privKey,
		pubKey:       pubKey,
		queries:      make([]storage.Query, 0),
	}

	c.log("new conn database %s", c.dbID)

	// get peers from BP
	if err = c.getPeers(); err != nil {
		return
	}

	return
}

func (c *conn) log(msg ...interface{}) {
	if c.logger != nil {
		c.logger.Println(msg...)
	}
}

// Prepare implements the driver.Conn.Prepare method.
func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

// Close implements the driver.Conn.Close method.
func (c *conn) Close() error {
	// close the meta connection
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.log("closed connection")
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
	c.log("begin transaction tx=%t", c.inTransaction)

	if c.inTransaction {
		return nil, sql.ErrTxDone
	}

	// TODO(xq262144), make use of the ctx argument
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

	// TODO(xq262144), make use of the ctx argument
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

	// TODO(xq262144), make use of the ctx argument
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

func (c *conn) addQuery(queryType wt.QueryType, query *storage.Query) (rows driver.Rows, err error) {
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

	return c.sendQuery(queryType, []storage.Query{*query})
}

func (c *conn) sendQuery(queryType wt.QueryType, queries []storage.Query) (rows driver.Rows, err error) {
	// dial remote node
	var conn net.Conn
	// TODO(xq262144), add connection pool support
	// currently connection pool server endpoint is not fully functional
	if conn, err = rpc.DialToNode(c.peers.Leader.ID, nil); err != nil {
		return
	}
	defer conn.Close()

	// dial
	var client *rpc.Client
	if client, err = rpc.InitClientConn(conn); err != nil {
		return
	}
	defer client.Close()

	// build request
	seqNo := atomic.AddUint64(&c.seqNo, 1)
	req := &wt.Request{
		Header: wt.SignedRequestHeader{
			RequestHeader: wt.RequestHeader{
				QueryType:    queryType,
				NodeID:       c.nodeID,
				DatabaseID:   c.dbID,
				ConnectionID: c.connectionID,
				SeqNo:        seqNo,
				Timestamp:    getLocalTime(),
			},
			Signee: c.pubKey,
		},
		Payload: wt.RequestPayload{
			Queries: queries,
		},
	}

	if err = req.Sign(c.privKey); err != nil {
		return
	}

	var response wt.Response
	if err = client.Call("DBS.Query", req, &response); err != nil {
		return
	}

	// verify response
	if err = response.Verify(); err != nil {
		return
	}

	// build ack
	ack := &wt.Ack{
		Header: wt.SignedAckHeader{
			AckHeader: wt.AckHeader{
				Response:  response.Header,
				NodeID:    c.nodeID,
				Timestamp: getLocalTime(),
			},
			Signee: c.pubKey,
		},
	}

	if err = ack.Sign(c.privKey); err != nil {
		return
	}

	var ackRes wt.AckResponse

	// send ack back
	if err = client.Call("DBS.Ack", ack, &ackRes); err != nil {
		return
	}

	rows = newRows(&response)

	return
}

func (c *conn) getPeers() (err error) {
	// TODO(xq262144), update local peers setting from BP
	// currently set static peers to localhost
	var nodeID proto.NodeID
	if nodeID, err = getLocalNodeID(); err != nil {
		return
	}

	c.peers = &kayak.Peers{
		Leader: &kayak.Server{
			ID: nodeID,
		},
	}

	return
}

func getLocalNodeID() (nodeID proto.NodeID, err error) {
	// TODO(xq262144), to use refactored node id interface by kms
	var rawNodeID []byte
	if rawNodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}
	var h *hash.Hash
	if h, err = hash.NewHash(rawNodeID); err != nil {
		return
	}
	nodeID = proto.NodeID(h.String())
	return
}

func getLocalTime() time.Time {
	return time.Now().UTC()
}

func convertQuery(query string, args []driver.NamedValue) (sq *storage.Query) {
	// rebuild args to named args
	sq = &storage.Query{
		Pattern: query,
	}

	sq.Args = make([]sql.NamedArg, len(args))

	for i, v := range args {
		sq.Args[i] = sql.Named(v.Name, v.Value)
	}

	return
}
