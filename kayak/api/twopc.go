/*
 * Copyright 2018 The ThunderDB Authors.
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
	"context"

	"gitlab.com/thunderdb/ThunderDB/kayak"
	kt "gitlab.com/thunderdb/ThunderDB/kayak/transport"
	"gitlab.com/thunderdb/ThunderDB/proto"
	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/twopc"
	"gitlab.com/thunderdb/ThunderDB/rpc"
	"gitlab.com/thunderdb/ThunderDB/crypto/etls"
	"gitlab.com/thunderdb/ThunderDB/crypto/kms"
	"gitlab.com/thunderdb/ThunderDB/crypto/hash"
)

var (
	DefaultProcessTimeout = time.Second * 5
	DefaultTransportID    = "DEFAULT"
	DefaultLogger         = log.New()
)

type TwoPCOptions struct {
	ProcessTimeout time.Duration
	NodeID         proto.NodeID
	TransportID    string
	ClientBuilder  kt.ETLSRPCClientBuilder
	Logger         *log.Logger
}

func DefaultClientBuilder(ctx context.Context, nodeID proto.NodeID) (client *rpc.Client, err error) {
	var conn *etls.CryptoConn
	conn, err = rpc.DialToNode(nodeID)
	if err != nil {
		return
	}

	client, err = rpc.InitClientConn(conn)

	return
}

func NewTwoPCOptions() *TwoPCOptions {
	return &TwoPCOptions{
		ProcessTimeout: DefaultProcessTimeout,
		TransportID:    DefaultTransportID,
		ClientBuilder:  DefaultClientBuilder,
		Logger:         DefaultLogger,
	}
}

func NewDefaultTwoPCOptions() *TwoPCOptions {
	// TODO(xq262144), node id hack
	rawNodeID, _ := kms.GetLocalNodeID()
	rawNodeHash, _ := hash.NewHash(rawNodeID)
	nodeID := proto.NodeID(rawNodeHash.String())
	return NewTwoPCOptions().WithNodeID(nodeID)
}

func (o *TwoPCOptions) WithProcessTimeout(timeout time.Duration) *TwoPCOptions {
	o.ProcessTimeout = timeout
	return o
}

func (o *TwoPCOptions) WithNodeID(nodeID proto.NodeID) *TwoPCOptions {
	o.NodeID = nodeID
	return o
}

func (o *TwoPCOptions) WithTransportID(id string) *TwoPCOptions {
	o.TransportID = id
	return o
}

func (o *TwoPCOptions) WithClientBuilder(builder kt.ETLSRPCClientBuilder) *TwoPCOptions {
	o.ClientBuilder = builder
	return o
}

func (o *TwoPCOptions) WithLogger(l *log.Logger) *TwoPCOptions {
	o.Logger = l
	return o
}

func NewTwoPCKayak(peers *kayak.Peers, config kayak.Config) (*kayak.Runtime, error) {
	return kayak.NewRuntime(config, peers)
}

func NewTwoPCConfig(rootDir string, service *kt.ETLSTransportService, worker twopc.Worker) kayak.Config {
	return NewTwoPCConfigWithOptions(rootDir, service, worker, NewDefaultTwoPCOptions())
}

func NewTwoPCConfigWithOptions(rootDir string, service *kt.ETLSTransportService,
	worker twopc.Worker, options *TwoPCOptions) kayak.Config {
	runner := kayak.NewTwoPCRunner()
	xptCfg := &kt.ETLSTransportConfig{
		TransportService: service,
		NodeID:           options.NodeID,
		TransportID:      options.TransportID,
		ServiceName:      service.ServiceName,
		ClientBuilder:    options.ClientBuilder,
	}
	xpt := kt.NewETLSTransport(xptCfg)
	cfg := &kayak.TwoPCConfig{
		RuntimeConfig: kayak.RuntimeConfig{
			RootDir:        rootDir,
			LocalID:        options.NodeID,
			Runner:         runner,
			Transport:      xpt,
			ProcessTimeout: options.ProcessTimeout,
			Logger:         options.Logger,
		},
		Storage: worker,
	}

	return cfg
}
