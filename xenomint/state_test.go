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

package xenomint

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/verifier"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/types"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

func TestState(t *testing.T) {
	Convey("Given a chain state object", t, func() {
		var (
			id1          = proto.DatabaseID("db-x1")
			fl1          = path.Join(testingDataDir, fmt.Sprint(t.Name(), "x1"))
			fl2          = path.Join(testingDataDir, fmt.Sprint(t.Name(), "x2"))
			st1, st2     *State
			strg1, strg2 xi.Storage
			err          error
		)
		nodeID := proto.NodeID("0000000000000000000000000000000000000000000000000000000000000000")
		strg1, err = xs.NewSqlite(fmt.Sprint("file:", fl1))
		So(err, ShouldBeNil)
		So(strg1, ShouldNotBeNil)
		st1, err = NewState(nodeID, strg1)
		So(err, ShouldBeNil)
		So(st1, ShouldNotBeNil)
		Reset(func() {
			// Clean database file after each pass
			err = st1.Close(true)
			So(err, ShouldBeNil)
			err = os.Remove(fl1)
			So(err, ShouldBeNil)
			err = os.Remove(fmt.Sprint(fl1, "-shm"))
			So(err == nil || os.IsNotExist(err), ShouldBeTrue)
			err = os.Remove(fmt.Sprint(fl1, "-wal"))
			So(err == nil || os.IsNotExist(err), ShouldBeTrue)
		})
		strg2, err = xs.NewSqlite(fmt.Sprint("file:", fl2))
		So(err, ShouldBeNil)
		So(strg1, ShouldNotBeNil)
		st2, err = NewState(nodeID, strg2)
		So(err, ShouldBeNil)
		So(st1, ShouldNotBeNil)
		Reset(func() {
			// Clean database file after each pass
			err = st2.Close(true)
			So(err, ShouldBeNil)
			err = os.Remove(fl2)
			So(err, ShouldBeNil)
			err = os.Remove(fmt.Sprint(fl2, "-shm"))
			So(err == nil || os.IsNotExist(err), ShouldBeTrue)
			err = os.Remove(fmt.Sprint(fl2, "-wal"))
			So(err == nil || os.IsNotExist(err), ShouldBeTrue)
		})
		Convey("When storage is closed", func() {
			err = st1.Close(false)
			So(err, ShouldBeNil)
			Convey("The storage should report error for any incoming query", func() {
				var req = buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`CREATE TABLE t1 (k INT, v TEXT, PRIMARY KEY(k))`),
				})
				_, _, err = st1.Query(req)
				So(err, ShouldNotBeNil)
				err = errors.Cause(err)
				So(err, ShouldNotBeNil)
				So(err, ShouldEqual, sql.ErrTxDone)
			})
		})
		Convey("The state will report error on read with uncommitted schema change", func() {
			var (
				req = buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`CREATE TABLE t1 (k INT, v TEXT, PRIMARY KEY(k))`),
				})
				resp *types.Response
			)
			_, resp, err = st1.Query(req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
				buildQuery(`SELECT * FROM t1`),
			}))
			// any schema change query will trigger performance degradation mode in current block
			So(err, ShouldBeNil)
		})
		Convey("When a basic KV table is created", func() {
			var (
				values = [][]interface{}{
					{int64(1), []byte("v1")},
					{int64(2), []byte("v2")},
					{int64(3), []byte("v3")},
					{int64(4), []byte("v4")},
				}
				req = buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`CREATE TABLE t1 (k INT, v TEXT, PRIMARY KEY(k))`),
				})
				resp *types.Response
			)
			_, resp, err = st1.Query(req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			err = st1.commit()
			So(err, ShouldBeNil)
			_, resp, err = st2.Query(req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			err = st2.commit()
			Convey("The state should not change after attempted writing in read query", func() {
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?)`, 1, "v1"),
					buildQuery(`SELECT v FROM t1 WHERE k=?`, 1),
				}))
				// The use of Query instead of Exec won't produce an "attempt to write" error
				// like Exec, but it should still keep it readonly -- which means writes will
				// be ignored in this case.
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 0)
			})
			Convey("The state should report invalid request with unknown query type", func() {
				req = buildRequest(types.QueryType(0xff), []types.Query{
					buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?)`, values[0]...),
				})
				_, resp, err = st1.Query(req)
				So(err, ShouldEqual, ErrInvalidRequest)
				So(resp, ShouldBeNil)
				err = st1.Replay(req, nil)
				So(err, ShouldEqual, ErrInvalidRequest)
			})
			Convey("The state should report error on malformed queries", func() {
				_, resp, err = st1.Query(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`XXXXXX INTO t1 (k, v) VALUES (?, ?)`, values[0]...),
				}))
				So(err, ShouldNotBeNil)
				So(resp, ShouldBeNil)
				st1.Stat(id1)
				err = st1.Replay(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`XXXXXX INTO t1 (k, v) VALUES (?, ?)`, values[0]...),
				}), &types.Response{
					Header: types.SignedResponseHeader{
						ResponseHeader: types.ResponseHeader{
							LogOffset: st1.getID(),
						},
					},
				})
				So(err, ShouldNotBeNil)
				_, resp, err = st1.Query(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`INSERT INTO t2 (k, v) VALUES (?, ?)`, values[0]...),
				}))
				So(err, ShouldNotBeNil)
				So(resp, ShouldBeNil)
				st1.Stat(id1)
				err = st1.Replay(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`INSERT INTO t2 (k, v) VALUES (?, ?)`, values[0]...),
				}), &types.Response{
					Header: types.SignedResponseHeader{
						ResponseHeader: types.ResponseHeader{
							LogOffset: st1.getID(),
						},
					},
				})
				So(err, ShouldNotBeNil)
				st1.Stat(id1)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`XXXXXX v FROM t1`),
				}))
				So(err, ShouldNotBeNil)
				So(resp, ShouldBeNil)
				st1.Stat(id1)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT v FROM t2`),
				}))
				So(err, ShouldNotBeNil)
				So(resp, ShouldBeNil)
				st1.Stat(id1)
				_, resp, err = st1.read(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT v FROM t2`),
				}))
				So(err, ShouldNotBeNil)
				So(resp, ShouldBeNil)
				st1.Stat(id1)
			})
			Convey("The state should work properly with reading/writing queries", func() {
				_, resp, err = st1.Query(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?)`, values[0]...),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 0)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT v FROM t1 WHERE k=?`, values[0][0]),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 1)
				So(resp.Payload, ShouldResemble, types.ResponsePayload{
					Columns:   []string{"v"},
					DeclTypes: []string{"TEXT"},
					Rows:      []types.ResponseRow{{Values: values[0][1:]}},
				})
				st1.Stat(id1)

				_, resp, err = st1.Query(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?)`, values[1]...),
					buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?);
INSERT INTO t1 (k, v) VALUES (?, ?)`, concat(values[2:4])...),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 0)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT v FROM t1`),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 4)
				So(resp.Payload, ShouldResemble, types.ResponsePayload{
					Columns:   []string{"v"},
					DeclTypes: []string{"TEXT"},
					Rows: []types.ResponseRow{
						{Values: values[0][1:]},
						{Values: values[1][1:]},
						{Values: values[2][1:]},
						{Values: values[3][1:]},
					},
				})
				st1.Stat(id1)

				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT * FROM t1`),
				}))
				So(err, ShouldBeNil)
				So(resp.Payload, ShouldResemble, types.ResponsePayload{
					Columns:   []string{"k", "v"},
					DeclTypes: []string{"INT", "TEXT"},
					Rows: []types.ResponseRow{
						{Values: values[0][:]},
						{Values: values[1][:]},
						{Values: values[2][:]},
						{Values: values[3][:]},
					},
				})
				st1.Stat(id1)

				// Test show statements
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SHOW TABLE t1`),
				}))
				So(err, ShouldBeNil)
				So(resp, ShouldNotBeNil)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SHOW CREATE TABLE t1`),
				}))
				So(err, ShouldBeNil)
				So(resp, ShouldNotBeNil)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SHOW INDEX FROM TABLE t1`),
				}))
				So(err, ShouldBeNil)
				So(resp, ShouldNotBeNil)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SHOW TABLES`),
				}))
				So(err, ShouldBeNil)
				So(resp, ShouldNotBeNil)
				st1.Stat(id1)

				// Also test a non-transaction read implementation
				_, resp, err = st1.read(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT * FROM t1`),
				}))
				So(err, ShouldBeNil)
				So(resp.Payload, ShouldResemble, types.ResponsePayload{
					Columns:   []string{"k", "v"},
					DeclTypes: []string{"INT", "TEXT"},
					Rows: []types.ResponseRow{
						{Values: values[0][:]},
						{Values: values[1][:]},
						{Values: values[2][:]},
						{Values: values[3][:]},
					},
				})
				st1.Stat(id1)
			})
			Convey("The state should skip read query while replaying", func() {
				err = st1.Replay(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT * FROM t1`),
				}), nil)
				So(err, ShouldBeNil)
			})
			Convey("The state should report conflict state while replaying bad request", func() {
				err = st1.Replay(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?)`, values[0]...),
				}), &types.Response{
					Header: types.SignedResponseHeader{
						ResponseHeader: types.ResponseHeader{
							LogOffset: uint64(0xff),
						},
					},
				})
				err = errors.Cause(err)
				So(err, ShouldEqual, ErrQueryConflict)
			})
			Convey("The state should be reproducible in another instance", func() {
				var (
					qt   *QueryTracker
					reqs = []*types.Request{
						buildRequest(types.WriteQuery, []types.Query{
							buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?)`, values[0]...),
						}),
						buildRequest(types.WriteQuery, []types.Query{
							buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?)`, values[1]...),
							buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?);
INSERT INTO t1 (k, v) VALUES (?, ?)`, concat(values[2:4])...),
						}),
						buildRequest(types.WriteQuery, []types.Query{
							buildQuery(`DELETE FROM t1 WHERE k=?`, values[2][0]),
						}),
					}
				)
				for i := range reqs {
					qt, resp, err = st1.Query(reqs[i])
					So(err, ShouldBeNil)
					So(qt, ShouldNotBeNil)
					So(resp, ShouldNotBeNil)
					qt.UpdateResp(resp)
					// Replay to st2
					err = st2.Replay(reqs[i], resp)
					So(err, ShouldBeNil)
				}
				// Should be in same state
				for i := range values {
					var resp1, resp2 *types.Response
					req = buildRequest(types.ReadQuery, []types.Query{
						buildQuery(`SELECT v FROM t1 WHERE k=?`, values[i][0]),
					})
					_, resp1, err = st1.Query(req)
					So(err, ShouldBeNil)
					So(resp1, ShouldNotBeNil)
					_, resp2, err = st2.Query(req)
					So(err, ShouldBeNil)
					So(resp2, ShouldNotBeNil)
					So(resp1.Payload, ShouldResemble, resp2.Payload)
				}
			})
			Convey("When queries are committed to blocks on state instance #1", func() {
				var (
					qt   *QueryTracker
					reqs = []*types.Request{
						buildRequest(types.WriteQuery, []types.Query{
							buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?)`, values[0]...),
						}),
						buildRequest(types.WriteQuery, []types.Query{
							buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?)`, values[1]...),
							buildQuery(`INSERT INTO t1 (k, v) VALUES (?, ?);
INSERT INTO t1 (k, v) VALUES (?, ?)`, concat(values[2:4])...),
						}),
						buildRequest(types.WriteQuery, []types.Query{
							buildQuery(`DELETE FROM t1 WHERE k=?`, values[2][0]),
						}),
					}

					cmtpos = 0
					cmtps  = []int{1, len(reqs) - 1}
					blocks = make([]*types.Block, len(cmtps))
				)
				for i := range reqs {
					var resp *types.Response
					qt, resp, err = st1.Query(reqs[i])
					So(err, ShouldBeNil)
					So(qt, ShouldNotBeNil)
					So(resp, ShouldNotBeNil)
					qt.UpdateResp(resp)
					// Commit block if matches the next commit point
					if cmtpos < len(cmtps) && i == cmtps[cmtpos] {
						var qts []*QueryTracker
						_, qts, err = st1.CommitEx()
						So(err, ShouldBeNil)
						So(qts, ShouldNotBeNil)
						blocks[cmtpos] = &types.Block{
							QueryTxs: make([]*types.QueryAsTx, len(qts)),
						}
						for i, v := range qts {
							blocks[cmtpos].QueryTxs[i] = &types.QueryAsTx{
								Request:  v.Req,
								Response: &v.Resp.Header,
							}
						}
						cmtpos++
					}
				}
				Convey(
					"The state should report missing parent while replaying later block first",
					func() {
						err = st2.ReplayBlock(blocks[len(blocks)-1])
						So(err, ShouldEqual, ErrMissingParent)
					},
				)
				Convey(
					"The state should report conflict error while replaying modified query",
					func() {
						// Replay by request to st2 first
						for _, v := range blocks {
							for _, w := range v.QueryTxs {
								err = st2.Replay(w.Request, &types.Response{
									Header: *w.Response,
								})
								So(err, ShouldBeNil)
							}
						}
						// Try to replay modified block #0
						var blockx = &types.Block{
							QueryTxs: []*types.QueryAsTx{
								&types.QueryAsTx{
									Request: &types.Request{
										Header: types.SignedRequestHeader{
											DefaultHashSignVerifierImpl: verifier.DefaultHashSignVerifierImpl{
												DataHash: [32]byte{
													0, 0, 0, 0, 0, 0, 0, 1,
													0, 0, 0, 0, 0, 0, 0, 1,
													0, 0, 0, 0, 0, 0, 0, 1,
													0, 0, 0, 0, 0, 0, 0, 1,
												},
											},
										},
									},
									Response: &types.SignedResponseHeader{
										ResponseHeader: types.ResponseHeader{
											LogOffset: blocks[0].QueryTxs[0].Response.LogOffset,
										},
									},
								},
							},
						}
						blockx.QueryTxs[0].Request.Header.DataHash = hash.Hash{0x0, 0x0, 0x0, 0x1}
						err = st2.ReplayBlock(blockx)
						So(err, ShouldEqual, ErrQueryConflict)
					},
				)
				Convey(
					"The state should be reproducible with block replaying in empty instance #2",
					func() {
						// BPBlock replaying
						for i := range blocks {
							err = st2.ReplayBlock(blocks[i])
							So(err, ShouldBeNil)
						}
						// Should be in same state
						for i := range values {
							var resp1, resp2 *types.Response
							req = buildRequest(types.ReadQuery, []types.Query{
								buildQuery(`SELECT v FROM t1 WHERE k=?`, values[i][0]),
							})
							_, resp1, err = st1.Query(req)
							So(err, ShouldBeNil)
							So(resp1, ShouldNotBeNil)
							_, resp2, err = st2.Query(req)
							So(err, ShouldBeNil)
							So(resp2, ShouldNotBeNil)
							So(resp1.Payload, ShouldResemble, resp2.Payload)
						}
					},
				)
				Convey(
					"The state should be reproducible with block replaying in synchronized"+
						" instance #2",
					func() {
						// Replay by request to st2 first
						for _, v := range blocks {
							for _, w := range v.QueryTxs {
								err = st2.Replay(w.Request, &types.Response{
									Header: *w.Response,
								})
								So(err, ShouldBeNil)
							}
						}
						// BPBlock replaying
						for i := range blocks {
							err = st2.ReplayBlock(blocks[i])
							So(err, ShouldBeNil)
						}
						// Should be in same state
						for i := range values {
							var resp1, resp2 *types.Response
							req = buildRequest(types.ReadQuery, []types.Query{
								buildQuery(`SELECT v FROM t1 WHERE k=?`, values[i][0]),
							})
							_, resp1, err = st1.Query(req)
							So(err, ShouldBeNil)
							So(resp1, ShouldNotBeNil)
							_, resp2, err = st2.Query(req)
							So(err, ShouldBeNil)
							So(resp2, ShouldNotBeNil)
							So(resp1.Payload, ShouldResemble, resp2.Payload)
						}
					},
				)
			})
		})
	})
}
