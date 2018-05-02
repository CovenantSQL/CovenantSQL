package rpc

import (
	log "github.com/sirupsen/logrus"
	"github.com/thunderdb/ThunderDB/utils"
	"net"
	"testing"
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
