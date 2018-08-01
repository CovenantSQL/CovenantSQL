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
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const historyCount = 2

const updateInterval = 5 * time.Minute

// thunderDBStatsMetrics provide description, value, and value type for thunderDB stat metrics.
type thunderDBStatsMetrics []struct {
	desc    *prometheus.Desc
	eval    func(*ThunderDBCollector) float64
	valType prometheus.ValueType
}

// ThunderDBCollector collects ThunderDB metrics
type ThunderDBCollector struct {
	thunderDBStatHistory [historyCount]int64
	sync.RWMutex

	// metrics to describe and collect
	metrics thunderDBStatsMetrics
}

func thunderDBStatNamespace(s string) string {
	return fmt.Sprintf("thunderstats_%s", s)
}

// NewThunderDBCollector returns a new ThunderDBCollector
func NewThunderDBCollector() prometheus.Collector {
	cc := &ThunderDBCollector{
		thunderDBStatHistory: [historyCount]int64{},
		RWMutex:              sync.RWMutex{},
		metrics: thunderDBStatsMetrics{
			{
				desc: prometheus.NewDesc(
					thunderDBStatNamespace("db_random"),
					"ThunderDB random",
					nil,
					nil,
				),
				eval:    ThunderDBIdle,
				valType: prometheus.GaugeValue,
			},
		},
	}

	go func(cc *ThunderDBCollector) {
		for {
			cc.updateThunderDBStat()
			time.Sleep(updateInterval)
		}
	}(cc)
	return cc
}

// Describe returns all descriptions of the collector.
func (cc *ThunderDBCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, i := range cc.metrics {
		ch <- i.desc
	}
}

// Collect returns the current state of all metrics of the collector.
func (cc *ThunderDBCollector) Collect(ch chan<- prometheus.Metric) {
	if !cc.ThunderDBPrepared() {
		return
	}

	for _, i := range cc.metrics {
		ch <- prometheus.MustNewConstMetric(i.desc, i.valType, i.eval(cc))
	}
}

// updateThunderDBStat updates metric in background
func (cc *ThunderDBCollector) updateThunderDBStat() error {
	cc.Lock()
	defer cc.Unlock()
	for i := historyCount - 1; i > 0; i-- {
		cc.thunderDBStatHistory[i] = cc.thunderDBStatHistory[i-1]
	}

	cc.thunderDBStatHistory[0] = time.Now().UnixNano()
	return nil
}

// ThunderDBIdle gets the idle of DB
func ThunderDBIdle(cc *ThunderDBCollector) float64 {
	cc.RLock()
	defer cc.RUnlock()
	//TODO(auxten): implement ThunderDB Idle metric
	return float64(cc.thunderDBStatHistory[0] - cc.thunderDBStatHistory[1])
}

// ThunderDBPrepared returns true when the metric is ready to be collected
func (cc *ThunderDBCollector) ThunderDBPrepared() bool {
	cc.RLock()
	defer cc.RUnlock()
	return cc.thunderDBStatHistory[1] != 0
}
