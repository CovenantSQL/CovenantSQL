/*
 * Copyright 2026 The CovenantSQL Authors.
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

package blockproducer

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/types"
)

func TestAdviseNewBlock(t *testing.T) {
	Convey("A nil block should not reach the block processing of the chain", t, func() {
		var (
			blocks = make(chan *types.BPBlock, 1)
			s      = &ChainRPCService{chain: &Chain{pendingBlocks: blocks}}
		)
		err := s.AdviseNewBlock(&types.AdviseNewBlockReq{}, &types.AdviseNewBlockResp{})
		So(err, ShouldEqual, ErrNilBlock)
		So(blocks, ShouldBeEmpty)
	})
}
