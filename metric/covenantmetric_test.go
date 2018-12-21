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
	"flag"
	"testing"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/prometheus/client_golang/prometheus"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMetrics(t *testing.T) {
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	reg := prometheus.NewRegistry()
	reg.MustRegister(NewCovenantSQLCollector())
	log.Debug("gauge Collector 'CovenantSQLCollector' registered.")

	time.Sleep(1100 * time.Millisecond)
	Convey("get metric", t, func() {
		mfs, err := reg.Gather()
		if err != nil {
			t.Fatal(err)
		}
		for _, mf := range mfs {
			log.Debugf("mfs: %s", mf.String())
		}
	})
}
