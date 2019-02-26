//
//Copyright 2018 The CovenantSQL Authors.
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
//

// +build darwin linux openbsd
// +build !nomeminfo

package metric

import (
	"fmt"
	"path"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

const (
	memInfoSubsystem = "memory"
)

type meminfoCollector struct{}

// ProcPath is the proc system path, expose for test.
var ProcPath = "/proc"

// NewMeminfoCollector returns a new Collector exposing memory stats.
func NewMeminfoCollector() (Collector, error) {
	return &meminfoCollector{}, nil
}

// Update calls (*meminfoCollector).getMemInfo to get the platform specific
// memory metrics.
func (c *meminfoCollector) Update(ch chan<- prometheus.Metric) error {
	memInfo, err := c.getMemInfo()
	if err != nil {
		return fmt.Errorf("couldn't get meminfo: %s", err)
	}
	log.Debugf("set node_mem: %#v", memInfo)
	for k, v := range memInfo {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, memInfoSubsystem, k),
				fmt.Sprintf("Memory information field %s.", k),
				nil, nil,
			),
			prometheus.GaugeValue, v,
		)
	}
	return nil
}

func procFilePath(name string) string {
	return path.Join(ProcPath, name)
}

func sysFilePath(name string) string {
	return path.Join("/sys", name)
}
