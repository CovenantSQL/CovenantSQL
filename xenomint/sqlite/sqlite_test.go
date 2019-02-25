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
	"sync/atomic"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	xi "github.com/CovenantSQL/CovenantSQL/xenomint/interfaces"
)

func TestStorage(t *testing.T) {
	Convey("Given a sqlite storage implementation", t, func() {
		const passes = 1000
		var (
			fl  = path.Join(testingDataDir, t.Name())
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
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "sql: database is closed")
					err = st.Reader().QueryRow(`SELECT "v" FROM "t1" WHERE "k"=?`, 1).Scan(nil)
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "sql: database is closed")
					_, err = st.Writer().Exec(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 1, "v1")
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldEqual, "sql: database is closed")
				})
			})
			Convey("The storage should report error when readers attempt to write", func() {
				_, err = st.DirtyReader().Exec(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 1, "v1")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "attempt to write a readonly database")
				_, err = st.Reader().Exec(`INSERT INTO "t1" ("k", "v") VALUES (?, ?)`, 1, "v1")
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "attempt to write a readonly database")
			})
			Convey("The storage should work properly under concurrent reading/writing", func(c C) {
				var (
					ec = make(chan error, passes)
					sc = make(chan struct{})
					wg = &sync.WaitGroup{}

					abortReaders = func() { close(sc) }
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

					abortReaders = func() { close(sc) }
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

const (
	benchmarkQueriesPerTx      = 100
	benchmarkVNum              = 3
	benchmarkVLen              = 333
	benchmarkKeySubspaceLength = 1000000

	benchmarkReservedKeyOffset = iota * benchmarkKeySubspaceLength
	benchmarkNewKeyOffset
	benchmarkKeySpace
)

type keygen interface {
	next() int
	reset()
}

type randKeygen struct {
	offset int
	length int
}

func newRandKeygen(offset, length int) *randKeygen {
	return &randKeygen{
		offset: offset,
		length: length,
	}
}

func newIndexRandKeygen(length int) *randKeygen { return newRandKeygen(0, length) }

func (k *randKeygen) next() int { return rand.Intn(k.length) + k.offset }
func (k *randKeygen) reset()    {}

type permKeygen struct {
	offset int
	length int
	perm   []int
	pos    int32
}

func newPermKeygen(offset, length int) *permKeygen {
	return &permKeygen{
		offset: offset,
		length: length,
		perm:   rand.Perm(length),
	}
}

func newIndexPermKeygen(length int) *permKeygen { return newPermKeygen(0, length) }

func (k *permKeygen) next() int {
	var pos = atomic.AddInt32(&k.pos, 1) - 1
	if pos >= int32(k.length) {
		panic("permKeygen: keys have been exhausted")
	}
	return k.perm[pos] + k.offset
}

func (k *permKeygen) reset() { k.pos = 0 }

var (
	irkg = newIndexRandKeygen(benchmarkKeySubspaceLength)
	ipkg = newIndexPermKeygen(benchmarkKeySubspaceLength)
	rrkg = newRandKeygen(benchmarkReservedKeyOffset, benchmarkKeySubspaceLength)
	nrkg = newRandKeygen(benchmarkNewKeyOffset, benchmarkKeySubspaceLength)
	trkg = newRandKeygen(0, benchmarkKeySpace)
)

func setupBenchmarkStorage(b *testing.B) (
	st xi.Storage,
	q string, makeDest func() []interface{},
	e string, src [][]interface{},
) {
	// Setup storage
	var (
		fl   = path.Join(testingDataDir, b.Name())
		err  error
		stmt *sql.Stmt
	)
	if st, err = NewSqlite(fmt.Sprint("file:", fl)); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	if _, err = st.Writer().Exec(
		`CREATE TABLE "t2" ("k" INT, "v1" TEXT, "v2" TEXT, "v3" TEXT, PRIMARY KEY("k"))`,
	); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	if stmt, err = st.Writer().Prepare(
		`INSERT INTO "t2" VALUES (?, ?, ?, ?)`,
	); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	for i := 0; i < benchmarkKeySubspaceLength; i++ {
		var (
			vals [benchmarkVNum][benchmarkVLen]byte
			args [benchmarkVNum + 1]interface{}
		)
		args[0] = benchmarkReservedKeyOffset + i
		for i := range vals {
			rand.Read(vals[i][:])
			args[i+1] = string(vals[i][:])
		}
		if _, err = stmt.Exec(args[:]...); err != nil {
			b.Fatalf("failed to setup bench environment: %v", err)
		}
		if i%10000 == 0 {
			fmt.Printf("Done setup key at %v\n", i)
		}
	}
	// Setup query string and dest slice
	q = `SELECT "v1", "v2", "v3" FROM "t2" WHERE "k"=?`
	makeDest = func() (dest []interface{}) {
		var outv [benchmarkVNum]string
		dest = make([]interface{}, benchmarkVNum)
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
	src = make([][]interface{}, benchmarkKeySubspaceLength)
	for i := range src {
		var vals [benchmarkVNum][benchmarkVLen]byte
		src[i] = make([]interface{}, benchmarkVNum+1)
		src[i][0] = benchmarkNewKeyOffset + i
		for j := range vals {
			rand.Read(vals[j][:])
			src[i][j+1] = string(vals[j][:])
		}
	}

	return
}

func teardownBenchmarkStorage(b *testing.B, st xi.Storage) {
	var (
		fl  = path.Join(testingDataDir, b.Name())
		err error
	)
	if err = st.Close(); err != nil {
		b.Fatalf("failed to teardown bench environment: %v", err)
	}
	if err = os.Remove(fl); err != nil {
		b.Fatalf("failed to teardown bench environment: %v", err)
	}
	if err = os.Remove(fmt.Sprint(fl, "-shm")); err != nil && !os.IsNotExist(err) {
		b.Fatalf("failed to teardown bench environment: %v", err)
	}
	if err = os.Remove(fmt.Sprint(fl, "-wal")); err != nil && !os.IsNotExist(err) {
		b.Fatalf("failed to teardown bench environment: %v", err)
	}
}

func setupSubBenchmarkStorage(b *testing.B, st xi.Storage) {
	// Reset key generators
	irkg.reset()
	ipkg.reset()
	rrkg.reset()
	nrkg.reset()
	trkg.reset()
}

func teardownSubBenchmarkStorage(b *testing.B, st xi.Storage) {
	var (
		d   = `DELETE FROM "t2" WHERE "k">=?`
		err error
	)
	if _, err = st.Writer().Exec(d, benchmarkNewKeyOffset); err != nil {
		b.Fatalf("failed to teardown sub bench environment: %v", err)
	}
}

type benchmarkProfile struct {
	name   string
	parall bool
	proc   func(*testing.B, int)
	pproc  func(*testing.PB)
	bg     func(*testing.B, *sync.WaitGroup, <-chan struct{})
}

func BenchmarkStorage(b *testing.B) {
	var (
		st, q, dm, e, src = setupBenchmarkStorage(b)

		tx   *sql.Tx
		dest = dm()
		read = func(b *testing.B, conn *sql.DB, dest []interface{}) {
			var err error
			if err = conn.QueryRow(q, rrkg.next()).Scan(dest...); err != nil {
				b.Fatalf("failed to query values: %v", err)
			}
		}
		readTx = func(b *testing.B, i int, conn *sql.DB, dest []interface{}) {
			var err error
			if i%benchmarkQueriesPerTx == 0 {
				if tx, err = conn.Begin(); err != nil {
					b.Fatalf("failed to begin transaction: %v", err)
				}
			}
			// Query in [n, 2n-1] key space
			if err = tx.QueryRow(q, nrkg.next()).Scan(dest...); err != nil && err != sql.ErrNoRows {
				b.Fatalf("failed to query values: %v", err)
			}
			if (i+1)%benchmarkQueriesPerTx == 0 || i == b.N-1 {
				if err = tx.Rollback(); err != nil {
					b.Fatalf("failed to close transaction: %v", err)
				}
			}
		}
		write = func(b *testing.B, conn *sql.DB) {
			var err error
			if _, err = conn.Exec(e, src[ipkg.next()]...); err != nil {
				b.Errorf("failed to execute: %v", err)
			}
		}
		writeTx = func(b *testing.B, i int, conn *sql.DB) {
			var err error
			if i%benchmarkQueriesPerTx == 0 {
				if tx, err = st.Writer().Begin(); err != nil {
					b.Errorf("failed to begin transaction: %v", err)
				}
			}
			if _, err = tx.Exec(e, src[ipkg.next()]...); err != nil {
				b.Errorf("failed to execute: %v", err)
			}
			if (i+1)%benchmarkQueriesPerTx == 0 || i == b.N-1 {
				if err = tx.Commit(); err != nil {
					b.Errorf("failed to commit transaction: %v", err)
				}
			}
		}
		mixRW = func(b *testing.B, rconn, wconn *sql.DB, dest []interface{}) {
			if rand.Int()%2 == 0 {
				read(b, rconn, dest)
			} else {
				write(b, wconn)
			}
		}

		bgw = func(b *testing.B, wg *sync.WaitGroup, sc <-chan struct{}) {
			busyWrite(b, wg, sc, st, ipkg, e, src)
		}
		bgbwtx = func(b *testing.B, wg *sync.WaitGroup, sc <-chan struct{}) {
			busyWriteTx(b, wg, sc, st, ipkg, e, src)
		}
		bgiwtx = func(b *testing.B, wg *sync.WaitGroup, sc <-chan struct{}) {
			idleWriteTx(b, wg, sc, st, ipkg, e, src)
		}

		pproc = func(pb *testing.PB, proc func()) {
			for pb.Next() {
				proc()
			}
		}

		profiles = [...]benchmarkProfile{
			{
				name: "SequentialDirtyRead",
				proc: func(b *testing.B, _ int) { read(b, st.DirtyReader(), dest) },
			}, {
				name: "SequentialRead",
				proc: func(b *testing.B, _ int) { read(b, st.Reader(), dest) },
			}, {
				name: "SequentialWrite",
				proc: func(b *testing.B, _ int) { write(b, st.Writer()) },
			}, {
				name: "SequentialWriteTx",
				proc: func(b *testing.B, i int) { writeTx(b, i, st.Writer()) },
			}, {
				name: "SequentialMixDRW",
				proc: func(b *testing.B, _ int) { mixRW(b, st.DirtyReader(), st.Writer(), dest) },
			}, {
				name: "SequentialMixRW",
				proc: func(b *testing.B, _ int) { mixRW(b, st.Reader(), st.Writer(), dest) },
			}, {
				name: "SequentialDirtyReadWithBackgroundWriter",
				proc: func(b *testing.B, _ int) { read(b, st.DirtyReader(), dest) },
				bg:   bgw,
			}, {
				name: "SequentialReadWithBackgroundWriter",
				proc: func(b *testing.B, _ int) { read(b, st.Reader(), dest) },
				bg:   bgw,
			}, {
				name: "SequentialDirtyReadWithBackgroundBusyTxWriter",
				proc: func(b *testing.B, _ int) { read(b, st.DirtyReader(), dest) },
				bg:   bgbwtx,
			}, {
				name: "SequentialReadWithBackgroundBusyTxWriter",
				proc: func(b *testing.B, _ int) { read(b, st.Reader(), dest) },
				bg:   bgbwtx,
			}, {
				name: "SequentialDirtyReadWithBackgroundIdleTxWriter",
				proc: func(b *testing.B, _ int) { read(b, st.DirtyReader(), dest) },
				bg:   bgiwtx,
			}, {
				name: "SequentialReadWithBackgroundIdleTxWriter",
				proc: func(b *testing.B, _ int) { read(b, st.Reader(), dest) },
				bg:   bgiwtx,
			}, {
				name: "SequentialDirtyReadTxWithBackgroundWriter",
				proc: func(b *testing.B, i int) { readTx(b, i, st.DirtyReader(), dest) },
				bg:   bgw,
			}, {
				name: "SequentialReadTxWithBackgroundWriter",
				proc: func(b *testing.B, i int) { readTx(b, i, st.Reader(), dest) },
				bg:   bgw,
			}, {
				name: "SequentialDirtyReadTxWithBackgroundBusyTxWriter",
				proc: func(b *testing.B, i int) { readTx(b, i, st.DirtyReader(), dest) },
				bg:   bgbwtx,
			}, {
				name: "SequentialReadTxWithBackgroundBusyTxWriter",
				proc: func(b *testing.B, i int) { readTx(b, i, st.Reader(), dest) },
				bg:   bgbwtx,
			}, {
				name: "SequentialDirtyReadTxWithBackgroundIdleTxWriter",
				proc: func(b *testing.B, i int) { readTx(b, i, st.DirtyReader(), dest) },
				bg:   bgiwtx,
			}, {
				name: "SequentialReadTxWithBackgroundIdleTxWriter",
				proc: func(b *testing.B, i int) { readTx(b, i, st.Reader(), dest) },
				bg:   bgiwtx,
			}, {
				name:   "ParallelDirtyRead",
				parall: true,
				pproc: func(pb *testing.PB) {
					pproc(pb, func() { read(b, st.DirtyReader(), dm()) })
				},
			}, {
				name:   "ParallelRead",
				parall: true,
				pproc: func(pb *testing.PB) {
					pproc(pb, func() { read(b, st.Reader(), dm()) })
				},
			}, {
				name:   "ParallelWrite",
				parall: true,
				pproc: func(pb *testing.PB) {
					pproc(pb, func() { write(b, st.Writer()) })
				},
			}, {
				name:   "ParallelMixDRW",
				parall: true,
				pproc: func(pb *testing.PB) {
					pproc(pb, func() { mixRW(b, st.DirtyReader(), st.Writer(), dm()) })
				},
			}, {
				name:   "ParallelMixRW",
				parall: true,
				pproc: func(pb *testing.PB) {
					pproc(pb, func() { mixRW(b, st.Reader(), st.Writer(), dm()) })
				},
			},
		}
	)
	defer teardownBenchmarkStorage(b, st)
	// Run benchmark profiles
	for _, v := range profiles {
		b.Run(v.name, func(b *testing.B) {
			// Setup environment for sub-benchmark
			setupSubBenchmarkStorage(b, st)
			defer teardownSubBenchmarkStorage(b, st)
			// Start background goroutine
			var (
				wg = &sync.WaitGroup{}
				sc = make(chan struct{})
			)
			if v.bg != nil {
				wg.Add(1)
				go v.bg(b, wg, sc)
			}
			defer func() {
				close(sc)
				wg.Wait()
			}()
			// Test body
			b.ResetTimer()
			if v.parall {
				// Run parallel
				b.RunParallel(v.pproc)
			} else {
				// Run sequential
				for i := 0; i < b.N; i++ {
					v.proc(b, i)
				}
			}
			b.StopTimer()
		})
	}
}

func setupBenchmarkLargeWriteTx(
	b *testing.B, st xi.Storage, e string, src [][]interface{}, n int) (tx *sql.Tx,
) {
	var err error
	ipkg.reset()
	if tx, err = st.Writer().Begin(); err != nil {
		b.Fatalf("Failed to setup bench environment: %v", err)
	}
	for i := 0; i < n; i++ {
		if _, err = tx.Exec(e, src[ipkg.next()]...); err != nil {
			b.Fatalf("Failed to setup bench environment: %v", err)
		}
	}
	if _, err = tx.Exec(`SAVEPOINT "xmark"`); err != nil {
		b.Fatalf("Failed to setup bench environment: %v", err)
	}
	return
}

func resetBenchmarkLargeWriteTx(b *testing.B, tx *sql.Tx) {
	var err error
	b.StopTimer()
	if _, err := tx.Exec(`ROLLBACK TO "xmark"`); err != nil {
		b.Fatalf("Failed to reset bench environment: %v", err)
	}
	if _, err = tx.Exec(`SAVEPOINT "xmark"`); err != nil {
		b.Fatalf("Failed to reset bench environment: %v", err)
	}
}

func teardownBenchmarkLargeWriteTx(b *testing.B, tx *sql.Tx) {
	if err := tx.Rollback(); err != nil {
		b.Fatalf("Failed to teardown bench environment: %v", err)
	}
}

func BenchmarkLargeWriteTx(b *testing.B) {
	var (
		st, _, _, e, src = setupBenchmarkStorage(b)

		profiles = [...]int{0, 10, 100, 1000, 10000, 100000}
		err      error
	)
	for _, v := range profiles {
		func() {
			var tx = setupBenchmarkLargeWriteTx(b, st, e, src, v)
			defer teardownBenchmarkLargeWriteTx(b, tx)

			b.Run(fmt.Sprintf("%s#%d", b.Name(), v), func(b *testing.B) {
				defer resetBenchmarkLargeWriteTx(b, tx)
				for i := 0; i < b.N; i++ {
					if _, err = tx.Exec(e, src[ipkg.next()]...); err != nil {
						b.Errorf("Failed to execute: %v", err)
					}
				}
			})
		}()
	}
	// A simple commit duration testing, but not benchmark
	for _, v := range profiles {
		var (
			tx    = setupBenchmarkLargeWriteTx(b, st, e, src, v)
			start = time.Now()
		)
		if err = tx.Commit(); err != nil {
			b.Errorf("Failed to commit: %v", err)
		}
		b.Logf("Commit %d writes in %.3fms", v, float64(time.Since(start).Nanoseconds())/1000000)
	}
	teardownBenchmarkStorage(b, st)
}

//func BenchmarkStorageSequentialDirtyRead(b *testing.B) {
//	var (
//		st, q, dm, _, _ = setupBenchmarkStorage(b)
//		dest            = dm()
//		err             error
//	)
//	for i := 0; i < b.N; i++ {
//		if err = st.DirtyReader().QueryRow(q, rrkg.next()).Scan(dest...); err != nil {
//			b.Fatalf("failed to query values: %v", err)
//		}
//	}
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStoargeSequentialRead(b *testing.B) {
//	var (
//		st, q, dm, _, _ = setupBenchmarkStorage(b)
//		dest            = dm()
//		err             error
//	)
//	for i := 0; i < b.N; i++ {
//		if err = st.Reader().QueryRow(q, rrkg.next()).Scan(dest...); err != nil {
//			b.Fatalf("failed to query values: %v", err)
//		}
//	}
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStoargeSequentialWrite(b *testing.B) {
//	var (
//		st, _, _, e, src = setupBenchmarkStorage(b)
//		err              error
//	)
//	b.Run("BenchmarkStoargeSequentialWrite", func(b *testing.B) {
//		deleteBenchmarkData(b, st)
//		b.ResetTimer()
//		for i := 0; i < b.N; i++ {
//			if _, err = st.Writer().Exec(e, src[ipkg.next()]...); err != nil {
//				b.Errorf("failed to execute: %v", err)
//			}
//		}
//	})
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStoargeSequentialWriteTx(b *testing.B) {
//	var (
//		st, _, _, e, src = setupBenchmarkStorage(b)
//		tx               *sql.Tx
//		err              error
//	)
//	for i := 0; i < b.N; i++ {
//		if i%benchmarkQueriesPerTx == 0 {
//			if tx, err = st.Writer().Begin(); err != nil {
//				b.Errorf("failed to begin transaction: %v", err)
//			}
//		}
//		if _, err = tx.Exec(e, src[ipkg.next()]...); err != nil {
//			b.Errorf("failed to execute: %v", err)
//		}
//		if (i+1)%benchmarkQueriesPerTx == 0 || i == b.N-1 {
//			if err = tx.Commit(); err != nil {
//				b.Errorf("failed to commit transaction: %v", err)
//			}
//		}
//	}
//	teardownBenchmarkStorage(b, st)
//}
//
// BW is a background writer function passed to benchmark helper.
//type BW func(
//	*testing.B, *sync.WaitGroup, <-chan struct{}, xi.Storage, keygen, string, [][]interface{},
//)

func busyWrite(
	b *testing.B,
	wg *sync.WaitGroup, sc <-chan struct{},
	st xi.Storage, kg keygen, e string, src [][]interface{},
) {
	defer wg.Done()
	var err error
	for {
		select {
		case <-sc:
			return
		default:
			if _, err = st.Writer().Exec(e, src[kg.next()]...); err != nil {
				b.Errorf("failed to execute: %v", err)
			}
		}
	}
}

func busyWriteTx(
	b *testing.B,
	wg *sync.WaitGroup, sc <-chan struct{},
	st xi.Storage, kg keygen, e string, src [][]interface{},
) {
	defer wg.Done()
	var (
		tx  *sql.Tx
		err error
	)
	for i := 0; ; i++ {
		// Begin
		if i%benchmarkQueriesPerTx == 0 {
			if tx, err = st.Writer().Begin(); err != nil {
				b.Errorf("failed to begin transaction: %v", err)
			}
		}
		// Exec
		select {
		case <-sc:
			// Also commit on exiting
			if tx != nil {
				if err = tx.Commit(); err != nil {
					b.Errorf("failed to commit transaction: %v", err)
				}
				tx = nil
			}
			return
		default:
			// Exec
			if _, err = tx.Exec(e, src[kg.next()]...); err != nil {
				b.Errorf("failed to execute: %v", err)
			}
		}
		// Commit
		if (i+1)%benchmarkQueriesPerTx == 0 {
			if err = tx.Commit(); err != nil {
				b.Errorf("failed to commit transaction: %v", err)
			}
			tx = nil
		}
	}
}

func idleWriteTx(
	b *testing.B,
	wg *sync.WaitGroup, sc <-chan struct{},
	st xi.Storage, kg keygen, e string, src [][]interface{},
) {
	const writeIntlMS = 1
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
		if i%benchmarkQueriesPerTx == 0 {
			if tx, err = st.Writer().Begin(); err != nil {
				b.Errorf("failed to begin transaction: %v", err)
			}
		}
		// Exec
		select {
		case <-ticker.C:
			// Exec
			if _, err = tx.Exec(e, src[kg.next()]...); err != nil {
				b.Errorf("failed to execute: %v", err)
			}
		case <-sc:
			// Also commit on exiting
			if tx != nil {
				if err = tx.Commit(); err != nil {
					b.Errorf("failed to commit transaction: %v", err)
				}
				tx = nil
			}
			return
		}
		// Commit
		if (i+1)%benchmarkQueriesPerTx == 0 {
			if err = tx.Commit(); err != nil {
				b.Errorf("failed to commit transaction: %v", err)
			}
			tx = nil
		}
	}
}

// GR is a get reader function passed to benchmark helper.
//type GR func(xi.Storage) *sql.DB
//
//func getDirtyReader(st xi.Storage) *sql.DB { return st.DirtyReader() }
//func getReader(st xi.Storage) *sql.DB      { return st.Reader() }
//
//func benchmarkStorageSequentialReadWithBackgroundWriter(b *testing.B, getReader GR, write BW) {
//	var (
//		st, q, dm, e, src = setupBenchmarkStorage(b)
//
//		dest = dm()
//		wg   = &sync.WaitGroup{}
//		sc   = make(chan struct{})
//
//		err error
//	)
//
//	// Start background writer
//	wg.Add(1)
//	go write(b, wg, sc, st, ipkg, e, src)
//
//	for i := 0; i < b.N; i++ {
//		if err = getReader(st).QueryRow(
//			q, trkg.next(),
//		).Scan(dest...); err != nil && err != sql.ErrNoRows {
//			b.Fatalf("failed to query values: %v", err)
//		}
//	}
//
//	// Exit background writer
//	close(sc)
//	wg.Wait()
//
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStoargeSequentialDirtyReadWithBackgroundWriter(b *testing.B) {
//	benchmarkStorageSequentialReadWithBackgroundWriter(b, getDirtyReader, busyWrite)
//}
//
//func BenchmarkStoargeSequentialReadWithBackgroundWriter(b *testing.B) {
//	benchmarkStorageSequentialReadWithBackgroundWriter(b, getReader, busyWrite)
//}
//
//func BenchmarkStoargeSequentialDirtyReadWithBackgroundBusyTxWriter(b *testing.B) {
//	benchmarkStorageSequentialReadWithBackgroundWriter(b, getDirtyReader, busyWriteTx)
//}
//
//func BenchmarkStoargeSequentialReadWithBackgroundBusyTxWriter(b *testing.B) {
//	benchmarkStorageSequentialReadWithBackgroundWriter(b, getReader, busyWriteTx)
//}
//
//func BenchmarkStoargeSequentialDirtyReadWithBackgroundIdleTxWriter(b *testing.B) {
//	benchmarkStorageSequentialReadWithBackgroundWriter(b, getDirtyReader, idleWriteTx)
//}
//
//func BenchmarkStoargeSequentialReadWithBackgroundIdleTxWriter(b *testing.B) {
//	benchmarkStorageSequentialReadWithBackgroundWriter(b, getReader, idleWriteTx)
//}
//
//func benchmarkStorageSequentialReadTxWithBackgroundWriter(b *testing.B, getReader GR, write BW) {
//	var (
//		st, q, dm, e, src = setupBenchmarkStorage(b)
//
//		dest = dm()
//		wg   = &sync.WaitGroup{}
//		sc   = make(chan struct{})
//
//		err error
//		tx  *sql.Tx
//	)
//
//	// Start background writer
//	wg.Add(1)
//	go write(b, wg, sc, st, ipkg, e, src)
//
//	for i := 0; i < b.N; i++ {
//		if i%benchmarkQueriesPerTx == 0 {
//			if tx, err = getReader(st).Begin(); err != nil {
//				b.Fatalf("failed to begin transaction: %v", err)
//			}
//		}
//		// Query in [n, 2n-1] key space
//		if err = tx.QueryRow(q, nrkg.next()).Scan(dest...); err != nil && err != sql.ErrNoRows {
//			b.Fatalf("failed to query values: %v", err)
//		}
//		if (i+1)%benchmarkQueriesPerTx == 0 || i == b.N-1 {
//			if err = tx.Rollback(); err != nil {
//				b.Fatalf("failed to close transaction: %v", err)
//			}
//		}
//	}
//
//	// Exit background writer
//	close(sc)
//	wg.Wait()
//
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStoargeSequentialDirtyReadTxWithBackgroundWriter(b *testing.B) {
//	benchmarkStorageSequentialReadTxWithBackgroundWriter(b, getDirtyReader, busyWrite)
//}
//
//func BenchmarkStoargeSequentialReadTxWithBackgroundWriter(b *testing.B) {
//	benchmarkStorageSequentialReadTxWithBackgroundWriter(b, getReader, busyWrite)
//}
//
//func BenchmarkStoargeSequentialDirtyReadTxWithBackgroundBusyTxWriter(b *testing.B) {
//	benchmarkStorageSequentialReadTxWithBackgroundWriter(b, getDirtyReader, busyWriteTx)
//}
//
//func BenchmarkStoargeSequentialReadTxWithBackgroundBusyTxWriter(b *testing.B) {
//	benchmarkStorageSequentialReadTxWithBackgroundWriter(b, getReader, busyWriteTx)
//}
//
//func BenchmarkStoargeSequentialDirtyReadTxWithBackgroundIdleTxWriter(b *testing.B) {
//	benchmarkStorageSequentialReadTxWithBackgroundWriter(b, getDirtyReader, idleWriteTx)
//}
//
//func BenchmarkStoargeSequentialReadTxWithBackgroundIdleTxWriter(b *testing.B) {
//	benchmarkStorageSequentialReadTxWithBackgroundWriter(b, getReader, idleWriteTx)
//}
//
//func BenchmarkStoargeSequentialMixDRW(b *testing.B) {
//	var (
//		st, q, dm, e, src = setupBenchmarkStorage(b)
//		dest              = dm()
//		err               error
//	)
//	for i := 0; i < b.N; i++ {
//		if rand.Int()%2 == 0 {
//			if err = st.DirtyReader().QueryRow(q, rrkg.next()).Scan(dest...); err != nil {
//				b.Fatalf("failed to query values: %v", err)
//			}
//		} else {
//			if _, err = st.Writer().Exec(e, src[ipkg.next()]...); err != nil {
//				b.Fatalf("failed to execute: %v", err)
//			}
//		}
//	}
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStoargeSequentialMixRW(b *testing.B) {
//	var (
//		st, q, dm, e, src = setupBenchmarkStorage(b)
//		dest              = dm()
//		err               error
//	)
//	for i := 0; i < b.N; i++ {
//		if rand.Int()%2 == 0 {
//			if err = st.Reader().QueryRow(q, rrkg.next()).Scan(dest...); err != nil {
//				b.Fatalf("failed to query values: %v", err)
//			}
//		} else {
//			if _, err = st.Writer().Exec(e, src[ipkg.next()]...); err != nil {
//				b.Fatalf("failed to execute: %v", err)
//			}
//		}
//	}
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStorageParallelDirtyRead(b *testing.B) {
//	var (
//		st, q, dm, _, _ = setupBenchmarkStorage(b)
//	)
//	b.RunParallel(func(pb *testing.PB) {
//		var (
//			dest = dm()
//			err  error
//		)
//		for pb.Next() {
//			if err = st.DirtyReader().QueryRow(q, rrkg.next()).Scan(dest...); err != nil {
//				b.Fatalf("failed to query values: %v", err)
//			}
//		}
//	})
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStorageParallelRead(b *testing.B) {
//	var (
//		st, q, dm, _, _ = setupBenchmarkStorage(b)
//	)
//	b.RunParallel(func(pb *testing.PB) {
//		var (
//			dest = dm()
//			err  error
//		)
//		for pb.Next() {
//			if err = st.DirtyReader().QueryRow(q, rrkg.next()).Scan(dest...); err != nil {
//				b.Fatalf("failed to query values: %v", err)
//			}
//		}
//	})
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStoargeParallelWrite(b *testing.B) {
//	var st, _, _, e, src = setupBenchmarkStorage(b)
//	b.RunParallel(func(pb *testing.PB) {
//		var err error
//		for pb.Next() {
//			if _, err = st.Writer().Exec(e, src[ipkg.next()]...); err != nil {
//				b.Fatalf("failed to execute: %v", err)
//			}
//		}
//	})
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStorageParallelMixDRW(b *testing.B) {
//	var st, q, dm, e, src = setupBenchmarkStorage(b)
//	b.RunParallel(func(pb *testing.PB) {
//		var (
//			dest = dm()
//			err  error
//		)
//		for pb.Next() {
//			if rand.Int()%2 == 0 {
//				if err = st.DirtyReader().QueryRow(q, rrkg.next()).Scan(dest...); err != nil {
//					b.Fatalf("failed to query values: %v", err)
//				}
//			} else {
//				if _, err = st.Writer().Exec(e, src[ipkg.next()]...); err != nil {
//					b.Fatalf("failed to execute: %v", err)
//				}
//			}
//		}
//	})
//	teardownBenchmarkStorage(b, st)
//}
//
//func BenchmarkStorageParallelMixRW(b *testing.B) {
//	var st, q, dm, e, src = setupBenchmarkStorage(b)
//	b.RunParallel(func(pb *testing.PB) {
//		var (
//			dest = dm()
//			err  error
//		)
//		for pb.Next() {
//			if rand.Int()%2 == 0 {
//				if err = st.Reader().QueryRow(q, rrkg.next()).Scan(dest...); err != nil {
//					b.Fatalf("failed to query values: %v", err)
//				}
//			} else {
//				if _, err = st.Writer().Exec(e, src[ipkg.next()]...); err != nil {
//					b.Fatalf("failed to execute: %v", err)
//				}
//			}
//		}
//	})
//	teardownBenchmarkStorage(b, st)
//}
