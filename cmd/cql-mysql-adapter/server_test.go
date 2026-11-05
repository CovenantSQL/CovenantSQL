/*
 * Copyright 2026 The CovenantSQL Authors.
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
	"runtime"
	"testing"
	"time"

	myclient "github.com/go-mysql-org/go-mysql/client"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConnClosesDatabases(t *testing.T) {
	Convey("a connection closes the databases it opened", t, func() {
		s, err := NewServer("127.0.0.1:0", "root", "calvin")
		So(err, ShouldBeNil)
		go s.Serve()
		defer s.Shutdown()

		// Open db1 in the handshake, switch to db2, then disconnect.
		session := func() {
			conn, err := myclient.Connect(s.listener.Addr().String(), "root", "calvin", "db1")
			So(err, ShouldBeNil)
			_, err = conn.Execute("USE db2")
			So(err, ShouldBeNil)
			So(conn.Close(), ShouldBeNil)
		}

		// The first session starts whatever runs for the life of the process.
		session()
		before := runtime.NumGoroutine()
		for i := 0; i < 20; i++ {
			session()
		}

		// Every open sql.DB keeps a goroutine until it is closed, and the
		// connections end asynchronously on the server side.
		deadline := time.Now().Add(10 * time.Second)
		for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
		So(runtime.NumGoroutine(), ShouldBeLessThanOrEqualTo, before)
	})
}
