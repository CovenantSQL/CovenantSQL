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
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const historyCount = 2

const updateInterval = 5 * time.Minute

// covenantSQLStatsMetrics provide description, value, and value type for CovenantSQL stat metrics.
type covenantSQLStatsMetrics []struct {
	desc    *prometheus.Desc
	eval    func(*CovenantSQLCollector) float64
	valType prometheus.ValueType
}

// CovenantSQLCollector collects CovenantSQL metrics
type CovenantSQLCollector struct {
	covenantSQLStatHistory [historyCount]int64
	sync.RWMutex

	// metrics to describe and collect
	metrics covenantSQLStatsMetrics
}

func covenantSQLStatNamespace(s string) string {
	return fmt.Sprintf("covenantsqlstats_%s", s)
}

// NewCovenantSQLCollector returns a new CovenantSQLCollector
func NewCovenantSQLCollector() prometheus.Collector {
	cc := &CovenantSQLCollector{
		covenantSQLStatHistory: [historyCount]int64{},
		RWMutex:                sync.RWMutex{},
		metrics: covenantSQLStatsMetrics{
			{
				desc: prometheus.NewDesc(
					covenantSQLStatNamespace("db_random"),
					"CovenantSQL random",
					nil,
					nil,
				),
				eval:    CovenantSQLIdle,
				valType: prometheus.GaugeValue,
			},
		},
	}

	go func(cc *CovenantSQLCollector) {
		for {
			cc.updateCovenantSQLStat()
			time.Sleep(updateInterval)
		}
	}(cc)
	return cc
}

// Describe returns all descriptions of the collector.
func (cc *CovenantSQLCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, i := range cc.metrics {
		ch <- i.desc
	}
}

// Collect returns the current state of all metrics of the collector.
func (cc *CovenantSQLCollector) Collect(ch chan<- prometheus.Metric) {
	if !cc.CovenantSQLPrepared() {
		return
	}

	for _, i := range cc.metrics {
		ch <- prometheus.MustNewConstMetric(i.desc, i.valType, i.eval(cc))
	}
}

// updateCovenantSQLStat updates metric in background
func (cc *CovenantSQLCollector) updateCovenantSQLStat() error {
	cc.Lock()
	defer cc.Unlock()
	for i := historyCount - 1; i > 0; i-- {
		cc.covenantSQLStatHistory[i] = cc.covenantSQLStatHistory[i-1]
	}

	cc.covenantSQLStatHistory[0] = time.Now().UnixNano()
	return nil
}

// CovenantSQLIdle gets the idle of DB
func CovenantSQLIdle(cc *CovenantSQLCollector) float64 {
	cc.RLock()
	defer cc.RUnlock()
	//TODO(auxten): implement CovenantSQL Idle metric
	return float64(cc.covenantSQLStatHistory[0] - cc.covenantSQLStatHistory[1])
}

// CovenantSQLPrepared returns true when the metric is ready to be collected
func (cc *CovenantSQLCollector) CovenantSQLPrepared() bool {
	cc.RLock()
	defer cc.RUnlock()
	return cc.covenantSQLStatHistory[1] != 0
}
