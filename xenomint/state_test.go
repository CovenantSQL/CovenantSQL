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

	"github.com/CovenantSQL/CovenantSQL/types"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

func TestState(t *testing.T) {
	Convey("Given a chain state object", t, func() {
		var (
			fl1          = path.Join(testingDataDir, fmt.Sprint(t.Name(), "x1"))
			fl2          = path.Join(testingDataDir, fmt.Sprint(t.Name(), "x2"))
			st1, st2     *State
			strg1, strg2 xi.Storage
			err          error
		)
		strg1, err = xs.NewSqlite(fmt.Sprint("file:", fl1))
		So(err, ShouldBeNil)
		So(strg1, ShouldNotBeNil)
		st1, err = NewState(strg1)
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
		st2, err = NewState(strg2)
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
					buildQuery(`CREATE TABLE "t1" ("k" INT, "v" TEXT, PRIMARY KEY("k"))`),
				})
				_, _, err = st1.Query(req)
				So(err, ShouldNotBeNil)
				err = errors.Cause(err)
				So(err, ShouldNotBeNil)
				So(err, ShouldEqual, sql.ErrTxDone)
			})
		})
		Convey("The state will report error on read with uncommitted schema change", func() {
			// TODO(leventeliu): this case is not quite user friendly.
			var (
				req = buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`CREATE TABLE "t1" ("k" INT, "v" TEXT, PRIMARY KEY("k"))`),
				})
				resp *types.Response
			)
			_, resp, err = st1.Query(req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
				buildQuery(`SELECT * FROM "t1"`),
			}))
			So(err, ShouldNotBeNil)
			err = errors.Cause(err)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "database schema is locked: main")
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
					buildQuery(`CREATE TABLE "t1" ("k" INT, "v" TEXT, PRIMARY KEY("k"))`),
				})
				resp *types.Response
			)
			_, resp, err = st1.Query(req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			err = st1.commit(nil)
			So(err, ShouldBeNil)
			_, resp, err = st2.Query(req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			err = st2.commit(nil)
			Convey("The state should not change after attempted writing in read query", func() {
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 1, "v1"),
					buildQuery(`SELECT "v" FROM "t1" WHERE "k"=?`, 1),
				}))
				// The use of Query instead of Exec won't produce an "attempt to write" error
				// like Exec, but it should still keep it readonly -- which means writes will
				// be ignored in this case.
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 0)
			})
			Convey("The state should report invalid request with unknown query type", func() {
				req = buildRequest(types.QueryType(0xff), []types.Query{
					buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, values[0]...),
				})
				_, resp, err = st1.Query(req)
				So(err, ShouldEqual, ErrInvalidRequest)
				So(resp, ShouldBeNil)
				err = st1.Replay(req, nil)
				So(err, ShouldEqual, ErrInvalidRequest)
			})
			Convey("The state should report error on malformed queries", func() {
				_, resp, err = st1.Query(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`XXXXXX INTO "t1" ("k", "v") VALUES (?, ?)`, values[0]...),
				}))
				So(err, ShouldNotBeNil)
				So(resp, ShouldBeNil)
				_, resp, err = st1.Query(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`INSERT INTO "t2" ("k", "v") VALUES (?, ?)`, values[0]...),
				}))
				So(err, ShouldNotBeNil)
				So(resp, ShouldBeNil)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`XXXXXX "v" FROM "t1"`),
				}))
				So(err, ShouldNotBeNil)
				So(resp, ShouldBeNil)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT "v" FROM "t2"`),
				}))
				So(err, ShouldNotBeNil)
				So(resp, ShouldBeNil)
			})
			Convey("The state should work properly with reading/writing queries", func() {
				_, resp, err = st1.Query(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, values[0]...),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 0)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT "v" FROM "t1" WHERE "k"=?`, values[0][0]),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 1)
				So(resp.Payload, ShouldResemble, types.ResponsePayload{
					Columns:   []string{"v"},
					DeclTypes: []string{"TEXT"},
					Rows:      []types.ResponseRow{{Values: values[0][1:]}},
				})

				_, resp, err = st1.Query(buildRequest(types.WriteQuery, []types.Query{
					buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, values[1]...),
					buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?);
INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, concat(values[2:4])...),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 0)
				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT "v" FROM "t1"`),
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

				_, resp, err = st1.Query(buildRequest(types.ReadQuery, []types.Query{
					buildQuery(`SELECT * FROM "t1"`),
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
			})
			Convey("The state should be reproducable in another instance", func() {
				var (
					qt   *QueryTracker
					reqs = []*types.Request{
						buildRequest(types.WriteQuery, []types.Query{
							buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, values[0]...),
						}),
						buildRequest(types.WriteQuery, []types.Query{
							buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, values[1]...),
							buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?);
INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, concat(values[2:4])...),
						}),
						buildRequest(types.WriteQuery, []types.Query{
							buildQuery(`DELETE FROM "t1" WHERE "k"=?`, values[2][0]),
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
					err = st2.replay(reqs[i], resp)
					So(err, ShouldBeNil)
				}
				// Should be in same state
				for i := range values {
					var resp1, resp2 *types.Response
					req = buildRequest(types.ReadQuery, []types.Query{
						buildQuery(`SELECT "v" FROM "t1" WHERE "k"=?`, values[i][0]),
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
		})
	})
}
