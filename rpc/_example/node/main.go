/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// TestService to be register to RPC server
type TestService struct {
}

func NewTestService() *TestService {
	return &TestService{}
}

func (s *TestService) Talk(msg string, ret *string) error {
	fmt.Println(msg)
	resp := fmt.Sprintf("got %s", msg)
	*ret = resp
	return nil
}

func main() {
	//log.SetLevel(log.DebugLevel)
	conf.GConf, _ = conf.LoadConfig(os.Args[1])
	log.Debugf("GConf: %#v", conf.GConf)

	// Init Key Management System
	route.InitKMS(conf.GConf.PubKeyStoreFile)

	// Register DHT service
	server, err := rpc.NewServerWithService(rpc.ServiceMap{
		"Test": NewTestService(),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Init RPC server with an empty master key, which is not recommend
	addr := conf.GConf.ListenAddr
	masterKey := []byte("")
	server.InitRPCServer(addr, conf.GConf.PrivateKeyFile, masterKey)

	// Start Node RPC server
	go server.Serve()

	pClient := rpc.NewPersistentCaller(conf.GConf.BP.NodeID)
	// Register Node public key, addr to Tracker(BP)
	for _, n := range conf.GConf.KnownNodes {
		reqA := &proto.PingReq{
			Node: n,
		}
		respA := new(proto.PingResp)
		err = pClient.Call("DHT.Ping", reqA, respA)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("respA: %v", respA)
	}

	// Read target node and connect to it
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Input target node ID: ")
	scanner.Scan()
	if scanner.Err() == nil {
		target := proto.NodeID(strings.TrimSpace(scanner.Text()))
		pc := rpc.NewPersistentCaller(target)
		log.Debugf("connecting to %s", scanner.Text())

		fmt.Print("Input msg: ")
		for scanner.Scan() {
			input := scanner.Text()
			log.Debugf("get input %s", input)
			repSimple := new(string)
			err = pc.Call("Test.Talk", input, repSimple)
			if err != nil {
				log.Fatal(err)
			}
			log.Infof("resp msg: %s", *repSimple)
		}
	}
}
