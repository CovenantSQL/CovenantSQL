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
	"github.com/CovenantSQL/CovenantSQL/kayak"
	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/rpc"
)

// KayakService defines the leader service kayak.
type KayakService struct {
	serviceName string
	rt          *kayak.Runtime
}

// NewKayakService returns new kayak service instance for block producer consensus.
func NewKayakService(server *rpc.Server, serviceName string, rt *kayak.Runtime) (s *KayakService, err error) {
	s = &KayakService{
		serviceName: serviceName,
		rt:          rt,
	}
	err = server.RegisterService(serviceName, s)
	return
}

// Apply handles kayak apply call.
func (s *KayakService) Apply(req *kt.ApplyRequest, _ *interface{}) (err error) {
	return s.rt.FollowerApply(req.Log)
}

// Fetch handles kayak log fetch call.
func (s *KayakService) Fetch(req *kt.FetchRequest, resp *kt.FetchResponse) (err error) {
	var l *kt.Log
	if l, err = s.rt.Fetch(req.GetContext(), req.Index); err != nil {
		return
	}

	resp.Log = l
	return
}
