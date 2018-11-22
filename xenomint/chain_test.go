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
	"testing"

	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/types"
)

func setupBenchmarkChain(b *testing.B) (c *Chain, n int, r []*types.Request) {
	// Setup chain state
	var (
		fl   = path.Join(testingDataDir, b.Name())
		err  error
		stmt *sql.Stmt
	)
	if c, err = NewChain(fmt.Sprint("file:", fl)); err != nil {
		b.Fatalf("Failed to setup bench environment: %v", err)
	}
	if _, err = c.state.strg.Writer().Exec(
		`CREATE TABLE "bench" ("k" INT, "v1" TEXT, "v2" TEXT, "v3" TEXT, PRIMARY KEY("k"))`,
	); err != nil {
		b.Fatalf("Failed to setup bench environment: %v", err)
	}
	if stmt, err = c.state.strg.Writer().Prepare(
		`INSERT INTO "bench" VALUES (?, ?, ?, ?)`,
	); err != nil {
		b.Fatalf("Failed to setup bench environment: %v", err)
	}
	for i := 0; i < benchmarkKeySpace; i++ {
		var (
			vals [benchmarkVNum][benchmarkVLen]byte
			args [benchmarkVNum + 1]interface{}
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
	n = benchmarkKeySpace
	// Setup query requests
	var (
		sel  = `SELECT v1, v2, v3 FROM bench WHERE k=?`
		ins  = `INSERT OR REPLACE INTO bench VALUES (?, ?, ?, ?)`
		priv *ca.PrivateKey
		src  = make([][]interface{}, benchmarkKeySpace)
	)
	if priv, err = kms.GetLocalPrivateKey(); err != nil {
		b.Fatalf("Failed to setup bench environment: %v", err)
	}
	r = make([]*types.Request, 2*benchmarkKeySpace)
	// Read query key space [0, n-1]
	for i := 0; i < benchmarkKeySpace; i++ {
		r[i] = buildRequest(types.ReadQuery, []types.Query{
			buildQuery(sel, i),
		})
		if err = r[i].Sign(priv); err != nil {
			b.Fatalf("Failed to setup bench environment: %v", err)
		}
	}
	// Write query key space [n, 2n-1]
	for i := range src {
		var vals [benchmarkVNum][benchmarkVLen]byte
		src[i] = make([]interface{}, benchmarkVNum+1)
		src[i][0] = i + benchmarkKeySpace
		for j := range vals {
			rand.Read(vals[j][:])
			src[i][j+1] = string(vals[j][:])
		}
	}
	for i := 0; i < benchmarkKeySpace; i++ {
		r[benchmarkKeySpace+i] = buildRequest(types.WriteQuery, []types.Query{
			buildQuery(ins, src[i]...),
		})
		if err = r[i+benchmarkKeySpace].Sign(priv); err != nil {
			b.Fatalf("Failed to setup bench environment: %v", err)
		}
	}

	b.ResetTimer()
	return
}

func teardownBenchmarkChain(b *testing.B, c *Chain) {
	b.StopTimer()

	var (
		fl  = path.Join(testingDataDir, b.Name())
		err error
	)
	if err = c.Stop(); err != nil {
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

func BenchmarkChainParallelWrite(b *testing.B) {
	var c, n, r = setupBenchmarkChain(b)
	b.RunParallel(func(pb *testing.PB) {
		var err error
		for i := 0; pb.Next(); i++ {
			if _, err = c.Query(r[n+rand.Intn(n)]); err != nil {
				b.Fatalf("Failed to execute: %v", err)
			}
			if (i+1)%benchmarkQueriesPerBlock == 0 {
				if err = c.state.commit(); err != nil {
					b.Fatalf("Failed to commit block: %v", err)
				}
			}
		}
	})
	teardownBenchmarkChain(b, c)
}

func BenchmarkChainParallelMixRW(b *testing.B) {
	var c, n, r = setupBenchmarkChain(b)
	b.RunParallel(func(pb *testing.PB) {
		var err error
		for i := 0; pb.Next(); i++ {
			if _, err = c.Query(r[rand.Intn(2*n)]); err != nil {
				b.Fatalf("Failed to execute: %v", err)
			}
			if (i+1)%benchmarkQueriesPerBlock == 0 {
				if err = c.state.commit(); err != nil {
					b.Fatalf("Failed to commit block: %v", err)
				}
			}
		}
	})
	teardownBenchmarkChain(b, c)
}
