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

package api

import (
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/kayak"
	kt "github.com/CovenantSQL/CovenantSQL/kayak/transport"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/twopc"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	// DefaultProcessTimeout defines package default process timeout.
	DefaultProcessTimeout = time.Second * 5
	// DefaultTransportID defines default transport id for service multiplex.
	DefaultTransportID = "DEFAULT"
)

// TwoPCOptions defines optional arguments for kayak twopc config.
type TwoPCOptions struct {
	ProcessTimeout time.Duration
	NodeID         proto.NodeID
	TransportID    string
	Logger         *log.Logger
}

// NewTwoPCOptions creates empty twopc configuration options.
func NewTwoPCOptions() *TwoPCOptions {
	return &TwoPCOptions{
		ProcessTimeout: DefaultProcessTimeout,
		TransportID:    DefaultTransportID,
	}
}

// NewDefaultTwoPCOptions creates twopc configuration options with default settings.
func NewDefaultTwoPCOptions() *TwoPCOptions {
	nodeID, _ := kms.GetLocalNodeID()
	return NewTwoPCOptions().WithNodeID(nodeID)
}

// WithProcessTimeout set custom process timeout to options.
func (o *TwoPCOptions) WithProcessTimeout(timeout time.Duration) *TwoPCOptions {
	o.ProcessTimeout = timeout
	return o
}

// WithNodeID set custom node id to options.
func (o *TwoPCOptions) WithNodeID(nodeID proto.NodeID) *TwoPCOptions {
	o.NodeID = nodeID
	return o
}

// WithTransportID set custom transport id to options.
func (o *TwoPCOptions) WithTransportID(id string) *TwoPCOptions {
	o.TransportID = id
	return o
}

// WithLogger set custom logger to options.
func (o *TwoPCOptions) WithLogger(l *log.Logger) *TwoPCOptions {
	o.Logger = l
	return o
}

// NewTwoPCKayak creates new kayak runtime.
func NewTwoPCKayak(peers *kayak.Peers, config kayak.Config) (*kayak.Runtime, error) {
	return kayak.NewRuntime(config, peers)
}

// NewTwoPCConfig creates new twopc config object.
func NewTwoPCConfig(rootDir string, service *kt.ETLSTransportService, worker twopc.Worker) kayak.Config {
	return NewTwoPCConfigWithOptions(rootDir, service, worker, NewDefaultTwoPCOptions())
}

// NewTwoPCConfigWithOptions creates new twopc config object with custom options.
func NewTwoPCConfigWithOptions(rootDir string, service *kt.ETLSTransportService,
	worker twopc.Worker, options *TwoPCOptions) kayak.Config {
	runner := kayak.NewTwoPCRunner()
	xptCfg := &kt.ETLSTransportConfig{
		TransportService: service,
		NodeID:           options.NodeID,
		TransportID:      options.TransportID,
		ServiceName:      service.ServiceName,
	}
	xpt := kt.NewETLSTransport(xptCfg)
	cfg := &kayak.TwoPCConfig{
		RuntimeConfig: kayak.RuntimeConfig{
			RootDir:        rootDir,
			LocalID:        options.NodeID,
			Runner:         runner,
			Transport:      xpt,
			ProcessTimeout: options.ProcessTimeout,
		},
		Storage: worker,
	}

	return cfg
}
