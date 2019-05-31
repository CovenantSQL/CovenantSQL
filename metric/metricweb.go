/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package metric

import (
	"expvar"
	"net/http"
	"runtime"
	"time"

	"github.com/pkg/errors"
	mw "github.com/zserge/metric"

	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

func collect(cc *CollectClient) (err error) {
	mfs, err := cc.Registry.Gather()
	if err != nil {
		err = errors.Wrap(err, "gathering node metrics failed")
		return
	}
	mm := make(SimpleMetricMap, 0)
	for _, mf := range mfs {
		mm[*mf.Name] = mf
		log.Debugf("gathered node: %v", mf)
	}
	crucialMetrics := mm.FilterCrucialMetrics()
	for k, v := range crucialMetrics {
		var val expvar.Var
		if val = expvar.Get(k); val == nil {
			expvar.Publish(k, mw.NewGauge("1h1m"))
			val = expvar.Get(k)
		}
		val.(mw.Metric).Add(v)
	}

	return
}

// InitMetricWeb initializes the /debug/metrics web.
func InitMetricWeb(metricWeb string) (err error) {
	// Some Go internal metrics
	expvar.Publish("go:numgoroutine", mw.NewGauge("1m1s", "5m5s", "1h1m"))
	expvar.Publish("go:numcgocall", mw.NewGauge("1m1s", "5m5s", "1h1m"))
	expvar.Publish("go:alloc", mw.NewGauge("1m1s", "5m5s", "1h1m"))
	expvar.Publish("go:alloctotal", mw.NewGauge("1m1s", "5m5s", "1h1m"))

	// start period provide service transaction generator
	// start prometheus collector
	cc := NewCollectClient()
	err = collect(cc)
	if err != nil {
		return
	}

	go func() {
		for range time.Tick(time.Minute) {
			_ = collect(cc)
		}
	}()

	go func() {
		for range time.Tick(5 * time.Second) {
			m := &runtime.MemStats{}
			runtime.ReadMemStats(m)
			expvar.Get("go:numgoroutine").(mw.Metric).Add(float64(runtime.NumGoroutine()))
			expvar.Get("go:numcgocall").(mw.Metric).Add(float64(runtime.NumCgoCall()))
			expvar.Get("go:alloc").(mw.Metric).Add(float64(m.Alloc) / float64(utils.MB))
			expvar.Get("go:alloctotal").(mw.Metric).Add(float64(m.TotalAlloc) / float64(utils.MB))
		}
	}()
	http.Handle("/debug/metrics", mw.Handler(mw.Exposed))
	go func() {
		_ = http.ListenAndServe(metricWeb, nil)
	}()
	return
}
