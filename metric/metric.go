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
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func init() {
	prometheus.MustRegister(version.NewCollector("CovenantSQL"))
}

// StartMetricCollector starts collector registered in NewNodeCollector().
func StartMetricCollector() (registry *prometheus.Registry) {
	nc, err := NewNodeCollector()
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
