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
	server, err := rpc.NewServerWithService(rpc.ServiceMap{route.DHTRPCName: dht})
	if err != nil {
		log.Fatal(err)
	}

	// Init RPC server with an empty master key, which is not recommend
	addr := conf.GConf.ListenAddr
	masterKey := []byte("")
	server.InitRPCServer(addr, conf.GConf.PrivateKeyFile, masterKey)
	server.Serve()
}
