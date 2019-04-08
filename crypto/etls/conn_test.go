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

package etls

import (
	"net"
	"net/rpc"
	"strings"
	"sync"
	"testing"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const service = "127.0.0.1:28000"
const serviceComplex = "127.0.0.1:28001"
const contentLength = 9999
const pass = "123"

var simpleCipherHandler CipherHandler = func(conn net.Conn) (cryptoConn *CryptoConn, err error) {
	cipher := NewCipher([]byte(pass))
	cryptoConn = NewConn(conn, cipher)
	return
}

type Foo bool

type Result struct {
	Data int
}

func (f *Foo) Bar(args *string, res *Result) error {
	res.Data = len(*args)
	log.Debugf("Received %q, its length is %d", *args, res.Data)
	return nil
}

type FooComplex bool

type QueryComplex struct {
	DataS struct {
		Strs []string
	}
}

type ResultComplex struct {
	Count int
	Hash  struct {
		Str string
	}
}

func qHash(q *QueryComplex) string {
	return string(hash.THashB([]byte(strings.Join(q.DataS.Strs, ""))))
}

func (f *FooComplex) Bar(args *QueryComplex, res *ResultComplex) error {
	res.Count = len(args.DataS.Strs)
	res.Hash.Str = qHash(args)
	log.Debugf("Received %v", *args)
	return nil
}

func server() *CryptoListener {
	if err := rpc.Register(new(Foo)); err != nil {
		log.Error("failed to register RPC method")
	}

	listener, err := NewCryptoListener("tcp", service, simpleCipherHandler)
	//listener, err := net.Listen("tcp", service)
	if err != nil {
		log.Errorf("server: listen: %s", err)
	}
	log.Debug("server: listening")
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Debugf("server: accept: %s", err)
				break
			}
			log.Debugf("server: accepted from %s", conn.RemoteAddr())
			go handleClient(conn)
		}
	}()
	return listener
}

func client(pass string) (ret int, err error) {

	cipher := NewCipher([]byte(pass))

	conn, err := Dial("tcp", service, cipher)
	//conn, err := net.dial("tcp", service)
	if err != nil {
		log.Errorf("client: dial: %s", err)
		return 0, err
	}
	defer func() { _ = conn.Close() }()
	//conn.SetDeadline(time.Time{})
	//conn.SetReadDeadline(time.Time{})
	//conn.SetWriteDeadline(time.Time{})

	log.Debugln("client: connected to: ", conn.RemoteAddr())
	log.Debugln("client: LocalAddr: ", conn.LocalAddr())
	rpcClient := rpc.NewClient(conn)
	res := new(Result)
	args := strings.Repeat("a", contentLength)
	if err := rpcClient.Call("Foo.Bar", args, &res); err != nil {
		log.Errorf("failed to call RPC %v", err)
		return 0, err
	}
	log.Debugf("Returned result is %d", res.Data)
	return res.Data, err
}

func serverComplex() *CryptoListener {
	if err := rpc.Register(new(FooComplex)); err != nil {
		log.Error("failed to register RPC method")
	}

	listener, err := NewCryptoListener("tcp", serviceComplex, simpleCipherHandler)
	if err != nil {
		log.Errorf("server: listen: %s", err)
	}
	log.Debug("server: listening")
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Debugf("server: accept: %s", err)
				break
			}
			log.Debugf("server: accepted from %s", conn.RemoteAddr())
			go handleClient(conn)
		}
	}()
	return listener
}

func clientComplex(pass string, args *QueryComplex) (ret *ResultComplex, err error) {
	cipher := NewCipher([]byte(pass))

	conn, err := Dial("tcp", serviceComplex, cipher)
	if err != nil {
		log.Errorf("client: dial: %s", err)
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	log.Debugln("client: connected to: ", conn.RemoteAddr())
	log.Debugln("client: LocalAddr: ", conn.LocalAddr())
	rpcClient := rpc.NewClient(conn)
	res := new(ResultComplex)

	if err := rpcClient.Call("FooComplex.Bar", args, &res); err != nil {
		log.Errorf("failed to call RPC %v", err)
		return nil, err
	}
	return res, err
}

func handleClient(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	var err error

	if c, ok := conn.(*CryptoConn); ok {
		conn, err = simpleCipherHandler(c.Conn)
		if err != nil {
			err = errors.Wrap(err, "handle ETLS handler failed")
			return
		}
	}

	rpc.ServeConn(conn)
	log.Debugln("server: conn: closed")
}

func TestSimpleRPC(t *testing.T) {
	l := server()
	Convey("get addr", t, func() {
		addr := l.Addr().String()
		So(addr, ShouldEqual, service)
	})
	Convey("server client OK", t, func() {
		ret, err := client(pass)
		So(ret, ShouldEqual, contentLength)
		So(err, ShouldBeNil)
	})
	Convey("server client OK 100", t, func() {
		for i := 0; i < 100; i++ {
			ret, err := client(pass)
			So(ret, ShouldEqual, contentLength)
			So(err, ShouldBeNil)
		}
	})
	Convey("pass not match", t, func() {
		ret, err := client("1234")
		So(ret, ShouldEqual, 0)
		So(err, ShouldNotBeNil)
	})
	Convey("pass not match 100", t, func() {
		for i := 0; i < 100; i++ {
			ret, err := client("12345")
			So(ret, ShouldEqual, 0)
			So(err, ShouldNotBeNil)
		}
	})
	Convey("server close", t, func() {
		err := l.Close()
		So(err, ShouldBeNil)
	})
}

func TestComplexRPC(t *testing.T) {
	l := serverComplex()
	Convey("get addr", t, func() {
		addr := l.Addr().String()
		So(addr, ShouldEqual, serviceComplex)
	})
	Convey("server client OK", t, func() {
		argsComplex := &QueryComplex{
			DataS: struct{ Strs []string }{Strs: []string{"aaa", "bbbbbbb"}},
		}
		ret, err := clientComplex(pass, argsComplex)
		So(ret.Count, ShouldEqual, len(argsComplex.DataS.Strs))
		So(ret.Hash.Str, ShouldResemble, qHash(argsComplex))
		So(err, ShouldBeNil)
	})
	Convey("server client pass error", t, func() {
		argsComplex := &QueryComplex{
			DataS: struct{ Strs []string }{Strs: []string{"aaa", "bbbbbbb"}},
		}
		ret, err := clientComplex(pass+"1", argsComplex)
		So(ret, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})

	Convey("server client pass error", t, func() {
		argsComplex := &QueryComplex{
			DataS: struct{ Strs []string }{Strs: []string{"aaa", "bbbbbbb"}},
		}
		ret, err := clientComplex(strings.Repeat(pass, 100), argsComplex)
		So(ret, ShouldBeNil)
		So(err, ShouldNotBeNil)
	})

	Convey("server close", t, func() {
		err := l.Close()
		So(err, ShouldBeNil)
	})
}

func TestCryptoConn_RW(t *testing.T) {
	cipher := NewCipher([]byte(pass))
	var nilCipherHandler CipherHandler = func(conn net.Conn) (cryptoConn *CryptoConn, err error) {
		cryptoConn = NewConn(conn, cipher)
		return
	}

	Convey("server client OK", t, func(c C) {
		msg := "xxxxxxxxxxxxxxxx"
		l, _ := NewCryptoListener("tcp", "127.0.0.1:0", nilCipherHandler)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			rBuf := make([]byte, len(msg))
			conn, err := l.Accept()

			if c, ok := conn.(*CryptoConn); ok {
				conn, err = l.CHandler(c.Conn)
				if err != nil {
					err = errors.Wrap(err, "handle ETLS handler failed")
					return
				}
			}

			n, err := conn.Read(rBuf)
			c.So(n, ShouldEqual, len(msg))
			c.So(string(rBuf), ShouldResemble, msg)
			c.So(err, ShouldBeNil)
			wg.Done()
		}()
		conn, _ := Dial("tcp", l.Addr().String(), cipher)
		n, err := conn.Write([]byte(msg))
		So(n, ShouldEqual, len(msg))
		So(err, ShouldBeNil)
		wg.Wait()
	})
}
