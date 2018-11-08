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

	"github.com/CovenantSQL/CovenantSQL/blockproducer/types"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/rpc"
)

const (
	blockVersion int32 = 0x01
)

// Config is the main chain configuration.
type Config struct {
	Genesis *types.Block

	DataFile string

	Server *rpc.Server

	Peers  *proto.Peers
	NodeID proto.NodeID

	Period time.Duration
	Tick   time.Duration
}

// NewConfig creates new config.
func NewConfig(genesis *types.Block, dataFile string,
	server *rpc.Server, peers *proto.Peers,
	nodeID proto.NodeID, period time.Duration, tick time.Duration) *Config {
	config := Config{
		Genesis:  genesis,
		DataFile: dataFile,
		Server:   server,
		Peers:    peers,
		NodeID:   nodeID,
		Period:   period,
		Tick:     tick,
	}
	return &config
}
