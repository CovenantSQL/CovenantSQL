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

package kayak

import (
	"encoding/binary"

	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
)

func (r *Runtime) getCaller(id proto.NodeID) Caller {
	var caller Caller = rpc.NewPersistentCaller(id)
	rawCaller, _ := r.callerMap.LoadOrStore(id, caller)
	return rawCaller.(Caller)
}

func (r *Runtime) goFunc(f func()) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		f()
	}()
}

/// utils
func (r *Runtime) uint64ToBytes(i uint64) (res []byte) {
	res = make([]byte, 8)
	binary.BigEndian.PutUint64(res, i)
	return
}

func (r *Runtime) bytesToUint64(b []byte) (uint64, error) {
	if len(b) < 8 {
		return 0, kt.ErrInvalidLog
	}
	return binary.BigEndian.Uint64(b), nil
}
