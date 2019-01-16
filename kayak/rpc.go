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
	kt "github.com/CovenantSQL/CovenantSQL/kayak/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/pkg/errors"
)

func (r *Runtime) errorSummary(errs map[proto.NodeID]error) error {
	failNodes := make(map[proto.NodeID]error)

	for s, err := range errs {
		if err != nil {
			failNodes[s] = err
		}
	}

	if len(failNodes) == 0 {
		return nil
	}

	return errors.Wrapf(kt.ErrPrepareFailed, "fail on nodes: %v", failNodes)
}

/// rpc related
func (r *Runtime) applyRPC(l *kt.Log, minCount int) (tracker *rpcTracker) {
	req := &kt.ApplyRequest{
		Instance: r.instanceID,
		Log:      l,
	}

	tracker = newTracker(r, req, minCount)
	tracker.send()

	// TODO(): track this rpc

	// TODO(): log remote errors

	return
}
