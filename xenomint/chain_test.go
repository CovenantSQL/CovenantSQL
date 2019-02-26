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
	"math/rand"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/types"
)

func setupChain(testName string) (c *Chain, err error) {
	// Setup chain state
	var (
		fl = path.Join(testingDataDir, strings.Replace(testName, "/", "-", -1))
	)
	if c, err = NewChain(fmt.Sprint("file:", fl)); err != nil {
		err = errors.Wrap(err, "failed to setup bench environment: ")
		return
	}
	if _, err = c.state.strg.Writer().Exec(
		`CREATE TABLE "bench" ("k" INT, "v1" TEXT, "v2" TEXT, "v3" TEXT, PRIMARY KEY("k"))`,
	); err != nil {
		err = errors.Wrap(err, "failed to setup bench environment: ")
		return
	}
	return
}

func setupBenchmarkChainRequest(b *testing.B) (r []*types.Request) {
	// Setup query requests
	var (
		sel  = `SELECT v1, v2, v3 FROM bench WHERE k=?`
		ins  = `INSERT INTO bench VALUES (?, ?, ?, ?)`
		priv *ca.PrivateKey
		src  = make([][]interface{}, benchmarkNewKeyLength)
		err  error
	)
	if priv, err = kms.GetLocalPrivateKey(); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	r = make([]*types.Request, benchmarkMaxKey)
	// Read query key space [0, n-1]
	for i := 0; i < benchmarkReservedKeyLength; i++ {
		r[i] = buildRequest(types.ReadQuery, []types.Query{
			buildQuery(sel, i+benchmarkReservedKeyOffset),
		})
		if err = r[i].Sign(priv); err != nil {
			b.Fatalf("failed to setup bench environment: %v", err)
		}
	}
	// Write query key space [n, 2n-1]
	for i := range src {
		var vals [benchmarkVNum][benchmarkVLen]byte
		src[i] = make([]interface{}, benchmarkVNum+1)
		src[i][0] = i + benchmarkNewKeyOffset
		for j := range vals {
			rand.Read(vals[j][:])
			src[i][j+1] = string(vals[j][:])
		}
	}
	for i := 0; i < benchmarkNewKeyLength; i++ {
		r[i+benchmarkNewKeyOffset] = buildRequest(types.WriteQuery, []types.Query{
			buildQuery(ins, src[i]...),
		})
		if err = r[i+benchmarkNewKeyOffset].Sign(priv); err != nil {
			b.Fatalf("Failed to setup bench environment: %v", err)
		}
	}
	return
}

func setupBenchmarkChain(b *testing.B) (c *Chain) {
	var (
		err  error
		stmt *sql.Stmt
	)

	c, err = setupChain(b.Name())
	if err != nil {
		b.Fatal(err)
	}

	if stmt, err = c.state.strg.Writer().Prepare(
		`INSERT INTO "bench" VALUES (?, ?, ?, ?)`,
	); err != nil {
		b.Fatalf("failed to setup bench environment: %v", err)
	}
	for i := 0; i < benchmarkReservedKeyLength; i++ {
		var (
			vals [benchmarkVNum][benchmarkVLen]byte
			args [benchmarkVNum + 1]interface{}
		)
		args[0] = i + benchmarkReservedKeyOffset
		for i := range vals {
			rand.Read(vals[i][:])
			args[i+1] = string(vals[i][:])
		}
		if _, err = stmt.Exec(args[:]...); err != nil {
			b.Fatalf("failed to setup bench environment: %v", err)
		}
	}

	allKeyPermKeygen.reset()
	newKeyPermKeygen.reset()

	b.ResetTimer()
	return
}

func teardownChain(testName string, c *Chain) (err error) {
	var (
		fl = path.Join(testingDataDir, strings.Replace(testName, "/", "-", -1))
	)
	if err = c.Stop(); err != nil {
		err = errors.Wrap(err, "failed to teardown bench environment: ")
		return
	}
	if err = os.Remove(fl); err != nil {
		err = errors.Wrap(err, "failed to teardown bench environment: ")
		return
	}
	if err = os.Remove(fmt.Sprint(fl, "-shm")); err != nil && !os.IsNotExist(err) {
		err = errors.Wrap(err, "failed to teardown bench environment: ")
		return
	}
	if err = os.Remove(fmt.Sprint(fl, "-wal")); err != nil && !os.IsNotExist(err) {
		err = errors.Wrap(err, "failed to teardown bench environment: ")
		return
	}

	return nil
}

func teardownBenchmarkChain(b *testing.B, c *Chain) {
	b.StopTimer()

	err := teardownChain(b.Name(), c)
	if err != nil {
		b.Fatal(err)
	}
}

func BenchmarkChainParallelWrite(b *testing.B) {
	var r = setupBenchmarkChainRequest(b)
	var c = setupBenchmarkChain(b)
	b.RunParallel(func(pb *testing.PB) {
		var (
			err     error
			counter int32
		)
		for pb.Next() {
			if _, err = c.Query(r[newKeyPermKeygen.next()]); err != nil {
				b.Fatalf("Failed to execute: %v", err)
			}
			if atomic.AddInt32(&counter, 1)%benchmarkQueriesPerBlock == 0 {
				if err = c.state.commit(); err != nil {
					b.Fatalf("failed to commit block: %v", err)
				}
			}
		}
	})
	teardownBenchmarkChain(b, c)
}

func BenchmarkChainParallelMixRW(b *testing.B) {
	var r = setupBenchmarkChainRequest(b)
	var c = setupBenchmarkChain(b)
	b.RunParallel(func(pb *testing.PB) {
		var (
			err     error
			counter int32
		)
		for pb.Next() {
			if _, err = c.Query(r[allKeyPermKeygen.next()]); err != nil {
				b.Fatalf("Failed to execute: %v", err)
			}
			if atomic.AddInt32(&counter, 1)%benchmarkQueriesPerBlock == 0 {
				if err = c.state.commit(); err != nil {
					b.Fatalf("failed to commit block: %v", err)
				}
			}
		}
	})
	teardownBenchmarkChain(b, c)
}

func TestChain(t *testing.T) {
	Convey("test xenomint chain", t, func() {
		var c, err = setupChain(t.Name())
		So(err, ShouldBeNil)

		// Setup query requests
		var (
			sel  = `SELECT v1, v2, v3 FROM bench WHERE k=?`
			ins  = `INSERT INTO bench VALUES (?, ?, ?, ?)`
			rr   *types.Request
			wr   *types.Request
			priv *ca.PrivateKey
		)
		priv, err = kms.GetLocalPrivateKey()
		So(err, ShouldBeNil)

		// Write query
		wr = buildRequest(types.WriteQuery, []types.Query{
			buildQuery(ins, 0, 1, 2, 3),
		})
		err = wr.Sign(priv)
		So(err, ShouldBeNil)

		_, err = c.Query(wr)
		So(err, ShouldBeNil)

		// Read query
		rr = buildRequest(types.ReadQuery, []types.Query{
			buildQuery(sel, 0),
		})
		err = rr.Sign(priv)
		So(err, ShouldBeNil)

		_, err = c.Query(rr)
		So(err, ShouldBeNil)

		err = teardownChain(t.Name(), c)
		So(err, ShouldBeNil)
	})
}
