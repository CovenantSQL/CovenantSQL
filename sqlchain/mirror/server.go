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

package mirror

import (
	"net"

	"github.com/pkg/errors"

	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/rpc/mux"
)

func createServer(listenAddr string) (s *mux.Server, err error) {
	var l net.Listener
	if l, err = net.Listen("tcp", listenAddr); err != nil {
		err = errors.Wrap(err, "listen rpc server failed")
		return
	}

	s = mux.NewServer().WithAcceptConnFunc(rpc.AcceptRawConn)
	s.SetListener(l)

	return
}

// StartMirror starts the mirror server and start mirror database.
func StartMirror(database string, listenAddr string) (service *Service, err error) {
	var server *mux.Server
	if server, err = createServer(listenAddr); err != nil {
		return
	}

	if service, err = NewService(database, server); err != nil {
		return
	}

	// start mirror
	err = service.start()

	return
}

// StopMirror stops the mirror server.
func StopMirror(service *Service) {
	service.stop()
}
