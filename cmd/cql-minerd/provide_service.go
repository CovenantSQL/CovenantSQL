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
	"runtime"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/metric"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	rpc "github.com/CovenantSQL/CovenantSQL/rpc/mux"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	defaultGasPrice = 1
)

func sendProvideService() {
	var (
		nodeID     proto.NodeID
		privateKey *asymmetric.PrivateKey
		err        error
		minerAddr  proto.AccountAddress
	)

	if nodeID, err = kms.GetLocalNodeID(); err != nil {
		log.WithError(err).Error("get local node id failed")
		return
	}

	if privateKey, err = kms.GetLocalPrivateKey(); err != nil {
		log.WithError(err).Error("get local private key failed")
		return
	}

	if minerAddr, err = crypto.PubKeyHash(privateKey.PubKey()); err != nil {
		log.WithError(err).Error("get miner account address failed")
		return
	}

	host := metric.ReadHost(conf.GConf.Miner.RootDir)
	loadAvg := host.Load15 / float64(runtime.NumCPU())

	log.WithFields(log.Fields{
		"memory":  host.MemAvailable,
		"loadAvg": loadAvg,
		"space":   host.DiskAvailable,
	}).Info("sending provide service transaction with resource parameters")

	var (
		nonceReq  = new(types.NextAccountNonceReq)
		nonceResp = new(types.NextAccountNonceResp)
		req       = new(types.AddTxReq)
		resp      = new(types.AddTxResp)
	)

	nonceReq.Addr = minerAddr

	if err = rpc.RequestBP(route.MCCNextAccountNonce.String(), nonceReq, nonceResp); err != nil {
		// allocate nonce failed
		log.WithError(err).Error("allocate nonce for transaction failed")
		return
	}

	tx := types.NewProvideService(
		&types.ProvideServiceHeader{
			Space:         host.DiskAvailable,
			Memory:        host.MemAvailable,
			LoadAvgPerCPU: loadAvg,
			GasPrice:      defaultGasPrice,
			TokenType:     types.Particle,
			NodeID:        nodeID,
		},
	)

	if conf.GConf.Miner != nil && len(conf.GConf.Miner.TargetUsers) > 0 {
		tx.ProvideServiceHeader.TargetUser = conf.GConf.Miner.TargetUsers
	}

	tx.Nonce = nonceResp.Nonce

	if err = tx.Sign(privateKey); err != nil {
		log.WithError(err).Error("sign provide service transaction failed")
		return
	}

	req.TTL = 1
	req.Tx = tx

	if err = rpc.RequestBP(route.MCCAddTx.String(), req, resp); err != nil {
		// add transaction failed
		log.WithError(err).Error("send provide service transaction failed")
		return
	}
}
