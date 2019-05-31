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

package rpc

import (
	"errors"
	"net"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/etls"
)

func TestAcceptFunc(t *testing.T) {
	Convey("Setup a single server with raw conn accept func", t, func(c C) {
		nodes, stop, err := setupEnvironment(1, AcceptRawConn)
		So(err, ShouldBeNil)
		defer stop()

		conn, err := net.Dial("tcp", nodes[0].Addr)
		So(err, ShouldBeNil)
		cli := NewClient(conn)
		defer func() { _ = cli.Close() }()
		err = cli.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{})
		So(err, ShouldBeNil)
	})
	Convey("Setup a single server with crypto conn accept func", t, func(c C) {
		secret := `6?$X7$<OmTFR{<!O}`
		f := func(conn net.Conn) (*etls.CryptoConn, error) {
			return etls.NewConn(conn, etls.NewCipher([]byte(secret))), nil
		}

		nodes, stop, err := setupEnvironment(1, NewAcceptCryptoConnFunc(f))
		So(err, ShouldBeNil)
		defer stop()

		conn, err := etls.Dial("tcp", nodes[0].Addr, etls.NewCipher([]byte(secret)))
		So(err, ShouldBeNil)
		cli := NewClient(conn)
		defer func() { _ = cli.Close() }()
		err = cli.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{})
		So(err, ShouldBeNil)

		// Call crypto server with raw TCP conn
		rawconn, err := net.Dial("tcp", nodes[0].Addr)
		So(err, ShouldBeNil)
		rawcli := NewClient(rawconn)
		defer func() { _ = rawcli.Close() }()
		err = NewClient(rawconn).Call("Count.Add", &AddReq{Delta: 1}, &AddResp{})
		So(err, ShouldNotBeNil)
	})
	Convey("Setup a single server with bad crypto conn accept func", t, func(c C) {
		f := func(net.Conn) (*etls.CryptoConn, error) {
			return nil, errors.New("won't accept")
		}

		nodes, stop, err := setupEnvironment(1, NewAcceptCryptoConnFunc(f))
		So(err, ShouldBeNil)
		defer stop()

		conn, err := etls.Dial("tcp", nodes[0].Addr, etls.NewCipher([]byte{}))
		So(err, ShouldBeNil)
		cli := NewClient(conn)
		defer func() { _ = cli.Close() }()
		err = cli.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{})
		So(err, ShouldNotBeNil)
	})
	Convey("Setup a single server with naconn accept func", t, func(c C) {
		nodes, stop, err := setupEnvironment(1, AcceptNAConn)
		So(err, ShouldBeNil)
		defer stop()

		conn, err := Dial(nodes[0].ID)
		So(err, ShouldBeNil)
		cli := NewClient(conn)
		defer func() { _ = cli.Close() }()
		err = cli.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{})
		So(err, ShouldBeNil)

		// Call naconn server with raw TCP conn
		rawconn, err := net.Dial("tcp", nodes[0].Addr)
		So(err, ShouldBeNil)
		rawcli := NewClient(rawconn)
		defer func() { _ = rawcli.Close() }()
		err = rawcli.Call("Count.Add", &AddReq{Delta: 1}, &AddResp{})
		So(err, ShouldNotBeNil)
	})
}
