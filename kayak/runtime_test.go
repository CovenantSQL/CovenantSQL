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

package kayak_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	kl "github.com/CovenantSQL/CovenantSQL/kayak/wal"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/storage"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	mock_conn "github.com/jordwest/mock-conn"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type sqliteStorage struct {
	st  *storage.Storage
	dsn string
}

type queryStructure struct {
	ConnID    uint64
	SeqNo     uint64
	Timestamp int64
	Queries   []storage.Query
}

func newSQLiteStorage(dsn string) (s *sqliteStorage, err error) {
	s = &sqliteStorage{}
	s.st, err = storage.New(dsn)
	s.dsn = dsn
	return
}

func (s *sqliteStorage) EncodePayload(request interface{}) (data []byte, err error) {
	var buf *bytes.Buffer
	if buf, err = utils.EncodeMsgPack(request); err != nil {
		err = errors.Wrap(err, "encode payload failed")
		return
	}

	data = buf.Bytes()
	return
}

func (s *sqliteStorage) DecodePayload(data []byte) (request interface{}, err error) {
	var req *queryStructure
	if err = utils.DecodeMsgPack(data, &req); err != nil {
		err = errors.Wrap(err, "decode payload failed")
		return
	}

	request = req
	return
}

func (s *sqliteStorage) Check(data interface{}) (err error) {
	// no check
	return nil
}

func (s *sqliteStorage) Commit(data interface{}) (result interface{}, err error) {
	var d *queryStructure
	var ok bool
	if d, ok = data.(*queryStructure); !ok {
		err = errors.New("invalid data")
		return
	}

	result, err = s.st.Exec(context.Background(), d.Queries)

	return
}

func (s *sqliteStorage) Query(ctx context.Context, queries []storage.Query) (columns []string, types []string,
	data [][]interface{}, err error) {
	return s.st.Query(ctx, queries)
}

func (s *sqliteStorage) Close() {
	if s.st != nil {
		s.st.Close()
	}
}

type fakeMux struct {
	mux map[proto.NodeID]*fakeService
}

func newFakeMux() *fakeMux {
	return &fakeMux{
		mux: make(map[proto.NodeID]*fakeService),
	}
}

func (m *fakeMux) register(nodeID proto.NodeID, s *fakeService) {
	m.mux[nodeID] = s
}

func (m *fakeMux) get(nodeID proto.NodeID) *fakeService {
	return m.mux[nodeID]
}

type fakeService struct {
	rt *kayak.Runtime
	s  *rpc.Server
}

func newFakeService(rt *kayak.Runtime) (fs *fakeService) {
	fs = &fakeService{
		rt: rt,
		s:  rpc.NewServer(),
	}

	fs.s.RegisterName("Test", fs)

	return
}

func (s *fakeService) Call(req *kt.ApplyRequest, resp *interface{}) (err error) {
	return s.rt.FollowerApply(req.Log)
}

func (s *fakeService) serveConn(c net.Conn) {
	s.s.ServeCodec(utils.GetMsgPackServerCodec(c))
}

type fakeCaller struct {
	m      *fakeMux
	target proto.NodeID
}

func newFakeCaller(m *fakeMux, nodeID proto.NodeID) *fakeCaller {
	return &fakeCaller{
		m:      m,
		target: nodeID,
	}
}

func (c *fakeCaller) Call(method string, req interface{}, resp interface{}) (err error) {
	fakeConn := mock_conn.NewConn()

	go c.m.get(c.target).serveConn(fakeConn.Server)
	client := rpc.NewClientWithCodec(utils.GetMsgPackClientCodec(fakeConn.Client))
	defer client.Close()

	return client.Call(method, req, resp)
}

func TestRuntime(t *testing.T) {
	Convey("runtime test", t, func(c C) {
		lvl := log.GetLevel()
		log.SetLevel(log.FatalLevel)
		defer log.SetLevel(lvl)
		db1, err := newSQLiteStorage("test1.db")
		So(err, ShouldBeNil)
		defer func() {
			db1.Close()
			os.Remove("test1.db")
		}()
		db2, err := newSQLiteStorage("test2.db")
		So(err, ShouldBeNil)
		defer func() {
			db2.Close()
			os.Remove("test2.db")
		}()

		node1 := proto.NodeID("000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade")
		node2 := proto.NodeID("000005f4f22c06f76c43c4f48d5a7ec1309cc94030cbf9ebae814172884ac8b5")

		peers := &proto.Peers{
			PeersHeader: proto.PeersHeader{
				Leader: node1,
				Servers: []proto.NodeID{
					node1,
					node2,
				},
			},
		}

		privKey, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		err = peers.Sign(privKey)
		So(err, ShouldBeNil)

		wal1 := kl.NewMemWal()
		defer wal1.Close()
		cfg1 := &kt.RuntimeConfig{
			Handler:          db1,
			PrepareThreshold: 1.0,
			CommitThreshold:  1.0,
			PrepareTimeout:   time.Second,
			CommitTimeout:    10 * time.Second,
			Peers:            peers,
			Wal:              wal1,
			NodeID:           node1,
			ServiceName:      "Test",
			MethodName:       "Call",
		}
		rt1, err := kayak.NewRuntime(cfg1)
		So(err, ShouldBeNil)

		wal2 := kl.NewMemWal()
		defer wal2.Close()
		cfg2 := &kt.RuntimeConfig{
			Handler:          db2,
			PrepareThreshold: 1.0,
			CommitThreshold:  1.0,
			PrepareTimeout:   time.Second,
			CommitTimeout:    10 * time.Second,
			Peers:            peers,
			Wal:              wal2,
			NodeID:           node2,
			ServiceName:      "Test",
			MethodName:       "Call",
		}
		rt2, err := kayak.NewRuntime(cfg2)
		So(err, ShouldBeNil)

		m := newFakeMux()
		fs1 := newFakeService(rt1)
		m.register(node1, fs1)
		fs2 := newFakeService(rt2)
		m.register(node2, fs2)

		rt1.SetCaller(node2, newFakeCaller(m, node2))
		rt2.SetCaller(node1, newFakeCaller(m, node1))

		err = rt1.Start()
		So(err, ShouldBeNil)
		defer rt1.Shutdown()

		err = rt2.Start()
		So(err, ShouldBeNil)
		defer rt2.Shutdown()

		q1 := &queryStructure{
			Queries: []storage.Query{
				{Pattern: "CREATE TABLE IF NOT EXISTS test (t1 text, t2 text, t3 text)"},
			},
		}
		So(err, ShouldBeNil)

		r1 := RandStringRunes(333)
		r2 := RandStringRunes(333)
		r3 := RandStringRunes(333)

		q2 := &queryStructure{
			Queries: []storage.Query{
				{
					Pattern: "INSERT INTO test (t1, t2, t3) VALUES(?, ?, ?)",
					Args: []sql.NamedArg{
						sql.Named("", r1),
						sql.Named("", r2),
						sql.Named("", r3),
					},
				},
			},
		}

		rt1.Apply(context.Background(), q1)
		rt2.Apply(context.Background(), q2)
		rt1.Apply(context.Background(), q2)
		db1.Query(context.Background(), []storage.Query{
			{Pattern: "SELECT * FROM test"},
		})

		var count uint64
		atomic.StoreUint64(&count, 1)

		for i := 0; i != 1000; i++ {
			atomic.AddUint64(&count, 1)
			q := &queryStructure{
				Queries: []storage.Query{
					{
						Pattern: "INSERT INTO test (t1, t2, t3) VALUES(?, ?, ?)",
						Args: []sql.NamedArg{
							sql.Named("", r1),
							sql.Named("", r2),
							sql.Named("", r3),
						},
					},
				},
			}

			_, _, err = rt1.Apply(context.Background(), q)
			So(err, ShouldBeNil)
		}

		// test rollback
		q := &queryStructure{
			Queries: []storage.Query{
				{
					Pattern: "INVALID QUERY",
				},
			},
		}
		_, _, err = rt1.Apply(context.Background(), q)
		So(err, ShouldNotBeNil)

		// test timeout
		q = &queryStructure{
			Queries: []storage.Query{
				{
					Pattern: "INSERT INTO test (t1, t2, t3) VALUES(?, ?, ?)",
					Args: []sql.NamedArg{
						sql.Named("", r1),
						sql.Named("", r2),
						sql.Named("", r3),
					},
				},
			},
		}
		cancelCtx, cancelCtxFunc := context.WithCancel(context.Background())
		cancelCtxFunc()
		_, _, err = rt1.Apply(cancelCtx, q)
		So(err, ShouldNotBeNil)

		total := atomic.LoadUint64(&count)
		_, _, d1, _ := db1.Query(context.Background(), []storage.Query{
			{Pattern: "SELECT COUNT(1) FROM test"},
		})
		So(d1, ShouldHaveLength, 1)
		So(d1[0], ShouldHaveLength, 1)
		So(fmt.Sprint(d1[0][0]), ShouldEqual, fmt.Sprint(total))

		_, _, d2, _ := db2.Query(context.Background(), []storage.Query{
			{Pattern: "SELECT COUNT(1) FROM test"},
		})
		So(d2, ShouldHaveLength, 1)
		So(d2[0], ShouldHaveLength, 1)
		So(fmt.Sprint(d2[0][0]), ShouldResemble, fmt.Sprint(total))
	})
	Convey("trivial cases", t, func() {
		node1 := proto.NodeID("000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade")
		node2 := proto.NodeID("000005f4f22c06f76c43c4f48d5a7ec1309cc94030cbf9ebae814172884ac8b5")
		node3 := proto.NodeID("000003f49592f83d0473bddb70d543f1096b4ffed5e5f942a3117e256b7052b8")

		peers := &proto.Peers{
			PeersHeader: proto.PeersHeader{
				Leader: node1,
				Servers: []proto.NodeID{
					node1,
					node2,
				},
			},
		}

		_, err := kayak.NewRuntime(nil)
		So(err, ShouldNotBeNil)
		_, err = kayak.NewRuntime(&kt.RuntimeConfig{})
		So(err, ShouldNotBeNil)
		_, err = kayak.NewRuntime(&kt.RuntimeConfig{
			Peers: peers,
		})
		So(err, ShouldNotBeNil)

		privKey, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		err = peers.Sign(privKey)
		So(err, ShouldBeNil)

		_, err = kayak.NewRuntime(&kt.RuntimeConfig{
			Peers:  peers,
			NodeID: node3,
		})
		So(err, ShouldNotBeNil)
	})
	Convey("test log loading", t, func() {
		w, err := kl.NewLevelDBWal("testLoad.db")
		defer os.RemoveAll("testLoad.db")
		So(err, ShouldBeNil)
		err = w.Write(&kt.Log{
			LogHeader: kt.LogHeader{
				Index:    0,
				Type:     kt.LogPrepare,
				Producer: proto.NodeID("0000000000000000000000000000000000000000000000000000000000000000"),
			},
			Data: []byte("happy1"),
		})
		So(err, ShouldBeNil)
		err = w.Write(&kt.Log{
			LogHeader: kt.LogHeader{
				Index:    1,
				Type:     kt.LogPrepare,
				Producer: proto.NodeID("0000000000000000000000000000000000000000000000000000000000000000"),
			},
			Data: []byte("happy1"),
		})
		So(err, ShouldBeNil)
		data := make([]byte, 16)
		binary.BigEndian.PutUint64(data, 0) // prepare log index
		binary.BigEndian.PutUint64(data, 0) // last commit index
		err = w.Write(&kt.Log{
			LogHeader: kt.LogHeader{
				Index:    2,
				Type:     kt.LogCommit,
				Producer: proto.NodeID("0000000000000000000000000000000000000000000000000000000000000000"),
			},
			Data: data,
		})
		So(err, ShouldBeNil)
		data = make([]byte, 8)
		binary.BigEndian.PutUint64(data, 1) // prepare log index
		err = w.Write(&kt.Log{
			LogHeader: kt.LogHeader{
				Index:    3,
				Type:     kt.LogRollback,
				Producer: proto.NodeID("0000000000000000000000000000000000000000000000000000000000000000"),
			},
			Data: data,
		})
		So(err, ShouldBeNil)
		w.Close()

		w, err = kl.NewLevelDBWal("testLoad.db")
		So(err, ShouldBeNil)
		defer w.Close()

		node1 := proto.NodeID("000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade")
		peers := &proto.Peers{
			PeersHeader: proto.PeersHeader{
				Leader:  node1,
				Servers: []proto.NodeID{node1},
			},
		}

		privKey, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		err = peers.Sign(privKey)
		So(err, ShouldBeNil)

		cfg := &kt.RuntimeConfig{
			Handler:          nil,
			PrepareThreshold: 1.0,
			CommitThreshold:  1.0,
			PrepareTimeout:   time.Second,
			CommitTimeout:    10 * time.Second,
			Peers:            peers,
			Wal:              w,
			NodeID:           node1,
			ServiceName:      "Test",
			MethodName:       "Call",
		}
		rt, err := kayak.NewRuntime(cfg)
		So(err, ShouldBeNil)

		So(rt.Start(), ShouldBeNil)
		So(func() { rt.Start() }, ShouldNotPanic)

		So(rt.Shutdown(), ShouldBeNil)
		So(func() { rt.Shutdown() }, ShouldNotPanic)
	})
}

func BenchmarkRuntime(b *testing.B) {
	Convey("runtime test", b, func(c C) {
		log.SetLevel(log.DebugLevel)
		f, err := os.OpenFile("test.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		So(err, ShouldBeNil)
		log.SetOutput(f)
		defer f.Close()

		db1, err := newSQLiteStorage("test1.db")
		So(err, ShouldBeNil)
		defer func() {
			db1.Close()
			os.Remove("test1.db")
		}()
		db2, err := newSQLiteStorage("test2.db")
		So(err, ShouldBeNil)
		defer func() {
			db2.Close()
			os.Remove("test2.db")
		}()

		node1 := proto.NodeID("000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade")
		node2 := proto.NodeID("000005f4f22c06f76c43c4f48d5a7ec1309cc94030cbf9ebae814172884ac8b5")

		peers := &proto.Peers{
			PeersHeader: proto.PeersHeader{
				Leader: node1,
				Servers: []proto.NodeID{
					node1,
					node2,
				},
			},
		}

		privKey, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)
		err = peers.Sign(privKey)
		So(err, ShouldBeNil)

		wal1 := kl.NewMemWal()
		defer wal1.Close()
		cfg1 := &kt.RuntimeConfig{
			Handler:          db1,
			PrepareThreshold: 1.0,
			CommitThreshold:  1.0,
			PrepareTimeout:   time.Second,
			CommitTimeout:    10 * time.Second,
			Peers:            peers,
			Wal:              wal1,
			NodeID:           node1,
			ServiceName:      "Test",
			MethodName:       "Call",
		}
		rt1, err := kayak.NewRuntime(cfg1)
		So(err, ShouldBeNil)

		wal2 := kl.NewMemWal()
		defer wal2.Close()
		cfg2 := &kt.RuntimeConfig{
			Handler:          db2,
			PrepareThreshold: 1.0,
			CommitThreshold:  1.0,
			PrepareTimeout:   time.Second,
			CommitTimeout:    10 * time.Second,
			Peers:            peers,
			Wal:              wal2,
			NodeID:           node2,
			ServiceName:      "Test",
			MethodName:       "Call",
		}
		rt2, err := kayak.NewRuntime(cfg2)
		So(err, ShouldBeNil)

		m := newFakeMux()
		fs1 := newFakeService(rt1)
		m.register(node1, fs1)
		fs2 := newFakeService(rt2)
		m.register(node2, fs2)

		rt1.SetCaller(node2, newFakeCaller(m, node2))
		rt2.SetCaller(node1, newFakeCaller(m, node1))

		err = rt1.Start()
		So(err, ShouldBeNil)
		defer rt1.Shutdown()

		err = rt2.Start()
		So(err, ShouldBeNil)
		defer rt2.Shutdown()

		q1 := &queryStructure{
			Queries: []storage.Query{
				{Pattern: "CREATE TABLE IF NOT EXISTS test (t1 text, t2 text, t3 text)"},
			},
		}
		So(err, ShouldBeNil)

		r1 := RandStringRunes(333)
		r2 := RandStringRunes(333)
		r3 := RandStringRunes(333)

		q2 := &queryStructure{
			Queries: []storage.Query{
				{
					Pattern: "INSERT INTO test (t1, t2, t3) VALUES(?, ?, ?)",
					Args: []sql.NamedArg{
						sql.Named("", r1),
						sql.Named("", r2),
						sql.Named("", r3),
					},
				},
			},
		}

		rt1.Apply(context.Background(), q1)
		rt2.Apply(context.Background(), q2)
		rt1.Apply(context.Background(), q2)
		db1.Query(context.Background(), []storage.Query{
			{Pattern: "SELECT * FROM test"},
		})

		b.ResetTimer()

		var count uint64
		atomic.StoreUint64(&count, 1)

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				atomic.AddUint64(&count, 1)
				q := &queryStructure{
					Queries: []storage.Query{
						{
							Pattern: "INSERT INTO test (t1, t2, t3) VALUES(?, ?, ?)",
							Args: []sql.NamedArg{
								sql.Named("", r1),
								sql.Named("", r2),
								sql.Named("", r3),
							},
						},
					},
				}
				_ = err
				//c.So(err, ShouldBeNil)

				_, _, err = rt1.Apply(context.Background(), q)
				//c.So(err, ShouldBeNil)
			}
		})

		b.StopTimer()

		total := atomic.LoadUint64(&count)
		_, _, d1, _ := db1.Query(context.Background(), []storage.Query{
			{Pattern: "SELECT COUNT(1) FROM test"},
		})
		So(d1, ShouldHaveLength, 1)
		So(d1[0], ShouldHaveLength, 1)
		So(fmt.Sprint(d1[0][0]), ShouldEqual, fmt.Sprint(total))

		//_, _, d2, _ := db2.Query(context.Background(), []storage.Query{
		//	{Pattern: "SELECT COUNT(1) FROM test"},
		//})
		//So(d2, ShouldHaveLength, 1)
		//So(d2[0], ShouldHaveLength, 1)
		//So(fmt.Sprint(d2[0][0]), ShouldResemble, fmt.Sprint(total))

		b.StartTimer()
	})
}
