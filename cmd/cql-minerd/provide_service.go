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
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/crypto"
	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/route"
	"github.com/CovenantSQL/CovenantSQL/rpc"
	"github.com/CovenantSQL/CovenantSQL/types"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	defaultGasPrice = 1
	maxUint64       = 1<<64 - 1
)

var (
	metricKeyMemory   = "node_memory_MemAvailable_bytes"
	metricKeyLoadAvg  = "node_load15"
	metricKeyCPUCount = "node_cpu_count"
	metricKeySpace    = "node_filesystem_free_bytes"
)

func sendProvideService(reg *prometheus.Registry) {
	var (
		memoryBytes uint64
		cpuCount    float64
		loadAvg     float64
		keySpace    uint64
		nodeID      proto.NodeID
		privateKey  *asymmetric.PrivateKey
		mf          []*dto.MetricFamily
		err         error
		minerAddr   proto.AccountAddress
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

	if mf, err = reg.Gather(); err != nil {
		log.WithError(err).Error("gathering node metrics failed")
		return
	}

	for _, m := range mf {
		switch m.GetName() {
		case metricKeyMemory, metricKeyCPUCount, metricKeyLoadAvg, metricKeySpace:
		default:
			continue
		}

		var metricVal float64

		switch m.GetType() {
		case dto.MetricType_GAUGE:
			metricVal = m.GetMetric()[0].GetGauge().GetValue()
		case dto.MetricType_COUNTER:
			metricVal = m.GetMetric()[0].GetCounter().GetValue()
		case dto.MetricType_HISTOGRAM:
			metricVal = m.GetMetric()[0].GetHistogram().GetBucket()[0].GetUpperBound()
		case dto.MetricType_SUMMARY:
			metricVal = m.GetMetric()[0].GetSummary().GetQuantile()[0].GetValue()
		case dto.MetricType_UNTYPED:
			metricVal = m.GetMetric()[0].GetUntyped().GetValue()
		default:
			continue
		}

		switch m.GetName() {
		case metricKeyMemory:
			if metricVal > 0 && metricVal < maxUint64 {
				memoryBytes = uint64(metricVal)
			}
		case metricKeySpace:
			if metricVal > 0 && metricVal < maxUint64 {
				keySpace = uint64(metricVal)
			}
		case metricKeyCPUCount:
			cpuCount = metricVal
		case metricKeyLoadAvg:
			loadAvg = metricVal
		default:
		}
	}

	if cpuCount > 0 {
		loadAvg = loadAvg / cpuCount
	}

	log.WithFields(log.Fields{
		"memory":  memoryBytes,
		"loadAvg": loadAvg,
		"space":   keySpace,
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
			Space:         keySpace,
			Memory:        memoryBytes,
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
