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

package kayak

import (
	"github.com/CovenantSQL/CovenantSQL/proto"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
)

// Caller defines the rpc caller, supports mocks for the default rpc.PersistCaller.
type Caller interface {
	Call(method string, req interface{}, resp interface{}) error
}

// NewCallerFunc defines the function type to return a Caller object.
type NewCallerFunc func(target proto.NodeID) Caller

var defaultNewCallerFunc = func(target proto.NodeID) Caller {
	return rpc.NewPersistentCaller(target)
}
