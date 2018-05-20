/*
 * Copyright 2018 The ThunderDB Authors.
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

package transport

import (
	"net"
	"net/rpc"
	"testing"

	"strings"

	"time"

	log "github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
)

type Foo bool

type Result struct {
	Data int
}

func (f *Foo) Bar(args *string, res *Result) error {
	res.Data = len(*args)
	log.Printf("Received %q, its length is %d", *args, res.Data)
	//return fmt.Errorf("Whoops, error happened")
	return nil
}

const service = "127.0.0.1:28000"

func server(pass string) *CryptoListener {
	if err := rpc.Register(new(Foo)); err != nil {
		log.Error("Failed to register RPC method")
	}

	listener, err := NewCryptoListener("tcp", service, pass)
	//listener, err := net.Listen("tcp", service)
	if err != nil {
		log.Errorf("server: listen: %s", err)
	}
	log.Print("server: listening")
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("server: accept: %s", err)
				break
			}
			log.Printf("server: accepted from %s", conn.RemoteAddr())
			go handleClient(conn)
		}
	}()
	return listener
}

func client(pass string) (ret int, err error) {

	cipher := NewCipher([]byte(pass))

	conn, err := Dial("tcp", service, cipher)
	//conn, err := net.Dial("tcp", service)
	if err != nil {
		log.Errorf("client: dial: %s", err)
		return 0, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Time{})
	conn.SetReadDeadline(time.Time{})
	conn.SetWriteDeadline(time.Time{})

	log.Println("client: connected to: ", conn.RemoteAddr())
	rpcClient := rpc.NewClient(conn)
	res := new(Result)
	args := strings.Repeat("a", 9999)
	if err := rpcClient.Call("Foo.Bar", args, &res); err != nil {
		log.Error("Failed to call RPC", err)
		return 0, err
	}
	log.Printf("Returned result is %d", res.Data)
	return res.Data, err
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	rpc.ServeConn(conn)
	log.Println("server: conn: closed")
}

func TestConn(t *testing.T) {
	const pass = "123"
	l := server(pass)
	Convey("get addr", t, func() {
		addr := l.Addr().String()
		So(addr, ShouldEqual, service)
	})
	Convey("server client OK", t, func() {
		ret, err := client(pass)
		So(ret, ShouldEqual, 9999)
		So(err, ShouldBeNil)
	})
	Convey("pass not match", t, func() {
		ret, err := client("1234")
		So(ret, ShouldEqual, 0)
		So(err, ShouldNotBeNil)
	})
	Convey("server close", t, func() {
		err := l.Close()
		So(err, ShouldBeNil)
	})
}
