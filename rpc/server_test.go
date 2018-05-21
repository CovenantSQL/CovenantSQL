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

package rpc

import (
	"net"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/crypto/etls"
	"github.com/thunderdb/ThunderDB/utils"
)

type TestService struct {
	counter int
}

type TestReq struct {
	Step int
}

type TestRep struct {
	Ret int
}

func NewTestService() *TestService {
	return &TestService{
		counter: 0,
	}
}

func (s *TestService) IncCounter(req *TestReq, rep *TestRep) error {
	log.Debugf("calling IncCounter req:%v, rep:%v", *req, *rep)
	s.counter += req.Step
	rep.Ret = s.counter
	return nil
}

func (s *TestService) IncCounterSimpleArgs(step int, ret *int) error {
	log.Debugf("calling IncCounter req:%v, rep:%v", step, ret)
	s.counter += step
	*ret = s.counter
	return nil
}

func TestIncCounter(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	server.SetListener(l)
	go server.Serve()

	rep := new(TestRep)
	client, err := InitClient(l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}

	err = client.Call("Test.IncCounter", &TestReq{Step: 10}, rep)
	if err != nil {
		log.Fatal(err)
	}
	utils.CheckNum(rep.Ret, 10, t)

	err = client.Call("Test.IncCounter", &TestReq{Step: 10}, rep)
	if err != nil {
		log.Fatal(err)
	}
	utils.CheckNum(rep.Ret, 20, t)

	repSimple := new(int)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	utils.CheckNum(*repSimple, 30, t)

	client.Close()
	server.Stop()
}

func TestIncCounterSimpleArgs(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	server.SetListener(l)
	go server.Serve()

	client, err := InitClient(l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}

	repSimple := new(int)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	utils.CheckNum(*repSimple, 10, t)

	client.Close()
	server.Stop()
}

func TestEncryptIncCounterSimpleArgs(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	pass := "12345"

	l, err := etls.NewCryptoListener("tcp", addr, pass)
	if err != nil {
		log.Fatal(err)
	}

	server, err := NewServerWithService(ServiceMap{"Test": NewTestService()})
	if err != nil {
		log.Fatal(err)
	}

	server.SetListener(l)
	go server.Serve()

	cipher := etls.NewCipher([]byte(pass))
	conn, err := etls.Dial("tcp", l.Addr().String(), cipher)
	if err != nil {
		log.Fatal(err)
	}

	client, err := InitClientConn(conn)
	if err != nil {
		log.Fatal(err)
	}

	repSimple := new(int)
	err = client.Call("Test.IncCounterSimpleArgs", 10, repSimple)
	if err != nil {
		log.Fatal(err)
	}
	utils.CheckNum(*repSimple, 10, t)

	client.Close()
	server.Stop()
}

func TestServer_Close(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:0"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	server := NewServer()
	testService := NewTestService()
	err = server.RegisterService("Test", testService)
	if err != nil {
		log.Fatal(err)
	}
	server.SetListener(l)
	go server.Serve()

	server.Stop()
}
