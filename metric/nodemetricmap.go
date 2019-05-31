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

package metric

import (
	"sync"

	dto "github.com/prometheus/client_model/go"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// SimpleMetricMap is map from metric name to MetricFamily.
type SimpleMetricMap map[string]*dto.MetricFamily

// NodeCrucialMetricMap is map[NodeID][MetricName]Value.
type NodeCrucialMetricMap map[proto.NodeID]map[string]float64

// FilterFunc is a function that knows how to filter a specific node
// that match the metric limits. if node picked return true else false.
type FilterFunc func(key proto.NodeID, value SimpleMetricMap) bool

// NodeMetricMap is sync.Map version of map[proto.NodeID]SimpleMetricMap.
type NodeMetricMap struct {
	sync.Map // map[proto.NodeID]SimpleMetricMap
}

// FilterNode return node id slice make filterFunc return true.
func (nmm *NodeMetricMap) FilterNode(filterFunc FilterFunc) (ret []proto.NodeID) {
	nodePicker := func(key, value interface{}) bool {
		id, ok := key.(proto.NodeID)
		if !ok {
			return true // continue iteration
		}
		metrics, ok := value.(SimpleMetricMap)
		if !ok {
			return true // continue iteration
		}
		if filterFunc(id, metrics) {
			ret = append(ret, id)
		}
		return true
	}
	nmm.Range(nodePicker)
	return
}

// GetMetrics returns nodes metrics.
func (nmm *NodeMetricMap) GetMetrics(nodes []proto.NodeID) (metrics map[proto.NodeID]SimpleMetricMap) {
	metrics = make(map[proto.NodeID]SimpleMetricMap)

	for _, node := range nodes {
		var ok bool
		var rawNodeMetrics interface{}

		if rawNodeMetrics, ok = nmm.Load(node); !ok {
			continue
		}

		var nodeMetrics SimpleMetricMap

		if nodeMetrics, ok = rawNodeMetrics.(SimpleMetricMap); !ok {
			continue
		}

		metrics[node] = nodeMetrics
	}

	return
}

// FilterCrucialMetrics filters crucial metrics and also add cpu_count.
func (mfm *SimpleMetricMap) FilterCrucialMetrics() (ret map[string]float64) {
	crucialMetricNameMap := map[string]string{
		"node_memory_MemAvailable_bytes": "mem_avail",
		"node_load1":                     "load1",
		"node_load5":                     "load5",
		"node_load15":                    "load15",
		"node_ntp_offset_seconds":        "ntp_offset",
		"node_filesystem_free_bytes":     "fs_avail",
		"node_cpu_count":                 "cpu_count",
	}
	ret = make(map[string]float64)
	for _, v := range *mfm {
		if newName, ok := crucialMetricNameMap[*v.Name]; ok {
			var metricVal float64
			switch v.GetType() {
			case dto.MetricType_GAUGE:
				metricVal = v.GetMetric()[0].GetGauge().GetValue()
			case dto.MetricType_COUNTER:
				metricVal = v.GetMetric()[0].GetCounter().GetValue()
			case dto.MetricType_HISTOGRAM:
				metricVal = v.GetMetric()[0].GetHistogram().GetBucket()[0].GetUpperBound()
			case dto.MetricType_SUMMARY:
				metricVal = v.GetMetric()[0].GetSummary().GetQuantile()[0].GetValue()
			case dto.MetricType_UNTYPED:
				metricVal = v.GetMetric()[0].GetUntyped().GetValue()
			default:
				continue
			}
			ret[newName] = metricVal
		}
	}
	log.Debugf("crucial Metric added: %v", ret)

	return
}

// GetCrucialMetrics gets NodeCrucialMetricMap from NodeMetricMap.
func (nmm *NodeMetricMap) GetCrucialMetrics() (ret NodeCrucialMetricMap) {
	ret = make(NodeCrucialMetricMap)
	metricsPicker := func(key, value interface{}) bool {
		nodeID, ok := key.(proto.NodeID)
		if !ok {
			return true // continue iteration
		}
		mfm, ok := value.(SimpleMetricMap)
		if !ok {
			return true // continue iteration
		}

		ret[nodeID] = mfm.FilterCrucialMetrics()
		return true // continue iteration
	}
	nmm.Range(metricsPicker)

	return
}
