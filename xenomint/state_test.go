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

	wt "github.com/CovenantSQL/CovenantSQL/worker/types"
	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	xs "github.com/CovenantSQL/CovenantSQL/xenomint/sqlite"
	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"
)

func buildQuery(query string, args ...interface{}) wt.Query {
	var nargs = make([]sql.NamedArg, len(args))
	for i := range args {
		nargs[i] = sql.NamedArg{
			Name:  "",
			Value: args[i],
		}
	}
	return wt.Query{
		Pattern: query,
		Args:    nargs,
	}
}

func buildRequest(qt wt.QueryType, qs []wt.Query) *wt.Request {
	return &wt.Request{
		Header: wt.SignedRequestHeader{
			RequestHeader: wt.RequestHeader{
				QueryType: qt,
			},
		},
		Payload: wt.RequestPayload{Queries: qs},
	}
}

func TestState(t *testing.T) {
	Convey("Given a chain state object", t, func() {
		var (
			fl     = path.Join(testDataDir, t.Name())
			st     *state
			closed bool
			strg   xi.Storage
			err    error
		)
		strg, err = xs.NewSqlite(fmt.Sprint("file:", fl))
		So(err, ShouldBeNil)
		So(strg, ShouldNotBeNil)
		st, err = newState(strg)
		So(err, ShouldBeNil)
		So(st, ShouldNotBeNil)
		Reset(func() {
			// Clean database file after each pass
			if !closed {
				err = st.close(true)
				So(err, ShouldBeNil)
				closed = true
			}
			err = os.Remove(fl)
			So(err, ShouldBeNil)
			err = os.Remove(fmt.Sprint(fl, "-shm"))
			So(err == nil || os.IsNotExist(err), ShouldBeTrue)
			err = os.Remove(fmt.Sprint(fl, "-wal"))
			So(err == nil || os.IsNotExist(err), ShouldBeTrue)
		})
		Convey("When storage is closed", func() {
			err = st.close(false)
			So(err, ShouldBeNil)
			closed = true
			Convey("The storage should report error for any incoming query", func() {
				var (
					req = buildRequest(wt.WriteQuery, []wt.Query{
						buildQuery(`CREATE TABLE "t1" ("k" INT, "v" TEXT, PRIMARY KEY("k"))`),
					})
				)
				_, err = st.Query(req)
				So(err, ShouldNotBeNil)
				err = errors.Cause(err)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual,
					"sql: Transaction has already been committed or rolled back")
			})
		})
		Convey("The state will report error on read with uncommitted schema change", func() {
			// TODO(leventeliu): this case is not quite user friendly.
			var (
				req = buildRequest(wt.WriteQuery, []wt.Query{
					buildQuery(`CREATE TABLE "t1" ("k" INT, "v" TEXT, PRIMARY KEY("k"))`),
				})
				resp *wt.Response
			)
			resp, err = st.Query(req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			resp, err = st.Query(buildRequest(wt.ReadQuery, []wt.Query{
				buildQuery(`SELECT * FROM "t1"`),
			}))
			So(err, ShouldNotBeNil)
			err = errors.Cause(err)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "database schema is locked: main")
		})
		Convey("When a basic KV table is created", func() {
			var (
				req = buildRequest(wt.WriteQuery, []wt.Query{
					buildQuery(`CREATE TABLE "t1" ("k" INT, "v" TEXT, PRIMARY KEY("k"))`),
				})
				resp *wt.Response
			)
			resp, err = st.Query(req)
			So(err, ShouldBeNil)
			So(resp, ShouldNotBeNil)
			err = st.commit()
			So(err, ShouldBeNil)
			Convey("The state should not change after attempted writing in read query", func() {
				resp, err = st.Query(buildRequest(wt.ReadQuery, []wt.Query{
					buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 1, "v1"),
					buildQuery(`SELECT "v" FROM "t1" WHERE "k"=?`, 1),
				}))
				// The use of Query instead of Exec won't produce an "attempt to write" error
				// like Exec, but it should still keep it readonly -- which means writes will
				// be ignored in this case.
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 0)
			})
			Convey("The state should work properly with reading/writing queries", func() {
				resp, err = st.Query(buildRequest(wt.WriteQuery, []wt.Query{
					buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 1, "v1"),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 0)
				resp, err = st.Query(buildRequest(wt.ReadQuery, []wt.Query{
					buildQuery(`SELECT "v" FROM "t1" WHERE "k"=?`, 1),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 1)
				So(resp.Payload.Columns, ShouldResemble, []string{"v"})
				So(resp.Payload.DeclTypes, ShouldResemble, []string{"TEXT"})
				So(resp.Payload.Rows[0].Values[0], ShouldResemble, []byte("v1"))

				resp, err = st.Query(buildRequest(wt.WriteQuery, []wt.Query{
					buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 2, "v2"),
					buildQuery(`INSERT INTO "t1" ("k", "v") VALUES (?, ?);
INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 3, "v3", 4, "v4"),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 0)
				resp, err = st.Query(buildRequest(wt.ReadQuery, []wt.Query{
					buildQuery(`SELECT "v" FROM "t1"`),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 4)
				So(resp.Payload.Columns, ShouldResemble, []string{"v"})
				So(resp.Payload.DeclTypes, ShouldResemble, []string{"TEXT"})
				So(resp.Payload.Rows[0].Values[0], ShouldResemble, []byte("v1"))
				So(resp.Payload.Rows[1].Values[0], ShouldResemble, []byte("v2"))
				So(resp.Payload.Rows[2].Values[0], ShouldResemble, []byte("v3"))
				So(resp.Payload.Rows[3].Values[0], ShouldResemble, []byte("v4"))

				resp, err = st.Query(buildRequest(wt.ReadQuery, []wt.Query{
					buildQuery(`SELECT * FROM "t1"`),
				}))
				So(err, ShouldBeNil)
				So(resp.Header.RowCount, ShouldEqual, 4)
				So(resp.Payload.Columns, ShouldResemble, []string{"k", "v"})
				So(resp.Payload.DeclTypes, ShouldResemble, []string{"INT", "TEXT"})
				So(resp.Payload.Rows[0].Values[0], ShouldResemble, int64(1))
				So(resp.Payload.Rows[0].Values[1], ShouldResemble, []byte("v1"))
				So(resp.Payload.Rows[1].Values[0], ShouldResemble, int64(2))
				So(resp.Payload.Rows[1].Values[1], ShouldResemble, []byte("v2"))
				So(resp.Payload.Rows[2].Values[0], ShouldResemble, int64(3))
				So(resp.Payload.Rows[2].Values[1], ShouldResemble, []byte("v3"))
				So(resp.Payload.Rows[3].Values[0], ShouldResemble, int64(4))
				So(resp.Payload.Rows[3].Values[1], ShouldResemble, []byte("v4"))
			})
		})
	})
}
