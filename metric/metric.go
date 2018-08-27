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
	"sort"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

const (
	// KB is 1024 Bytes
	KB int64 = 1024
	// MB is 1024 KB
	MB int64 = KB * 1024
	// GB is 1024 MB
	GB int64 = MB * 1024
	// TB is 1024 GB
	TB int64 = GB * 1024
	// PB is 1024 TB
	PB int64 = TB * 1024
	// EB is 1024 PB
	EB int64 = TB * 1024
	// ZB is 1024 EB
	ZB int64 = TB * 1024
)

func init() {
	prometheus.MustRegister(version.NewCollector("ThunderDB"))
}

// StartMetricCollector starts collector registered in NewNodeCollector()
func StartMetricCollector() (registry *prometheus.Registry) {
	nc, err := NewNodeCollector()
	if err != nil {
		log.Errorf("couldn't create node collector: %s", err)
		return
	}

	registry = prometheus.NewRegistry()
	err = registry.Register(nc)
	if err != nil {
		log.Errorf("couldn't register collector: %s", err)
		return nil
	}

	log.Infof("Enabled collectors:")
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
