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

package metric

import (
	"bytes"
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	log "github.com/sirupsen/logrus"
	"gitlab.com/thunderdb/ThunderDB/proto"
	"gitlab.com/thunderdb/ThunderDB/rpc"
)

// metricMap is map from metric name to MetricFamily
type metricMap map[string]*dto.MetricFamily

// MetricServiceName is the RPC name
const MetricServiceName = "Metric"

// CollectClient is the Metric Collect Client
type CollectClient struct {
	registry *prometheus.Registry
}

// NewCollectClient returns a new CollectClient
func NewCollectClient() *CollectClient {
	reg := StartMetricCollector()
	if reg == nil {
		log.Fatal("StartMetricCollector failed")
	}

	return &CollectClient{
		registry: reg,
	}
}

// CollectServer is the Metric receiver side
type CollectServer struct {
	NodeMetric sync.Map // map[proto.NodeID]metricMap
}

// NewCollectServer returns a new CollectServer
func NewCollectServer() *CollectServer {
	return &CollectServer{
		NodeMetric: sync.Map{},
	}
}

// UploadMetrics RPC uploads metric info
func (cs *CollectServer) UploadMetrics(req *proto.UploadMetricsReq, resp *proto.UploadMetricsResp) (err error) {
	if req.NodeID == "" {
		err = errors.New("empty node id")
		resp.Msg = "empty node id"
		log.Errorln(resp.Msg)
		return
	}
	mfm := make(metricMap, len(req.MFBytes))
	log.Debugf("RPC received MFS len %d", len(req.MFBytes))
	for _, mf := range req.MFBytes[:] {
		bufReader := bytes.NewReader(mf)
		//mf := new(dto.MetricFamily)
		//dec := expfmt.NewDecoder(bufReader, expfmt.FmtProtoCompact)
		//err = dec.Decode(mf)
		tp := expfmt.TextParser{}
		mf, err := tp.TextToMetricFamilies(bufReader)
		if err != nil {
			log.Warnf("decode MetricFamily failed: %s", err)
			continue
		}
		log.Debugf("RPC received MF: %#v", mf)
		for k, v := range mf {
			mfm[k] = v
		}
	}
	log.Debugf("MetricFamily uploaded: %v", mfm)
	if len(mfm) > 0 {
		cs.NodeMetric.Store(req.NodeID, mfm)
	} else {
		err = errors.New("no valid metric received")
	}
	return
}

// GatherMetricBytes gathers the registered metric info and encode it to [][]byte
func (cc *CollectClient) GatherMetricBytes() (mfb [][]byte, err error) {
	mfs, err := cc.registry.Gather()
	if err != nil {
		log.Errorf("gather metrics failed: %s", err)
		return
	}
	mfb = make([][]byte, 0, len(mfs))
	for _, mf := range mfs[:] {
		log.Debugf("mf: %s", *mf)
		buf := new(bytes.Buffer)
		//enc := expfmt.NewEncoder(buf, expfmt.FmtProtoCompact)
		//err = enc.Encode(mf)
		expfmt.MetricFamilyToText(buf, mf)
		if err != nil {
			log.Warnf("encode MetricFamily failed: %s", err)
			continue
		}
		mfb = append(mfb, buf.Bytes())
	}
	if len(mfb) == 0 {
		err = errors.New("no valid metric gathered")
	}

	return
}

// UploadMetrics calls RPC UploadMetrics to upload its metric info
func (cc *CollectClient) UploadMetrics(BPNodeID proto.NodeID, connPool *rpc.SessionPool) (err error) {
	mfb, err := cc.GatherMetricBytes()
	if err != nil {
		log.Errorf("GatherMetricBytes failed: %s", err)
		return
	}

	conn, err := rpc.DialToNode(BPNodeID, connPool)
	if err != nil {
		log.Errorf("dial to %s failed: %s", BPNodeID, err)
		return
	}
	client, err := rpc.InitClientConn(conn)
	if err != nil {
		log.Errorf("init client conn failed: %s", err)
		return
	}
	log.Debugf("Calling BP: %s", BPNodeID)
	reqType := MetricServiceName + ".UploadMetrics"
	req := &proto.UploadMetricsReq{
		NodeID:  BPNodeID,
		MFBytes: mfb,
	}
	resp := new(proto.UploadMetricsResp)
	log.Debugf("req %s: %v", reqType, req)
	err = client.Call(reqType, req, resp)
	if err != nil {
		log.Errorf("calling RPC %s failed: %s", reqType, err)
	}
	log.Debugf("resp %s: %v", reqType, resp)
	return
}
