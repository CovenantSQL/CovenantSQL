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

package observer

import (
	"net/http"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
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

func startService() (service *Service, err error) {
	// register observer service to rpc server
	service, err = NewService()
	if err != nil {
		return
	}

	// start observer service
	service.start()

	return
}

func stopService(service *Service) (err error) {
	// stop subscription
	return service.stop()
}

// StartObserver starts the observer service and http API server.
func StartObserver(listenAddr string, version string) (service *Service, httpServer *http.Server, err error) {
	// init node
	if err = initNode(); err != nil {
		log.WithError(err).Fatal("init node failed")
	}

	// start service
	if service, err = startService(); err != nil {
		log.WithError(err).Fatal("start observation failed")
	}

	// start explorer api
	httpServer, err = startAPI(service, listenAddr, version)
	if err != nil {
		log.WithError(err).Fatal("start explorer api failed")
	}

	// register node
	if err = registerNode(); err != nil {
		log.WithError(err).Fatal("register node failed")
	}
	return
}

// StopObserver stops the service and http API server returned by StartObserver.
func StopObserver(service *Service, httpServer *http.Server) (err error) {
	// stop explorer api
	if err = stopAPI(httpServer); err != nil {
		log.WithError(err).Fatal("stop explorer api failed")
	}

	// stop subscriptions
	if err = stopService(service); err != nil {
		log.WithError(err).Fatal("stop service failed")
	}
	return
}
