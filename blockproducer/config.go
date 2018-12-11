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

package blockproducer

import (
	"time"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
)

const (
	blockVersion int32 = 0x01
)

// Config is the main chain configuration.
type Config struct {
	Genesis *types.BPBlock

	DataFile string

	Server *rpc.Server

	Peers            *proto.Peers
	NodeID           proto.NodeID
	ConfirmThreshold float64

	Period time.Duration
	Tick   time.Duration

	QPS uint32
}

// NewConfig creates new config.
func NewConfig(genesis *types.BPBlock, dataFile string,
	server *rpc.Server, peers *proto.Peers,
	nodeID proto.NodeID, period time.Duration, tick time.Duration, qps uint32) *Config {
	config := Config{
		Genesis:  genesis,
		DataFile: dataFile,
		Server:   server,
		Peers:    peers,
		NodeID:   nodeID,
		Period:   period,
		Tick:     tick,
		QPS:      qps,
	}
	return &config
}
