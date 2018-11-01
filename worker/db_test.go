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
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	bp "github.com/CovenantSQL/CovenantSQL/blockproducer"
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/pow/cpuminer"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/sqlchain"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	"github.com/fortytw2/leaktest"
	. "github.com/smartystreets/goconvey/convey"
)

var rootHash = hash.Hash{}

const PubKeyStorePath = "./public.keystore"

func TestSingleDatabase(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	// init as single node database
	Convey("test database", t, func() {
		var err error
		var server *rpc.Server
		var cleanup func()
		cleanup, server, err = initNode()
		So(err, ShouldBeNil)

		var rootDir string
		rootDir, err = ioutil.TempDir("", "db_test_")
		So(err, ShouldBeNil)

		// create mux service
		kayakMuxService, err := NewDBKayakMuxService("DBKayak", server)
		So(err, ShouldBeNil)

		chainMuxService, err := sqlchain.NewMuxService("sqlchain", server)
		So(err, ShouldBeNil)

		// create peers
		var peers *proto.Peers
		peers, err = getPeers(1)
		So(err, ShouldBeNil)

		// create file
		cfg := &DBConfig{
			DatabaseID:      "TEST",
			DataDir:         rootDir,
			KayakMux:        kayakMuxService,
			ChainMux:        chainMuxService,
			MaxWriteTimeGap: time.Second * 5,
		}

		// create genesis block
		var block *ct.Block
		block, err = createRandomBlock(rootHash, true)
		So(err, ShouldBeNil)

		// create database
		var db *Database
		db, err = NewDatabase(cfg, peers, block)
		So(err, ShouldBeNil)

		Convey("test query rewrite", func() {
			// test query rewrite
			var writeQuery *wt.Request
			var res *wt.Response
			writeQuery, err = buildQuery(wt.WriteQuery, 1, 1, []string{
				"create table test (col1 int, col2 string)",
				"create index test_index on test (col1)",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			// test show tables query
			var readQuery *wt.Request
			readQuery, err = buildQuery(wt.ReadQuery, 1, 2, []string{
				"show tables",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values[0], ShouldResemble, []byte("test"))

			// test show full tables query
			readQuery, err = buildQuery(wt.ReadQuery, 1, 3, []string{
				"show full tables",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values[0], ShouldResemble, []byte("test"))

			// test show create table
			readQuery, err = buildQuery(wt.ReadQuery, 1, 4, []string{
				"show create table test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			byteStr, isByteStr := res.Payload.Rows[0].Values[0].([]byte)
			So(isByteStr, ShouldBeTrue)
			So(strings.ToUpper(string(byteStr)), ShouldContainSubstring, "CREATE")

			// test show table
			readQuery, err = buildQuery(wt.ReadQuery, 1, 5, []string{
				"show table test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(2))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldHaveLength, 6)
			So(res.Payload.Rows[0].Values[1], ShouldResemble, []byte("col1"))
			So(res.Payload.Rows[1].Values, ShouldHaveLength, 6)
			So(res.Payload.Rows[1].Values[1], ShouldResemble, []byte("col2"))

			// test desc table
			readQuery, err = buildQuery(wt.ReadQuery, 1, 6, []string{
				"desc test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(2))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldHaveLength, 6)
			So(res.Payload.Rows[0].Values[1], ShouldResemble, []byte("col1"))
			So(res.Payload.Rows[1].Values, ShouldHaveLength, 6)
			So(res.Payload.Rows[1].Values[1], ShouldResemble, []byte("col2"))

			// test show index from table
			readQuery, err = buildQuery(wt.ReadQuery, 1, 7, []string{
				"show index from table test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values[0], ShouldResemble, []byte("test_index"))
		})

		Convey("test read write", func() {
			// test write query
			var writeQuery *wt.Request
			var res *wt.Response
			writeQuery, err = buildQuery(wt.WriteQuery, 1, 1, []string{
				"create table test (test int)",
				"insert into test values(1)",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)
			So(res.Header.RowCount, ShouldEqual, 0)

			// test select query
			var readQuery *wt.Request
			readQuery, err = buildQuery(wt.ReadQuery, 1, 2, []string{
				"select * from test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Columns, ShouldResemble, []string{"test"})
			So(res.Payload.DeclTypes, ShouldResemble, []string{"int"})
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values[0], ShouldEqual, 1)

			err = db.Shutdown()
			So(err, ShouldBeNil)
		})

		Convey("test invalid request", func() {
			var writeQuery *wt.Request
			var res *wt.Response
			writeQuery, err = buildQuery(wt.WriteQuery, 1, 1, []string{
				"create table test (test int)",
				"insert into test values(1)",
			})
			So(err, ShouldBeNil)

			// request once
			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)
			So(res.Header.RowCount, ShouldEqual, 0)

			// request again with same sequence
			writeQuery, err = buildQuery(wt.WriteQuery, 1, 1, []string{
				"insert into test values(2)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request again with low sequence
			writeQuery, err = buildQuery(wt.WriteQuery, 1, 0, []string{
				"insert into test values(3)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request with invalid timestamp
			writeQuery, err = buildQueryWithTimeShift(wt.WriteQuery, 1, 2, time.Second*100, []string{
				"insert into test values(4)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request with invalid timestamp
			writeQuery, err = buildQueryWithTimeShift(wt.WriteQuery, 1, 2, -time.Second*100, []string{
				"insert into test values(5)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// request with different connection id
			writeQuery, err = buildQuery(wt.WriteQuery, 2, 1, []string{
				"insert into test values(6)",
			})
			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)

			// read query, test records
			var readQuery *wt.Request
			readQuery, err = buildQuery(wt.ReadQuery, 1, 2, []string{
				"select * from test",
			})
			So(err, ShouldBeNil)

			res, err = db.Query(readQuery)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(2))
			So(res.Payload.Columns, ShouldResemble, []string{"test"})
			So(res.Payload.DeclTypes, ShouldResemble, []string{"int"})
			So(res.Payload.Rows, ShouldNotBeEmpty)
			So(len(res.Payload.Rows), ShouldEqual, 2)
			So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
			So(res.Payload.Rows[0].Values[0], ShouldEqual, 1)
			So(res.Payload.Rows[1].Values[0], ShouldEqual, 6)

			err = db.Shutdown()
			So(err, ShouldBeNil)
		})

		Convey("corner case", func() {
			var req *wt.Request
			var err error
			req, err = buildQuery(-1, 1, 1, []string{
				"create table test (test int)",
			})
			So(err, ShouldBeNil)
			_, err = db.Query(req)
			So(err, ShouldNotBeNil)

			var writeQuery *wt.Request
			var res *wt.Response
			writeQuery, err = buildQuery(wt.WriteQuery, 1, 1, []string{
				"create table test (test int)",
			})
			So(err, ShouldBeNil)
			res, err = db.Query(writeQuery)
			So(err, ShouldBeNil)

			// read query, test records
			var readQuery *wt.Request
			readQuery, err = buildQuery(wt.ReadQuery, 1, 2, []string{
				"select * from test",
			})
			So(err, ShouldBeNil)
			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(0))
			So(res.Payload.Columns, ShouldResemble, []string{"test"})
			So(res.Payload.DeclTypes, ShouldResemble, []string{"int"})
			So(res.Payload.Rows, ShouldBeEmpty)

			// write query, test failed
			writeQuery, err = buildQuery(wt.WriteQuery, 1, 3, []string{
				"insert into test2 values(1)", // table should not exists
			})
			So(err, ShouldBeNil)
			res, err = db.Query(writeQuery)
			So(err, ShouldNotBeNil)

			// read query, test dynamic fields
			readQuery, err = buildQuery(wt.ReadQuery, 1, 4, []string{
				"select 1 as test",
			})
			So(err, ShouldBeNil)
			res, err = db.Query(readQuery)
			So(err, ShouldBeNil)
			err = res.Verify()
			So(err, ShouldBeNil)

			So(res.Header.RowCount, ShouldEqual, uint64(1))
			So(res.Payload.Columns, ShouldResemble, []string{"test"})
			So(res.Payload.DeclTypes, ShouldResemble, []string{""})
			So(res.Payload.Rows, ShouldNotBeEmpty)

			// test ack
			var ack *wt.Ack
			ack, err = buildAck(res)
			So(err, ShouldBeNil)

			err = db.Ack(ack)
			So(err, ShouldBeNil)

			// test update peers
			peers, err = getPeers(2)
			So(err, ShouldBeNil)
			err = db.UpdatePeers(peers)
			So(err, ShouldBeNil)
		})

		Reset(func() {
			db.Shutdown()
			os.RemoveAll(rootDir)
			cleanup()
		})
	})
}

func TestInitFailed(t *testing.T) {
	Convey("test database", t, func() {
		var err error
		var server *rpc.Server
		var cleanup func()
		cleanup, server, err = initNode()
		So(err, ShouldBeNil)

		defer cleanup()

		var rootDir string
		rootDir, err = ioutil.TempDir("", "db_test_")
		So(err, ShouldBeNil)

		defer os.RemoveAll(rootDir)

		// create mux service
		kayakMuxService, err := NewDBKayakMuxService("DBKayak", server)
		So(err, ShouldBeNil)

		chainMuxService, err := sqlchain.NewMuxService("sqlchain", server)
		So(err, ShouldBeNil)

		// create peers
		var peers *proto.Peers
		peers, err = getPeers(1)
		So(err, ShouldBeNil)

		// create file
		cfg := &DBConfig{
			DatabaseID:      "TEST",
			DataDir:         rootDir,
			KayakMux:        kayakMuxService,
			ChainMux:        chainMuxService,
			MaxWriteTimeGap: time.Duration(5 * time.Second),
		}

		// create genesis block
		var block *ct.Block
		block, err = createRandomBlock(rootHash, true)
		So(err, ShouldBeNil)

		// broken peers configuration
		peers.Term = 2

		// create database
		_, err = NewDatabase(cfg, peers, block)
		So(err, ShouldNotBeNil)
	})
}

func TestDatabaseRecycle(t *testing.T) {
	defer leaktest.Check(t)()

	// test init/shutdown/destroy
	// test goroutine status
	Convey("test init destroy", t, func() {
		var err error
		var server *rpc.Server
		var cleanup func()
		cleanup, server, err = initNode()
		So(err, ShouldBeNil)

		defer cleanup()

		var rootDir string
		rootDir, err = ioutil.TempDir("", "db_test_")
		So(err, ShouldBeNil)

		// create mux service
		kayakMuxService, err := NewDBKayakMuxService("DBKayak", server)
		So(err, ShouldBeNil)

		chainMuxService, err := sqlchain.NewMuxService("sqlchain", server)
		So(err, ShouldBeNil)

		// create peers
		var peers *proto.Peers
		peers, err = getPeers(1)
		So(err, ShouldBeNil)

		// create file
		cfg := &DBConfig{
			DatabaseID:      "TEST",
			DataDir:         rootDir,
			KayakMux:        kayakMuxService,
			ChainMux:        chainMuxService,
			MaxWriteTimeGap: time.Duration(5 * time.Second),
		}

		// create genesis block
		var block *ct.Block
		block, err = createRandomBlock(rootHash, true)
		So(err, ShouldBeNil)

		// create database
		var db *Database
		db, err = NewDatabase(cfg, peers, block)
		So(err, ShouldBeNil)

		// do some query
		var writeQuery *wt.Request
		var res *wt.Response
		writeQuery, err = buildQuery(wt.WriteQuery, 1, 1, []string{
			"create table test (test int)",
			"insert into test values(1)",
		})
		So(err, ShouldBeNil)

		res, err = db.Query(writeQuery)
		So(err, ShouldBeNil)
		err = res.Verify()
		So(err, ShouldBeNil)
		So(res.Header.RowCount, ShouldEqual, 0)

		// test select query
		var readQuery *wt.Request
		readQuery, err = buildQuery(wt.ReadQuery, 1, 2, []string{
			"select * from test",
		})
		So(err, ShouldBeNil)

		res, err = db.Query(readQuery)
		So(err, ShouldBeNil)
		err = res.Verify()
		So(err, ShouldBeNil)

		So(res.Header.RowCount, ShouldEqual, uint64(1))
		So(res.Payload.Columns, ShouldResemble, []string{"test"})
		So(res.Payload.DeclTypes, ShouldResemble, []string{"int"})
		So(res.Payload.Rows, ShouldNotBeEmpty)
		So(res.Payload.Rows[0].Values, ShouldNotBeEmpty)
		So(res.Payload.Rows[0].Values[0], ShouldEqual, 1)

		// destroy
		err = db.Destroy()
		So(err, ShouldBeNil)
		_, err = os.Stat(rootDir)
		So(err, ShouldNotBeNil)
	})
}

func buildAck(res *wt.Response) (ack *wt.Ack, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get private/public key
	var privateKey *asymmetric.PrivateKey

	if privateKey, _, err = getKeys(); err != nil {
		return
	}

	ack = &wt.Ack{
		Header: wt.SignedAckHeader{
			AckHeader: wt.AckHeader{
				Response:  res.Header,
				NodeID:    nodeID,
				Timestamp: getLocalTime(),
			},
		},
	}

	err = ack.Sign(privateKey, true)

	return
}

func buildQuery(queryType wt.QueryType, connID uint64, seqNo uint64, queries []string) (query *wt.Request, err error) {
	return buildQueryEx(queryType, connID, seqNo, time.Duration(0), proto.DatabaseID(""), queries)
}

func buildQueryWithDatabaseID(queryType wt.QueryType, connID uint64, seqNo uint64, databaseID proto.DatabaseID, queries []string) (query *wt.Request, err error) {
	return buildQueryEx(queryType, connID, seqNo, time.Duration(0), databaseID, queries)
}

func buildQueryWithTimeShift(queryType wt.QueryType, connID uint64, seqNo uint64, timeShift time.Duration, queries []string) (query *wt.Request, err error) {
	return buildQueryEx(queryType, connID, seqNo, timeShift, proto.DatabaseID(""), queries)
}

func buildQueryEx(queryType wt.QueryType, connID uint64, seqNo uint64, timeShift time.Duration, databaseID proto.DatabaseID, queries []string) (query *wt.Request, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get private/public key
	var privateKey *asymmetric.PrivateKey

	if privateKey, _, err = getKeys(); err != nil {
		return
	}

	tm := getLocalTime()
	tm = tm.Add(-timeShift)

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

func getPeers(term uint64) (peers *proto.Peers, err error) {
	// get node id
	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	// get private/public key
	var privateKey *asymmetric.PrivateKey

	if privateKey, _, err = getKeys(); err != nil {
		return
	}

	// generate peers and sign
	peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Term:    term,
			Leader:  nodeID,
			Servers: []proto.NodeID{nodeID},
		},
	}
	err = peers.Sign(privateKey)
	return
}

func getKeys() (privKey *asymmetric.PrivateKey, pubKey *asymmetric.PublicKey, err error) {
	// get public key
	if pubKey, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	// get private key
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	return
}

func initNode() (cleanupFunc func(), server *rpc.Server, err error) {
	var d string
	if d, err = ioutil.TempDir("", "db_test_"); err != nil {
		return
	}

	// init conf
	_, testFile, _, _ := runtime.Caller(0)
	pubKeyStoreFile := filepath.Join(d, PubKeyStorePath)
	os.Remove(pubKeyStoreFile)
	clientPubKeyStoreFile := filepath.Join(d, PubKeyStorePath+"_c")
	os.Remove(clientPubKeyStoreFile)
	dupConfFile := filepath.Join(d, "config.yaml")
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/config.yaml")
	if err = dupConf(confFile, dupConfFile); err != nil {
		return
	}
	privateKeyPath := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/private.key")

	conf.GConf, _ = conf.LoadConfig(dupConfFile)
	// reset the once
	route.Once = sync.Once{}
	route.InitKMS(clientPubKeyStoreFile)

	var dht *route.DHTService

	// init dht
	dht, err = route.NewDHTService(pubKeyStoreFile, new(consistent.KMSStorage), true)
	if err != nil {
		return
	}

	// init rpc
	if server, err = rpc.NewServerWithService(rpc.ServiceMap{route.DHTRPCName: dht}); err != nil {
		return
	}

	// register bpdb service
	if err = server.RegisterService(route.BPDBRPCName, &stubBPDBService{}); err != nil {
		return
	}

	// init private key
	masterKey := []byte("")
	if err = server.InitRPCServer(conf.GConf.ListenAddr, privateKeyPath, masterKey); err != nil {
		return
	}

	// start server
	go server.Serve()

	cleanupFunc = func() {
		os.RemoveAll(d)
		server.Listener.Close()
		server.Stop()
		// clear the connection pool
		rpc.GetSessionPoolInstance().Close()
	}

	return
}

// copied from sqlchain.xxx_test.
func createRandomBlock(parent hash.Hash, isGenesis bool) (b *ct.Block, err error) {
	// Generate key pair
	priv, pub, err := asymmetric.GenSecp256k1KeyPair()

	if err != nil {
		return
	}

	h := hash.Hash{}
	rand.Read(h[:])

	b = &ct.Block{
		SignedHeader: ct.SignedHeader{
			Header: ct.Header{
				Version:     0x01000000,
				Producer:    proto.NodeID(h.String()),
				GenesisHash: rootHash,
				ParentHash:  parent,
				Timestamp:   time.Now().UTC(),
			},
		},
		Queries: make([]*hash.Hash, rand.Intn(10)+10),
	}

	for i := range b.Queries {
		b.Queries[i] = new(hash.Hash)
		rand.Read(b.Queries[i][:])
	}

	if isGenesis {
		// Compute nonce with public key
		nonceCh := make(chan cpuminer.NonceInfo)
		quitCh := make(chan struct{})
		miner := cpuminer.NewCPUMiner(quitCh)
		go miner.ComputeBlockNonce(cpuminer.MiningBlock{
			Data:      pub.Serialize(),
			NonceChan: nonceCh,
			Stop:      nil,
		}, cpuminer.Uint256{A: 0, B: 0, C: 0, D: 0}, 4)
		nonce := <-nonceCh
		close(quitCh)
		close(nonceCh)
		// Add public key to KMS
		id := cpuminer.HashBlock(pub.Serialize(), nonce.Nonce)
		b.SignedHeader.Header.Producer = proto.NodeID(id.String())
		err = kms.SetPublicKey(proto.NodeID(id.String()), nonce.Nonce, pub)

		if err != nil {
			return nil, err
		}
	}

	err = b.PackAndSignBlock(priv)
	return
}

// fake BPDB service
type stubBPDBService struct{}

func (s *stubBPDBService) CreateDatabase(req *bp.CreateDatabaseRequest, resp *bp.CreateDatabaseResponse) (err error) {
	if resp.Header.InstanceMeta, err = s.getInstanceMeta("db2"); err != nil {
		return
	}
	if resp.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	err = resp.Sign(privateKey)
	return
}

func (s *stubBPDBService) DropDatabase(req *bp.DropDatabaseRequest, resp *bp.DropDatabaseRequest) (err error) {
	return
}

func (s *stubBPDBService) GetDatabase(req *bp.GetDatabaseRequest, resp *bp.GetDatabaseResponse) (err error) {
	if resp.Header.InstanceMeta, err = s.getInstanceMeta(req.Header.DatabaseID); err != nil {
		return
	}
	if resp.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		return
	}
	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	err = resp.Sign(privateKey)
	return
}

func (s *stubBPDBService) GetNodeDatabases(req *wt.InitService, resp *wt.InitServiceResponse) (err error) {
	resp.Header.Instances = make([]wt.ServiceInstance, 1)
	resp.Header.Instances[0], err = s.getInstanceMeta("db2")
	if resp.Header.Signee, err = kms.GetLocalPublicKey(); err != nil {
		return
	}

	var privateKey *asymmetric.PrivateKey
	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}
	err = resp.Sign(privateKey)

	return
}

func (s *stubBPDBService) getInstanceMeta(dbID proto.DatabaseID) (instance wt.ServiceInstance, err error) {
	var privKey *asymmetric.PrivateKey
	if privKey, err = kms.GetLocalPrivateKey(); err != nil {
		return
	}

	var nodeID proto.NodeID
	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	instance.DatabaseID = proto.DatabaseID(dbID)
	instance.Peers = &proto.Peers{
		PeersHeader: proto.PeersHeader{
			Term:    1,
			Leader:  nodeID,
			Servers: []proto.NodeID{nodeID},
		},
	}
	if err = instance.Peers.Sign(privKey); err != nil {
		return
	}
	instance.GenesisBlock, err = createRandomBlock(rootHash, true)

	return
}

// duplicate conf file using random new listen addr to avoid failure on concurrent test cases
func dupConf(confFile string, newConfFile string) (err error) {
	// replace port in confFile
	var fileBytes []byte
	if fileBytes, err = ioutil.ReadFile(confFile); err != nil {
		return
	}

	var ports []int
	if ports, err = utils.GetRandomPorts("127.0.0.1", 5000, 6000, 1); err != nil {
		return
	}

	newConfBytes := bytes.Replace(fileBytes, []byte(":2230"), []byte(fmt.Sprintf(":%v", ports[0])), -1)

	return ioutil.WriteFile(newConfFile, newConfBytes, 0644)
}
