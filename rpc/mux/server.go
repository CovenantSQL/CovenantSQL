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

package mux

import (
	rrpc "github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// Server is the RPC server struct.
type Server struct {
	*rrpc.Server
}

// NewServer return a new Server.
func NewServer() *Server {
	return &Server{
		Server: rrpc.NewServer(ServeMux),
	}
}

func NewServerWithService(serviceMap rrpc.ServiceMap) (sv *Server, err error) {
	server := rrpc.NewServer(ServeMux)
	for k, v := range serviceMap {
		err = server.RegisterService(k, v)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
	}
	return &Server{
		Server: server,
	}, nil
}
