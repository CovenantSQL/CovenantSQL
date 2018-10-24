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

package sqlite

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
	. "github.com/smartystreets/goconvey/convey"
)

func TestStorage(t *testing.T) {
	Convey("Given a sqlite storage implementation", t, func() {
		const passes = 1000
		var (
			fl  = path.Join(testDataDir, t.Name())
			st  xi.Storage
			err error
		)
		st, err = NewSqlite(fmt.Sprint("file:", fl))
		So(err, ShouldBeNil)
		So(st, ShouldNotBeNil)
		Reset(func() {
			// Clean database file after each pass
			err = st.Close()
			So(err, ShouldBeNil)
			err = os.Remove(fl)
			So(err, ShouldBeNil)
			err = os.Remove(fmt.Sprint(fl, "-shm"))
			So(err == nil || os.IsNotExist(err), ShouldBeTrue)
			err = os.Remove(fmt.Sprint(fl, "-wal"))
			So(err == nil || os.IsNotExist(err), ShouldBeTrue)
		})
		Convey("When a basic KV table is created", func(c C) {
			// Create basic table for testing
			_, err = st.Writer().Exec(`CREATE TABLE "t1" ("k" INT, "v" TEXT, PRIMARY KEY("k"))`)
			So(err, ShouldBeNil)
			Convey("When storage is closed", func() {
				err = st.Close()
				So(err, ShouldBeNil)
				Convey("The storage should report error for any incoming query", func() {
					err = st.DirtyReader().QueryRow(`SELECT "v" FROM "t1" WHERE "k"=?`, 1).Scan(nil)
					So(err.Error(), ShouldEqual, "sql: database is closed")
					err = st.Reader().QueryRow(`SELECT "v" FROM "t1" WHERE "k"=?`, 1).Scan(nil)
					So(err.Error(), ShouldEqual, "sql: database is closed")
					_, err = st.Writer().Exec(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 1, "v1")
					So(err.Error(), ShouldEqual, "sql: database is closed")
				})
			})
			Convey("The storage should report error when readers attempt to write", func() {
				_, err = st.DirtyReader().Exec(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 1, "v1")
				So(err.Error(), ShouldEqual, "attempt to write a readonly database")
				_, err = st.Reader().Exec(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 1, "v1")
				So(err.Error(), ShouldEqual, "attempt to write a readonly database")
			})
			Convey("The storage should work properly under concurrent reading/writing", func(c C) {
				var (
					ec = make(chan error, passes)
					sc = make(chan struct{})
					wg = &sync.WaitGroup{}

					abortReaders = func() {
						close(sc)
					}
				)
				for i := 0; i < passes; i++ {
					wg.Add(1)
					go func(k int) {
						var ticker = time.NewTicker(1 * time.Millisecond)
						defer func() {
							ticker.Stop()
							wg.Done()
						}()
						for {
							select {
							case <-ticker.C:
								var (
									err error
									v   string
								)
								if err = st.Reader().QueryRow(
									`SELECT "v" FROM "t1" WHERE "k"=?`, k,
								).Scan(&v); err != sql.ErrNoRows {
									if err != nil {
										ec <- err
									} else {
										c.Printf("\n        Read pair from t1: k=%d v=%s ", k, v)
									}
									return
								}
							case <-sc:
								return
							}
						}
					}(i)
				}
				defer func() {
					wg.Wait()
					close(ec)
					var errs = len(ec)
					for err = range ec {
						Printf("\n        Get error from channel: %v ", err)
					}
					So(errs, ShouldBeZeroValue)
				}()
				for i := 0; i < passes; i++ {
					if _, err = st.Writer().Exec(
						`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, i, fmt.Sprintf("v%d", i),
					); err != nil {
						abortReaders()
					}
					So(err, ShouldBeNil)
					c.Printf("\n        Write pair to t1: k=%d v=v%d ", i, i)
				}
			})
			Convey("The storage should see uncommitted changes from dirty reader", func(c C) {
				var (
					tx *sql.Tx
					ec = make(chan error, passes)
					sc = make(chan struct{})
					wg = &sync.WaitGroup{}

					abortReaders = func() {
						close(sc)
					}
				)
				// Open transaction
				tx, err = st.Writer().Begin()
				So(err, ShouldBeNil)
				So(tx, ShouldNotBeNil)
				for i := 0; i < passes; i++ {
					wg.Add(1)
					go func(k int) {
						var ticker = time.NewTicker(1 * time.Millisecond)
						defer func() {
							ticker.Stop()
							wg.Done()
						}()
						for {
							select {
							case <-ticker.C:
								var (
									err error
									v   string
								)
								if err = st.DirtyReader().QueryRow(
									`SELECT "v" FROM "t1" WHERE "k"=?`, k,
								).Scan(&v); err != sql.ErrNoRows {
									if err != nil {
										ec <- err
									} else {
										c.Printf("\n        Dirty read pair from t1: k=%d v=%s ",
											k, v)
									}
									return
								}
							case <-sc:
								return
							}
						}
					}(i)
				}
				defer func() {
					wg.Wait()
					close(ec)
					var errs = len(ec)
					for err = range ec {
						Printf("\n        Get error from channel: %v ", err)
					}
					So(errs, ShouldBeZeroValue)
					err = tx.Commit()
					So(err, ShouldBeNil)
				}()
				for i := 0; i < passes; i++ {
					var (
						v  = fmt.Sprintf("v%d", i)
						rv string
					)
					if _, err = tx.Exec(
						`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, i, v,
					); err != nil {
						abortReaders()
					}
					So(err, ShouldBeNil)
					// No isolation between operations on the same database connection
					if err = tx.QueryRow(
						`SELECT "v" FROM "t1" WHERE "k"=?`, i,
					).Scan(&rv); err != nil || rv != v {
						abortReaders()
					}
					So(err, ShouldBeNil)
					So(rv, ShouldEqual, v)
					c.Printf("\n        Write pair to t1 in transaction: k=%d v=%s ", i, v)
				}
				// Reader connection should not see any uncommitted change
				for i := 0; i < passes; i++ {
					err = st.Reader().QueryRow(`SELECT "v" FROM "t1" WHERE "k"=?`, i).Scan(nil)
					So(err, ShouldEqual, sql.ErrNoRows)
				}
			})
		})
	})
}

func setupBenchmarkStorage(
	b *testing.B,
) (
	st xi.Storage, n int,
	q string, makeDest func() []interface{},
	e string, src [][]interface{},
) {
	const (
		cols    = 3
		vlen    = 100
		records = 1000
	)
	// Setup storage
	var (
		fl   = path.Join(testDataDir, b.Name())
		err  error
		stmt *sql.Stmt
	)
	if st, err = NewSqlite(fmt.Sprint("file:", fl)); err != nil {
		b.Fatalf("Failed to setup bench environment: %v", err)
	}
	if _, err = st.Writer().Exec(
		`CREATE TABLE "t2" ("k" INT, "v1" TEXT, "v2" TEXT, "v3" TEXT, PRIMARY KEY("k"))`,
	); err != nil {
		b.Fatalf("Failed to setup bench environment: %v", err)
	}
	if stmt, err = st.Writer().Prepare(
		`INSERT INTO "t2" VALUES (?, ?, ?, ?)`,
	); err != nil {
		b.Fatalf("Failed to setup bench environment: %v", err)
	}
	for i := 0; i < records; i++ {
		var (
			vals [cols][vlen]byte
			args [cols + 1]interface{}
		)
		args[0] = i
		for i := range vals {
			rand.Read(vals[i][:])
			args[i+1] = string(vals[i][:])
		}
		if _, err = stmt.Exec(args[:]...); err != nil {
			b.Fatalf("Failed to setup bench environment: %v", err)
		}
	}
	n = records
	// Setup query string and dest slice
	q = `SELECT "v1", "v2", "v3" FROM "t2" WHERE "k"=?`
	makeDest = func() (dest []interface{}) {
		var outv [cols]string
		dest = make([]interface{}, cols)
		for i := range outv {
			dest[i] = &outv[i]
		}
		return
	}
	// Setup execute string and src table
	//
	// NOTE(leventeliu): allowing IGNORE and REPLACE both have impact on benchmark result,
	// while UPSERT is the best!
	//
	// e = `INSERT OR IGNORE INTO "t2" VALUES (?, ?, ?, ?)`
	// e = `REPLACE INTO "t2" VALUES (?, ?, ?, ?)`
	e = `INSERT INTO "t2" VALUES (?, ?, ?, ?)
	ON CONFLICT("k") DO UPDATE SET
		"v1"="excluded"."v1",
		"v2"="excluded"."v2",
		"v3"="excluded"."v3"
`
	src = make([][]interface{}, records)
	for i := range src {
		var vals [cols][vlen]byte
		src[i] = make([]interface{}, cols+1)
		src[i][0] = i + records
		for j := range vals {
			rand.Read(vals[j][:])
			src[i][j+1] = string(vals[j][:])
		}
	}
	return
}

func teardownBenchmarkStorage(b *testing.B, st xi.Storage) {
	var (
		fl  = path.Join(testDataDir, b.Name())
		err error
	)
	if err = st.Close(); err != nil {
		b.Fatalf("Failed to teardown bench environment: %v", err)
	}
	if err = os.Remove(fl); err != nil {
		b.Fatalf("Failed to teardown bench environment: %v", err)
	}
	if err = os.Remove(fmt.Sprint(fl, "-shm")); err != nil && !os.IsNotExist(err) {
		b.Fatalf("Failed to teardown bench environment: %v", err)
	}
	if err = os.Remove(fmt.Sprint(fl, "-wal")); err != nil && !os.IsNotExist(err) {
		b.Fatalf("Failed to teardown bench environment: %v", err)
	}
}

func BenchmarkStorageSequentialDirtyRead(b *testing.B) {
	var (
		st, n, q, dm, _, _ = setupBenchmarkStorage(b)

		dest = dm()
		err  error
	)
	for i := 0; i < b.N; i++ {
		if err = st.DirtyReader().QueryRow(q, rand.Intn(n)).Scan(dest...); err != nil {
			b.Fatalf("Failed to query values: %v", err)
		}
	}
	teardownBenchmarkStorage(b, st)
}

func BenchmarkStoargeSequentialRead(b *testing.B) {
	var (
		st, n, q, dm, _, _ = setupBenchmarkStorage(b)

		dest = dm()
		err  error
	)
	for i := 0; i < b.N; i++ {
		if err = st.Reader().QueryRow(q, rand.Intn(n)).Scan(dest...); err != nil {
			b.Fatalf("Failed to query values: %v", err)
		}
	}
	teardownBenchmarkStorage(b, st)
}

// BW is a background writer function passed to benchmark helper.
type BW func(
	*testing.B, *sync.WaitGroup, <-chan struct{}, xi.Storage, int, string, [][]interface{},
)

func backgroundWriter(
	b *testing.B,
	wg *sync.WaitGroup, sc <-chan struct{},
	st xi.Storage, n int, e string, src [][]interface{},
) {
	defer wg.Done()
	var err error
	for {
		select {
		case <-sc:
			return
		default:
			if _, err = st.Writer().Exec(e, src[rand.Intn(n)]...); err != nil {
				b.Errorf("Failed to execute: %v", err)
			}
		}
	}
}

func backgroundBusyTxWriter(
	b *testing.B,
	wg *sync.WaitGroup, sc <-chan struct{},
	st xi.Storage, n int, e string, src [][]interface{},
) {
	defer wg.Done()
	const cmtNum = 100
	var (
		tx  *sql.Tx
		err error
	)
	for i := 0; ; i++ {
		// Begin
		if i%cmtNum == 0 {
			if tx, err = st.Writer().Begin(); err != nil {
				b.Errorf("Failed to begin transaction: %v", err)
			}
		}
		// Exec
		select {
		case <-sc:
			// Also commit on exiting
			if tx != nil {
				if err = tx.Commit(); err != nil {
					b.Errorf("Failed to commit transaction: %v", err)
				}
				tx = nil
			}
			return
		default:
			// Exec
			if _, err = tx.Exec(e, src[rand.Intn(n)]...); err != nil {
				b.Errorf("Failed to execute: %v", err)
			}
		}
		// Commit
		if (i+1)%cmtNum == 0 {
			if err = tx.Commit(); err != nil {
				b.Errorf("Failed to commit transaction: %v", err)
			}
			tx = nil
		}
	}
}

func backgroundIdleTxWriter(
	b *testing.B,
	wg *sync.WaitGroup, sc <-chan struct{},
	st xi.Storage, n int, e string, src [][]interface{},
) {
	const (
		cmtNum      = 100
		writeIntlMS = 1
	)
	var (
		tx     *sql.Tx
		err    error
		ticker = time.NewTicker(writeIntlMS * time.Millisecond)
	)
	defer func() {
		ticker.Stop()
		wg.Done()
	}()
	for i := 0; ; i++ {
		// Begin
		if i%cmtNum == 0 {
			if tx, err = st.Writer().Begin(); err != nil {
				b.Errorf("Failed to begin transaction: %v", err)
			}
		}
		// Exec
		select {
		case <-ticker.C:
			// Exec
			if _, err = tx.Exec(e, src[rand.Intn(n)]...); err != nil {
				b.Errorf("Failed to execute: %v", err)
			}
		case <-sc:
			// Also commit on exiting
			if tx != nil {
				if err = tx.Commit(); err != nil {
					b.Errorf("Failed to commit transaction: %v", err)
				}
				tx = nil
			}
			return
		}
		// Commit
		if (i+1)%cmtNum == 0 {
			if err = tx.Commit(); err != nil {
				b.Errorf("Failed to commit transaction: %v", err)
			}
			tx = nil
		}
	}
}

// GR is a get reader function passed to benchmark helper.
type GR func(xi.Storage) *sql.DB

func getDirtyReader(st xi.Storage) *sql.DB { return st.DirtyReader() }
func getReader(st xi.Storage) *sql.DB      { return st.Reader() }

func benchmarkStorageSequentialReadWithBackgroundWriter(b *testing.B, getReader GR, write BW) {
	var (
		st, n, q, dm, e, src = setupBenchmarkStorage(b)

		dest = dm()
		wg   = &sync.WaitGroup{}
		sc   = make(chan struct{})

		err error
	)

	// Start background writer
	wg.Add(1)
	go write(b, wg, sc, st, n, e, src)

	for i := 0; i < b.N; i++ {
		if err = getReader(st).QueryRow(q, rand.Intn(n)).Scan(dest...); err != nil {
			b.Fatalf("Failed to query values: %v", err)
		}
	}

	// Exit background writer
	close(sc)
	wg.Wait()

	teardownBenchmarkStorage(b, st)
}

func BenchmarkStoargeSequentialDirtyReadWithBackgroundWriter(b *testing.B) {
	benchmarkStorageSequentialReadWithBackgroundWriter(b, getDirtyReader, backgroundWriter)
}

func BenchmarkStoargeSequentialReadWithBackgroundWriter(b *testing.B) {
	benchmarkStorageSequentialReadWithBackgroundWriter(b, getReader, backgroundWriter)
}

func BenchmarkStoargeSequentialDirtyReadWithBackgroundBusyTxWriter(b *testing.B) {
	benchmarkStorageSequentialReadWithBackgroundWriter(b, getDirtyReader, backgroundBusyTxWriter)
}

func BenchmarkStoargeSequentialReadWithBackgroundBusyTxWriter(b *testing.B) {
	benchmarkStorageSequentialReadWithBackgroundWriter(b, getReader, backgroundBusyTxWriter)
}

func BenchmarkStoargeSequentialDirtyReadWithBackgroundIdleTxWriter(b *testing.B) {
	benchmarkStorageSequentialReadWithBackgroundWriter(b, getDirtyReader, backgroundIdleTxWriter)
}

func BenchmarkStoargeSequentialReadWithBackgroundIdleTxWriter(b *testing.B) {
	benchmarkStorageSequentialReadWithBackgroundWriter(b, getReader, backgroundIdleTxWriter)
}

func BenchmarkStoargeSequentialMixDRW(b *testing.B) {
	var (
		st, n, q, dm, e, src = setupBenchmarkStorage(b)

		dest = dm()
		err  error
	)
	for i := 0; i < b.N; i++ {
		if rand.Int()%2 == 0 {
			if err = st.DirtyReader().QueryRow(q, rand.Intn(n)).Scan(dest...); err != nil {
				b.Fatalf("Failed to query values: %v", err)
			}
		} else {
			if _, err = st.Writer().Exec(e, src[rand.Intn(n)]...); err != nil {
				b.Fatalf("Failed to execute: %v", err)
			}
		}
	}
	teardownBenchmarkStorage(b, st)
}

func BenchmarkStoargeSequentialMixRW(b *testing.B) {
	var (
		st, n, q, dm, e, src = setupBenchmarkStorage(b)

		dest = dm()
		err  error
	)
	for i := 0; i < b.N; i++ {
		if rand.Int()%2 == 0 {
			if err = st.Reader().QueryRow(q, rand.Intn(n)).Scan(dest...); err != nil {
				b.Fatalf("Failed to query values: %v", err)
			}
		} else {
			if _, err = st.Writer().Exec(e, src[rand.Intn(n)]...); err != nil {
				b.Fatalf("Failed to execute: %v", err)
			}
		}
	}
	teardownBenchmarkStorage(b, st)
}

func BenchmarkStorageParallelDirtyRead(b *testing.B) {
	var st, n, q, dm, _, _ = setupBenchmarkStorage(b)
	b.RunParallel(func(pb *testing.PB) {
		var (
			dest = dm()
			err  error
		)
		for pb.Next() {
			if err = st.DirtyReader().QueryRow(q, rand.Intn(n)).Scan(dest...); err != nil {
				b.Fatalf("Failed to query values: %v", err)
			}
		}
	})
	teardownBenchmarkStorage(b, st)
}

func BenchmarkStorageParallelRead(b *testing.B) {
	var st, n, q, dm, _, _ = setupBenchmarkStorage(b)
	b.RunParallel(func(pb *testing.PB) {
		var (
			dest = dm()
			err  error
		)
		for pb.Next() {
			if err = st.DirtyReader().QueryRow(q, rand.Intn(n)).Scan(dest...); err != nil {
				b.Fatalf("Failed to query values: %v", err)
			}
		}
	})
	teardownBenchmarkStorage(b, st)
}

func BenchmarkStoargeParallelWrite(b *testing.B) {
	var st, n, _, _, e, src = setupBenchmarkStorage(b)
	b.RunParallel(func(pb *testing.PB) {
		var err error
		for pb.Next() {
			if _, err = st.Writer().Exec(e, src[rand.Intn(n)]...); err != nil {
				b.Fatalf("Failed to execute: %v", err)
			}
		}
	})
	teardownBenchmarkStorage(b, st)
}

func BenchmarkStorageParallelMixDRW(b *testing.B) {
	var st, n, q, dm, e, src = setupBenchmarkStorage(b)
	b.RunParallel(func(pb *testing.PB) {
		var (
			dest = dm()
			err  error
		)
		for pb.Next() {
			if rand.Int()%2 == 0 {
				if err = st.DirtyReader().QueryRow(q, rand.Intn(n)).Scan(dest...); err != nil {
					b.Fatalf("Failed to query values: %v", err)
				}
			} else {
				if _, err = st.Writer().Exec(e, src[rand.Intn(n)]...); err != nil {
					b.Fatalf("Failed to execute: %v", err)
				}
			}
		}
	})
	teardownBenchmarkStorage(b, st)
}

func BenchmarkStorageParallelMixRW(b *testing.B) {
	var st, n, q, dm, e, src = setupBenchmarkStorage(b)
	b.RunParallel(func(pb *testing.PB) {
		var (
			dest = dm()
			err  error
		)
		for pb.Next() {
			if rand.Int()%2 == 0 {
				if err = st.Reader().QueryRow(q, rand.Intn(n)).Scan(dest...); err != nil {
					b.Fatalf("Failed to query values: %v", err)
				}
			} else {
				if _, err = st.Writer().Exec(e, src[rand.Intn(n)]...); err != nil {
					b.Fatalf("Failed to execute: %v", err)
				}
			}
		}
	})
	teardownBenchmarkStorage(b, st)
}
