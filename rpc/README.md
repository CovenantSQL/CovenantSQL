[![Go Report Card](https://goreportcard.com/badge/github.com/CovenantSQL/CovenantSQL?style=flat-square)](https://goreportcard.com/report/github.com/CovenantSQL/CovenantSQL)
[![Coverage](https://codecov.io/gh/CovenantSQL/CovenantSQL/branch/develop/graph/badge.svg)](https://codecov.io/gh/CovenantSQL/CovenantSQL/tree/develop/rpc)
[![Build Status](https://travis-ci.org/CovenantSQL/CovenantSQL.png?branch=develop)](https://travis-ci.org/CovenantSQL/CovenantSQL)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/CovenantSQL/CovenantSQL)


# DH-RPC

DH-RPC is a secp256k1-ECDH-AES encrypted P2P RPC framework for decentralized applications written in golang.

CovenantSQL is built on DH-RPC, including:

- Byzantine Fault Tolerance consensus protocol [Kayak](https://godoc.org/github.com/CovenantSQL/CovenantSQL/kayak)
- Consistent Secure DHT
- DB API
- Metric Collect
- Blocks sync

<img align="middle" src="https://github.com/CovenantSQL/research/raw/master/images/rpc-small.gif">


**Demo Code see [Example](#example)**

## Features

- 75%+ code testing coverage.
- 100% compatible with Go [net/rpc](https://golang.org/pkg/net/rpc/) standard.
- ID based routing and Key exchange built on Secure Enhanced DHT.
- use MessagePack for serialization which support most types without writing `Marshal` and `Unmarshal`.
- Crypto Schema
    - Use Elliptic Curve Secp256k1 for Asymmetric Encryption
    - ECDH for Key Exchange
    - PKCS#7 for padding
    - AES-256-CBC for Symmetric Encryption
    - Private key protected by master key
    - Annoymous connection is also supported
- DHT persistence layer has 2 implementations:
    - BoltDB based simple traditional DHT
    - Kayak based 2PC strong consistent DHT
- Connection pool based on [Yamux](https://github.com/hashicorp/yamux), make thousands of connections multiplexed over **One TCP connection**.
    
## Main Sequence

<img src="../logo/rpc.png" width=600>

## Example

Complete code can be found [here](_example/)

#### Tracker

```golang
package main

import (
	"os"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/consistent"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func main() {
	//log.SetLevel(log.DebugLevel)
	conf.GConf, _ = conf.LoadConfig(os.Args[1])
	log.Debugf("GConf: %#v", conf.GConf)

	// Init Key Management System
	route.InitKMS(conf.GConf.PubKeyStoreFile)

	// Creating DHT RPC with simple BoltDB persistence layer
	dht, err := route.NewDHTService(conf.GConf.DHTFileName, new(consistent.KMSStorage), true)
	if err != nil {
		log.Fatalf("init dht failed: %v", err)
	}

	// Register DHT service
	server, err := rpc.NewServerWithService(rpc.ServiceMap{"DHT": dht})
	if err != nil {
		log.Fatal(err)
	}

	// Init RPC server with an empty master key, which is not recommend
	addr := conf.GConf.ListenAddr
	masterKey := []byte("")
	server.InitRPCServer(addr, conf.GConf.PrivateKeyFile, masterKey)
	server.Serve()
}
```

#### Node
```go
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

	// Register Node public key, addr to Tracker(BP)
	for _, n := range conf.GConf.KnownNodes {
		client := rpc.NewCaller()
		reqA := &proto.PingReq{
			Node: n,
		}
		respA := new(proto.PingResp)
		err = client.CallNode(conf.GConf.BP.NodeID, "DHT.Ping", reqA, respA)
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
```

Start
```bash
$ ./runTracker.sh &
$ ./runNode2.sh &
$ ./runNode1.sh
$ Input target node ID: 000005aa62048f85da4ae9698ed59c14ec0d48a88a07c15a32265634e7e64ade #node2
$ Input msg: abcdefg
```

