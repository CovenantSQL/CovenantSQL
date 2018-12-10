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

package worker

import (
	"sync"

	"github.com/CovenantSQL/CovenantSQL/kayak"
	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/pkg/errors"
)

const (
	// DBKayakMethodName defines the database kayak rpc method name.
	DBKayakMethodName = "Call"
)

// DBKayakMuxService defines a mux service for sqlchain kayak.
type DBKayakMuxService struct {
	serviceName string
	serviceMap  sync.Map
}

// NewDBKayakMuxService returns a new kayak mux service.
func NewDBKayakMuxService(serviceName string, server *rpc.Server) (s *DBKayakMuxService, err error) {
	s = &DBKayakMuxService{
		serviceName: serviceName,
	}
	err = server.RegisterService(serviceName, s)
	return
}

func (s *DBKayakMuxService) register(id proto.DatabaseID, rt *kayak.Runtime) {
	s.serviceMap.Store(id, rt)

}

func (s *DBKayakMuxService) unregister(id proto.DatabaseID) {
	s.serviceMap.Delete(id)
}

// Call handles kayak call.
func (s *DBKayakMuxService) Call(req *kt.ApplyRequest, _ *interface{}) (err error) {
	// call apply to specified kayak
	// treat req.Instance as DatabaseID
	id := proto.DatabaseID(req.Instance)

	if v, ok := s.serviceMap.Load(id); ok {
		return v.(*kayak.Runtime).FollowerApply(req.Log)
	}

	return errors.Wrapf(ErrUnknownMuxRequest, "instance %v", req.Instance)
}
