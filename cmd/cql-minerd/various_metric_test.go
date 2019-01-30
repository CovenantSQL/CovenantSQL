// +build !testbinary

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

package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	kw "github.com/CovenantSQL/CovenantSQL/kayak/wal"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils"
	x "github.com/CovenantSQL/CovenantSQL/xenomint"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
	. "github.com/smartystreets/goconvey/convey"
)

func BenchmarkDBWrite(b *testing.B) {
	priv, _, err := asymmetric.GenSecp256k1KeyPair()
	_ = err

	var n proto.NodeID
	var a proto.AccountAddress

	r := &types.Request{
		Header: types.SignedRequestHeader{
			RequestHeader: types.RequestHeader{
				QueryType:    types.WriteQuery,
				NodeID:       n.ToRawNodeID().ToNodeID(),
				DatabaseID:   a.DatabaseID(),
				ConnectionID: 0,
				SeqNo:        1,
				Timestamp:    time.Now().UTC(),
				BatchCount:   1,
			},
		},
		Payload: types.RequestPayload{
			Queries: []types.Query{
				{
					Pattern: "INSERT INTO insert_table0  ( k, v1 ) VALUES(?, ?)",
					Args: []types.NamedArg{
						{
							Value: 1,
						},
						{
							Value: 2,
						},
					},
				},
			},
		},
	}

	err = r.Sign(priv)

	var (
		strg  xi.Storage
		state *x.State
	)
	f, _ := ioutil.TempFile("", "f")
	_ = f.Close()
	_ = os.Remove(f.Name())

	strg, err = xs.NewSqlite(f.Name())
	if err == nil {
		defer strg.Close()
	}
	state = x.NewState(sql.LevelReadUncommitted, n.ToRawNodeID().ToNodeID(), strg)
	defer state.Close(true)

	b.ResetTimer()
	b.Run("commit", func(b *testing.B) {
		r1 := &types.Request{
			Header: types.SignedRequestHeader{
				RequestHeader: types.RequestHeader{
					QueryType:    types.WriteQuery,
					NodeID:       n.ToRawNodeID().ToNodeID(),
					DatabaseID:   a.DatabaseID(),
					ConnectionID: 0,
					SeqNo:        1,
					Timestamp:    time.Now().UTC(),
					BatchCount:   1,
				},
			},
			Payload: types.RequestPayload{
				Queries: []types.Query{
					{
						Pattern: "CREATE TABLE insert_table0 (k int, v1 int)",
					},
				},
			},
		}

		_ = r1.Sign(priv)
		_, _, _ = state.Query(r1, false)

		for i := 0; i != b.N; i++ {
			_, _, _ = state.Query(r, false)
		}
	})
}

func BenchmarkSignSignature(b *testing.B) {
	priv, _, err := asymmetric.GenSecp256k1KeyPair()
	_ = err

	var n proto.NodeID
	var a proto.AccountAddress

	r := &types.Request{
		Header: types.SignedRequestHeader{
			RequestHeader: types.RequestHeader{
				QueryType:    types.WriteQuery,
				NodeID:       n.ToRawNodeID().ToNodeID(),
				DatabaseID:   a.DatabaseID(),
				ConnectionID: 0,
				SeqNo:        1,
				Timestamp:    time.Now().UTC(),
				BatchCount:   1,
			},
		},
		Payload: types.RequestPayload{
			Queries: []types.Query{
				{
					Pattern: "INSERT INTO insert_table0  ( k, v1 ) VALUES(?, ?)",
					Args: []types.NamedArg{
						{
							Value: 1,
						},
						{
							Value: 2,
						},
					},
				},
			},
		},
	}

	b.ResetTimer()
	b.Run("sign", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			err = r.Sign(priv)
		}
	})

	b.ResetTimer()
	b.Run("verify", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			err = r.Verify()
		}
	})

	rs := &types.Response{
		Header: types.SignedResponseHeader{
			ResponseHeader: types.ResponseHeader{
				Request:      r.Header.RequestHeader,
				RequestHash:  r.Header.Hash(),
				NodeID:       n.ToRawNodeID().ToNodeID(),
				Timestamp:    time.Now().UTC(),
				RowCount:     1,
				LogOffset:    1,
				LastInsertID: 1,
				AffectedRows: 1,
			},
		},
	}

	_ = rs

	b.ResetTimer()
	b.Run("sign nested", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			err = rs.BuildHash()
		}
	})

	b.ResetTimer()
	b.Run("verify nested", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			err = rs.VerifyHash()
		}
	})

	var buf *bytes.Buffer

	b.ResetTimer()
	b.Run("encode request", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			buf, _ = utils.EncodeMsgPack(r)
		}
	})

	b.ResetTimer()
	b.Run("decode request", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			var tr *types.Request
			_ = utils.DecodeMsgPack(buf.Bytes(), &tr)
		}
	})

	var buf2 *bytes.Buffer
	l := &kt.Log{
		LogHeader: kt.LogHeader{
			Index:    1,
			Version:  1,
			Type:     kt.LogPrepare,
			Producer: n.ToRawNodeID().ToNodeID(),
		},
		Data: buf.Bytes(),
	}

	b.ResetTimer()
	b.Run("encode to binlog format", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			buf2, _ = utils.EncodeMsgPack(l)
			_ = buf2
		}
	})

	b.ResetTimer()
	b.Run("decode from binlog format", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			var l2 *kt.Log
			_ = utils.DecodeMsgPack(buf2.Bytes(), &l2)
		}
	})

	f, _ := ioutil.TempFile("", "f")
	_ = f.Close()
	_ = os.Remove(f.Name())
	defer os.Remove(f.Name())
	w, _ := kw.NewLevelDBWal(f.Name())
	defer w.Close()

	var index uint64

	b.Run("write wal", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			index = index + 1
			l.Index = index
			_ = w.Write(l)
		}
	})

	b.Run("get wal", func(b *testing.B) {
		for i := 0; i != b.N; i++ {
			index = index - 1
			if index > 0 {
				_, _ = w.Get(index)
			}
		}
	})
}

func TestComputeMetrics(t *testing.T) {
	Convey("compute metrics", t, func() {
		priv, _, err := asymmetric.GenSecp256k1KeyPair()
		So(err, ShouldBeNil)

		var n proto.NodeID
		var a proto.AccountAddress

		r := &types.Request{
			Header: types.SignedRequestHeader{
				RequestHeader: types.RequestHeader{
					QueryType:    types.WriteQuery,
					NodeID:       n.ToRawNodeID().ToNodeID(),
					DatabaseID:   a.DatabaseID(),
					ConnectionID: 0,
					SeqNo:        1,
					Timestamp:    time.Now().UTC(),
					BatchCount:   1,
				},
			},
			Payload: types.RequestPayload{
				Queries: []types.Query{
					{
						Pattern: "INSERT INTO insert_table0  ( k, v1 ) VALUES(?, ?)",
						Args: []types.NamedArg{
							{
								Value: 1,
							},
							{
								Value: 2,
							},
						},
					},
				},
			},
		}

		err = r.Sign(priv)
		So(err, ShouldBeNil)

		buf, err := utils.EncodeMsgPack(r)
		So(err, ShouldBeNil)

		t.Logf("RequestSize: %v", len(buf.Bytes()))

		l := &kt.Log{
			LogHeader: kt.LogHeader{
				Index:    1,
				Version:  1,
				Type:     kt.LogPrepare,
				Producer: n.ToRawNodeID().ToNodeID(),
			},
			Data: buf.Bytes(),
		}

		buf2, err := utils.EncodeMsgPack(l)
		So(err, ShouldBeNil)

		t.Logf("PrepareLogSize: %v", len(buf2.Bytes()))

		respNodeAddr, err := crypto.PubKeyHash(priv.PubKey())
		So(err, ShouldBeNil)

		rs := &types.Response{
			Header: types.SignedResponseHeader{
				ResponseHeader: types.ResponseHeader{
					Request:         r.Header.RequestHeader,
					RequestHash:     r.Header.Hash(),
					NodeID:          n.ToRawNodeID().ToNodeID(),
					ResponseAccount: respNodeAddr,
					Timestamp:       time.Now().UTC(),
					RowCount:        1,
					LogOffset:       1,
					LastInsertID:    1,
					AffectedRows:    1,
				},
			},
		}

		buf3, err := utils.EncodeMsgPack(rs)
		So(err, ShouldBeNil)

		t.Logf("ResponseSize: %v", len(buf3.Bytes()))

		bs := make([]byte, 16)
		binary.BigEndian.PutUint64(bs, 1)
		binary.BigEndian.PutUint64(bs, 2)

		l2 := kt.Log{
			LogHeader: kt.LogHeader{
				Index:    1,
				Version:  1,
				Type:     kt.LogCommit,
				Producer: n.ToRawNodeID().ToNodeID(),
			},
			Data: bs,
		}

		buf4, err := utils.EncodeMsgPack(l2)
		So(err, ShouldBeNil)

		t.Logf("CommitLogSize: %v", len(buf4.Bytes()))
	})
}
