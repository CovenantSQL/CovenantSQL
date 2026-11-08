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
	"log/slog"
	"runtime"
	"sort"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/node_exporter/collector"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func init() {
	prometheus.MustRegister(version.NewCollector("CovenantSQL"))
}

// StartMetricCollector starts the node_exporter collectors for memory, load and file systems.
func StartMetricCollector() (registry *prometheus.Registry) {
	// node_exporter sets its defaults, such as the procfs path, as kingpin
	// flag defaults, which only a parse applies.
	if _, err := kingpin.CommandLine.Parse(nil); err != nil {
		log.WithError(err).Error("couldn't parse node collector flags")
		return
	}
	nc, err := collector.NewNodeCollector(slog.Default(), "meminfo", "loadavg", "filesystem")
	if err != nil {
		log.WithError(err).Error("couldn't create node collector")
		return
	}

	registry = prometheus.NewRegistry()
	err = registry.Register(nc)
	if err != nil {
		log.WithError(err).Error("couldn't register collector")
		return nil
	}
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "node_cpu_count",
		Help: "Number of logical CPUs usable by the process.",
	}, func() float64 { return float64(runtime.NumCPU()) }))

	log.Info("enabled collectors:")
	var collectors []string
	for n := range nc.Collectors {
		collectors = append(collectors, n)
	}
	sort.Strings(collectors)
	for _, n := range collectors {
		log.Infof(" - %s", n)
	}

	return
}
