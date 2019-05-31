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
	"testing"

	dto "github.com/prometheus/client_model/go"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func TestCollectServer_FilterNode(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	filterTrue := func(key proto.NodeID, value SimpleMetricMap) bool {
		log.Debugf("key: %s, value: %#v", key, value)
		return true
	}
	filterFalse := func(key proto.NodeID, value SimpleMetricMap) bool {
		log.Debugf("key: %s, value: %#v", key, value)
		return false
	}
	filterMem1MB := func(key proto.NodeID, value SimpleMetricMap) bool {
		log.Debugf("key: %s, value: %#v", key, value)
		var v *dto.MetricFamily
		v, ok := value["node_memory_bytes_total"]
		if !ok {
			v, ok = value["node_memory_MemTotal_bytes"]
		}
		if ok && len(v.Metric) > 0 &&
			v.Metric[0].GetGauge() != nil &&
			v.Metric[0].GetGauge().Value != nil &&
			*v.Metric[0].GetGauge().Value > float64(1*utils.MB) {
			log.Debugf("has memory: %fGB", *v.Metric[0].GetGauge().Value/float64(utils.GB))
			return true
		}

		return false
	}
	Convey("filter node", t, func() {
		cc := NewCollectClient()
		mfs, _ := cc.Registry.Gather()
		mm := make(SimpleMetricMap, 0)
		for _, mf := range mfs {
			mm[*mf.Name] = mf
			log.Debugf("gathered node: %v", mf)
		}
		nmm := NodeMetricMap{}
		nmm.Store(proto.NodeID("node1"), mm)
		nmm.Store(proto.NodeID("node2"), nil)
		nmm.Store(proto.NodeID("node3"), mm)
		So(len(mm), ShouldEqual, len(mfs))
		So(len(mm), ShouldBeGreaterThan, 2)

		ids := nmm.FilterNode(filterTrue)
		So(len(ids), ShouldEqual, 2)

		ids1 := nmm.FilterNode(filterMem1MB)
		So(len(ids1), ShouldEqual, 2)

		ids2 := nmm.FilterNode(filterFalse)
		So(len(ids2), ShouldEqual, 0)
	})
	Convey("filter metrics", t, func() {
		cc := NewCollectClient()
		mfs, _ := cc.Registry.Gather()
		mm := make(SimpleMetricMap, 0)
		for _, mf := range mfs {
			mm[*mf.Name] = mf
			log.Debugf("gathered node: %v", mf)
		}
		nmm := NodeMetricMap{}
		nmm.Store(proto.NodeID("node1"), mm)
		nmm.Store(proto.NodeID("node2"), nil)

		cmm := nmm.GetCrucialMetrics()
		So(len(cmm), ShouldEqual, 1)
		So(len(cmm["node1"]), ShouldBeGreaterThanOrEqualTo, 6)
	})

}
