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

package main

import (
	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	ct "github.com/CovenantSQL/CovenantSQL/sqlchain/types"
)

func registerNode() (err error) {
	var nodeID proto.NodeID

	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		return
	}

	var nodeInfo *proto.Node
	if nodeInfo, err = kms.GetNodeInfo(nodeID); err != nil {
		return
	}

	err = rpc.PingBP(nodeInfo, conf.GConf.BP.NodeID)

	return
}

func startService(server *rpc.Server) (service *Service, err error) {
	// register observer service to rpc server
	service, err = NewService()
	if err != nil {
		return
	}

	if err = server.RegisterService(ct.ObserverService, service); err != nil {
		return
	}

	// start service rpc, observer acts as client role but listen to
	go server.Serve()

	// start observer service
	service.start()

	return
}

func stopService(service *Service, server *rpc.Server) (err error) {
	// stop subscription
	service.stop()

	// stop rpc service
	server.Listener.Close()
	server.Stop()

	return
}
