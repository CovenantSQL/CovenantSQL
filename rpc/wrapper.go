/*
 * Copyright 2019 The CovenantSQL Authors.
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
	"github.com/CovenantSQL/CovenantSQL/proto"
)

// NewServer return a new Server.
func NewServer() *Server {
	return NewServerWithServeFunc(ServeDirect)
}

// NewServerWithService returns a new Server and registers the Server.ServiceMap.
func NewServerWithService(serviceMap ServiceMap) (server *Server, err error) {
	server = NewServer()
	for k, v := range serviceMap {
		err = server.RegisterService(k, v)
		if err != nil {
			return nil, err
		}
	}
	return server, nil
}

// WithAcceptConnFunc resets the AcceptConn function of server.
func (s *Server) WithAcceptConnFunc(f AcceptConn) *Server {
	s.acceptConn = f
	return s
}

// PCaller defines generic interface shared with PersistentCaller and RawCaller.
type PCaller interface {
	Call(method string, request interface{}, reply interface{}) (err error)
	Close()
	Target() string
	New() PCaller // returns new instance of current caller
}

// NewCaller returns a new RPCCaller.
func NewCaller() *Caller {
	return NewCallerWithPool(defaultPool)
}

// NewPersistentCaller returns a persistent RPCCaller.
//  IMPORTANT: If a PersistentCaller is firstly used by a DHT.Ping, which is an anonymous
//  ETLS connection. It should not be used by any other RPC except DHT.Ping.
func NewPersistentCaller(target proto.NodeID) *PersistentCaller {
	return NewPersistentCallerWithPool(defaultPool, target)
}
