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

package rpc

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	RPCConcurrent = 10
	RPCCount      = 100
)

func TestCaller_CallNode(t *testing.T) {
	log.SetLevel(log.FatalLevel)
	os.Remove(PubKeyStorePath)
	defer os.Remove(PubKeyStorePath)
	os.Remove(publicKeyStore)
	defer os.Remove(publicKeyStore)

	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/config.yaml")
	privateKeyPath := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/private.key")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %#v", conf.GConf)
	// reset the once
	route.Once = sync.Once{}
	route.InitKMS(publicKeyStore)

	addr := conf.GConf.ListenAddr // see ../test/node_standalone/config.yaml
	masterKey := []byte("")
	dht, err := route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)

	server, err := NewServerWithService(ServiceMap{route.DHTRPCName: dht})
	if err != nil {
		t.Fatal(err)
	}

	server.InitRPCServer(addr, privateKeyPath, masterKey)
	go server.Serve()

	//publicKey, err := kms.GetLocalPublicKey()
	//nonce := asymmetric.GetPubKeyNonce(publicKey, 10, 100*time.Millisecond, nil)
	//serverNodeID := proto.NodeID(nonce.Hash.String())
	//kms.SetPublicKey(serverNodeID, nonce.Nonce, publicKey)
	//
	//kms.SetLocalNodeIDNonce(nonce.Hash.CloneBytes(), &nonce.Nonce)
	//route.SetNodeAddrCache(&proto.RawNodeID{Hash: nonce.Hash}, server.Listener.Addr().String())

	client := NewCaller()
	node1 := proto.NewNode()
	node1.InitNodeCryptoInfo(100 * time.Millisecond)
	node1.Addr = "1.1.1.1:1"

	reqA := &proto.PingReq{
		Node: *node1,
	}

	respA := new(proto.PingResp)
	err = client.CallNode(conf.GConf.BP.NodeID, route.DHTPing.String(), reqA, respA)
	if err != nil {
		t.Fatal(err)
	}
	log.Debugf("respA: %v", respA)

	node1addr, err := GetNodeAddr(node1.ID.ToRawNodeID())
	Convey("test GetNodeAddr", t, func() {
		So(err, ShouldBeNil)
		So(node1addr, ShouldEqual, node1.Addr)
	})

	node2, err := GetNodeInfo(node1.ID.ToRawNodeID())
	Convey("test GetNodeInfo", t, func() {
		So(err, ShouldBeNil)
		So(node2.PublicKey.IsEqual(node1.PublicKey), ShouldBeTrue)
		log.Debugf("\nnode1 %##v \nnode2 %##v", node1, node2)
	})

	kms.DelNode(node2.ID)
	node2, err = GetNodeInfo(node1.ID.ToRawNodeID())
	Convey("test GetNodeInfo", t, func() {
		So(err, ShouldBeNil)
		So(node2.PublicKey.IsEqual(node1.PublicKey), ShouldBeTrue)
		log.Debugf("\nnode1 %##v \nnode2 %##v", node1, node2)
	})

	err = PingBP(node1, conf.GConf.BP.NodeID)
	//err = client.CallNode(conf.GConf.BP.NodeID, "DHT.Ping", reqA, respA)
	if err != nil {
		t.Fatal(err)
	}
	//log.Debugf("respA2: %v", respA)

	// call with canceled context
	ctx, contextCancel := context.WithCancel(context.Background())
	contextCancel()
	err = client.CallNodeWithContext(ctx, conf.GConf.BP.NodeID, "DHT.Ping", reqA, respA)
	if err == nil {
		t.Fatal("this call should failed, but actually not")
	} else {
		log.Debugf("err: %v", err)
	}

	// call with empty context
	err = client.CallNodeWithContext(context.Background(), conf.GConf.BP.NodeID, "DHT.Ping", reqA, respA)
	if err != nil {
		t.Fatal(err)
	}
	log.Debugf("respA2: %v", respA)

	// test get current bp, should only be myself
	chiefBPNodeID, err := GetCurrentBP()
	if err != nil {
		t.Fatal(err)
	}
	log.Debugf("current chief bp is: %v", chiefBPNodeID)

	// set another random node as block producer
	randomNode := proto.NodeID("00000000011a34cb8142780f692a4097d883aa2ac8a534a070a134f11bcca573")
	SetCurrentBP(randomNode)
	chiefBPNodeID, err = GetCurrentBP()
	if err != nil {
		t.Fatal(err)
	}
	if chiefBPNodeID != randomNode {
		t.Fatalf("SetCurrentBP does not works, set: %v, current: %v", randomNode, chiefBPNodeID)
	}

	server.Stop()
	client.pool.Close()
}

func TestNewPersistentCaller(t *testing.T) {
	log.SetLevel(log.FatalLevel)
	os.Remove(PubKeyStorePath)
	defer os.Remove(PubKeyStorePath)
	os.Remove(publicKeyStore)
	defer os.Remove(publicKeyStore)

	var d string
	var err error
	if d, err = ioutil.TempDir("", "rpcutil_test_"); err != nil {
		return
	}

	// init conf
	_, testFile, _, _ := runtime.Caller(0)
	dupConfFile := filepath.Join(d, "config.yaml")
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/config.yaml")
	if err = utils.DupConf(confFile, dupConfFile); err != nil {
		return
	}
	privateKeyPath := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/private.key")

	conf.GConf, _ = conf.LoadConfig(dupConfFile)
	log.Debugf("GConf: %#v", conf.GConf)
	// reset the once
	route.Once = sync.Once{}
	route.InitKMS(publicKeyStore)

	addr := conf.GConf.ListenAddr // see ../test/node_standalone/config.yaml
	masterKey := []byte("")
	dht, err := route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)

	server, err := NewServerWithService(ServiceMap{route.DHTRPCName: dht})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err = utils.WaitForPorts(ctx, "127.0.0.1", []int{
		12230,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	err = server.InitRPCServer(addr, privateKeyPath, masterKey)
	if err != nil {
		t.Fatal(err)
	}
	go server.Serve()

	client := NewPersistentCaller(conf.GConf.BP.NodeID)
	node1 := proto.NewNode()
	node1.InitNodeCryptoInfo(100 * time.Millisecond)
	node1.Addr = "1.1.1.1:1"

	reqA := &proto.PingReq{
		Node: *node1,
	}

	respA := new(proto.PingResp)
	err = client.Call("DHT.Ping", reqA, respA)
	if err != nil {
		t.Fatal(err)
	}
	log.Debugf("respA: %v", respA)

	req := &proto.FindNeighborReq{
		ID:    "1234567812345678123456781234567812345678123456781234567812345678",
		Count: 10,
	}
	resp := new(proto.FindNeighborResp)

	err = client.Call("DHT.FindNeighbor", req, resp)
	if err == nil || !strings.Contains(err.Error(), "not permitted") {
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
		} else {
			t.Errorf("unexpected resp %v", resp)
		}
		t.Fatal("anonymous ETLS connection used by " +
			"RPC other than DHTPing should not permitted")
	}

	// close anonymous ETLS connection, and create new one
	client.ResetClient("DHT.FindNeighbor")

	wg := sync.WaitGroup{}
	client = NewPersistentCaller(conf.GConf.BP.NodeID)
	wg.Add(RPCConcurrent * RPCCount)
	for i := 0; i < RPCConcurrent; i++ {
		go func(tt *testing.T, wg *sync.WaitGroup) {
			for j := 0; j < RPCCount; j++ {
				reqF := &proto.FindNeighborReq{
					ID:    "1234567812345678123456781234567812345678123456781234567812345678",
					Count: 10,
				}
				respF := new(proto.FindNeighborResp)
				err := client.Call("DHT.FindNeighbor", reqF, respF)
				if err != nil {
					tt.Error(err)
				}
				log.Debugf("resp: %v", respF)
				wg.Done()
			}
		}(t, &wg)
	}

	client2 := NewPersistentCaller(conf.GConf.BP.NodeID)
	reqF2 := &proto.FindNeighborReq{
		ID:    "1234567812345678123456781234567812345678123456781234567812345678",
		Count: 10,
	}
	respF2 := new(proto.FindNeighborResp)

	err = client2.Call("DHT.FindNeighbor", reqF2, respF2)
	if err != nil {
		t.Error(err)
	}
	client2.CloseStream()

	wg.Wait()
	sess, ok := client2.pool.getSession(conf.GConf.BP.NodeID)
	if !ok {
		t.Fatalf("can not find session for %s", conf.GConf.BP.NodeID)
	}
	sess.Close()

	client3 := NewPersistentCaller(conf.GConf.BP.NodeID)
	err = client3.Call("DHT.FindNeighbor", reqF2, respF2)
	if err != nil {
		t.Error(err)
	}
	client3.CloseStream()

}

func BenchmarkPersistentCaller_CallKayakLog(b *testing.B) {
	log.SetLevel(log.FatalLevel)
	os.Remove(PubKeyStorePath)
	defer os.Remove(PubKeyStorePath)
	os.Remove(publicKeyStore)
	defer os.Remove(publicKeyStore)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err := utils.WaitForPorts(ctx, "127.0.0.1", []int{
		2230,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/config.yaml")
	privateKeyPath := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/private.key")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %#v", conf.GConf)
	// reset the once
	route.Once = sync.Once{}
	route.InitKMS(publicKeyStore)

	addr := conf.GConf.ListenAddr
	_, err = route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)

	server, err := NewServerWithService(ServiceMap{"Test": &fakeService{}})
	if err != nil {
		b.Fatal(err)
	}

	_ = server.InitRPCServer(addr, privateKeyPath, []byte{})
	go server.Serve()

	client := NewPersistentCaller(conf.GConf.BP.NodeID)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := &FakeRequest{}
			req.Log.Data = []byte(strings.Repeat("1", 500))
			err = client.Call("Test.Call", req, nil)
			if err != nil {
				b.Error(err)
			}
		}
	})
	b.StopTimer()
	time.Sleep(5 * time.Second)
	server.Stop()
	GetSessionPoolInstance().Close()
}

type fakeService struct{}

type FakeRequest struct {
	proto.Envelope
	Instance string
	Log      struct {
		Index      uint64       // log index
		Version    uint64       // log version
		Type       uint8        // log type
		Producer   proto.NodeID // producer node
		DataLength uint64       // data length
		Data       []byte
	}
}

func (s *fakeService) Call(req *FakeRequest, resp *interface{}) (err error) {
	time.Sleep(time.Microsecond * 600)
	return
}

func BenchmarkPersistentCaller_Call(b *testing.B) {
	log.SetLevel(log.InfoLevel)
	os.Remove(PubKeyStorePath)
	defer os.Remove(PubKeyStorePath)
	os.Remove(publicKeyStore)
	defer os.Remove(publicKeyStore)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	err := utils.WaitForPorts(ctx, "127.0.0.1", []int{
		2230,
	}, time.Millisecond*200)

	if err != nil {
		log.Fatalf("wait for port ready timeout: %v", err)
	}

	_, testFile, _, _ := runtime.Caller(0)
	confFile := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/config.yaml")
	privateKeyPath := filepath.Join(filepath.Dir(testFile), "../test/node_standalone/private.key")

	conf.GConf, _ = conf.LoadConfig(confFile)
	log.Debugf("GConf: %#v", conf.GConf)
	// reset the once
	route.Once = sync.Once{}
	route.InitKMS(publicKeyStore)

	addr := conf.GConf.ListenAddr // see ../test/node_standalone/config.yaml
	masterKey := []byte("")
	dht, err := route.NewDHTService(PubKeyStorePath, new(consistent.KMSStorage), true)

	server, err := NewServerWithService(ServiceMap{route.DHTRPCName: dht})
	if err != nil {
		b.Fatal(err)
	}

	server.InitRPCServer(addr, privateKeyPath, masterKey)
	go server.Serve()

	client := NewPersistentCaller(conf.GConf.BP.NodeID)
	node1 := proto.NewNode()
	node1.InitNodeCryptoInfo(100 * time.Millisecond)
	node1.Addr = "1.1.1.1:1"

	reqA := &proto.PingReq{
		Node: *node1,
	}

	respA := new(proto.PingResp)
	err = client.Call("DHT.Ping", reqA, respA)
	if err != nil {
		b.Fatal(err)
	}
	log.Debugf("respA: %v", respA)

	client = NewPersistentCaller(conf.GConf.BP.NodeID)
	b.Run("benchmark Persistent Call Nil", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err = client.Call("DHT.Nil", nil, nil)
			if err != nil {
				b.Error(err)
			}
		}
	})

	b.Run("benchmark Persistent Call parallel Nil", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := client.Call("DHT.Nil", nil, nil)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})

	b.Run("benchmark Persistent Call parallel 1k", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := client.Call("DHT.Nil", strings.Repeat("a", 1000), nil)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})

	req := &proto.FindNeighborReq{
		ID:    "1234567812345678123456781234567812345678123456781234567812345678",
		Count: 10,
	}
	resp := new(proto.FindNeighborResp)

	client = NewPersistentCaller(conf.GConf.BP.NodeID)
	b.Run("benchmark Persistent Call", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err = client.Call("DHT.FindNeighbor", req, resp)
			if err != nil {
				b.Error(err)
			}
		}
	})

	routineCount := runtime.NumGoroutine()
	if routineCount > 100 {
		b.Errorf("go routine count: %d", routineCount)
	} else {
		log.Infof("go routine count: %d", routineCount)
	}

	b.Run("benchmark Persistent New and Call", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			client = NewPersistentCaller(conf.GConf.BP.NodeID)
			err = client.Call("DHT.FindNeighbor", req, resp)
			if err != nil {
				b.Error(err)
			}
			client.Close()
		}
	})

	routineCount = runtime.NumGoroutine()
	if routineCount > 100 {
		b.Errorf("go routine count: %d", routineCount)
	} else {
		log.Infof("go routine count: %d", routineCount)
	}

	b.Run("benchmark non-Persistent", func(b *testing.B) {
		oldClient := NewCaller()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err = oldClient.CallNode(conf.GConf.BP.NodeID, "DHT.FindNeighbor", req, resp)
			if err != nil {
				b.Error(err)
			}
		}
	})

	routineCount = runtime.NumGoroutine()
	if routineCount > 100 {
		b.Errorf("go routine count: %d", routineCount)
	} else {
		log.Infof("go routine count: %d", routineCount)
	}

	server.Stop()
}

func TestRecordRPCCost(t *testing.T) {
	Convey("Bug: bad critical section for multiple values", t, func(c C) {
		var (
			start      = time.Now()
			rounds     = 1000
			concurrent = 10
			wg         = &sync.WaitGroup{}
			body       = func(i int) {
				defer func() {
					c.So(recover(), ShouldBeNil)
					wg.Done()
				}()
				recordRPCCost(start, fmt.Sprintf("M%d", i), nil)
			}
		)
		for i := 0; i < rounds; i++ {
			for j := 0; j < concurrent; j++ {
				wg.Add(1)
				go body(i)
			}
			wg.Wait()
		}
	})
}
