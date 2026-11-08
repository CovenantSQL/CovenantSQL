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

	mw "github.com/zserge/metric"

	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// collect adds a sample to each gauge whose reading succeeded.
func collect(dir string) {
	samples := map[string]float64{"cpu_count": float64(runtime.NumCPU())}
	if mem, err := MemAvailable(); err != nil {
		log.WithError(err).Warning("read available memory failed")
	} else {
		samples["mem_avail"] = float64(mem)
	}
	if load, err := LoadAvg(); err != nil {
		log.WithError(err).Warning("read load averages failed")
	} else {
		samples["load1"], samples["load5"], samples["load15"] = load[0], load[1], load[2]
	}
	if disk, err := DiskAvailable(dir); err != nil {
		log.WithError(err).Warning("read available disk space failed")
	} else {
		samples["fs_avail"] = float64(disk)
	}
	for k, v := range samples {
		var val expvar.Var
		if val = expvar.Get(k); val == nil {
			expvar.Publish(k, mw.NewGauge("1h1m"))
			val = expvar.Get(k)
		}
		val.(mw.Metric).Add(v)
	}
}

// InitMetricWeb serves /debug/metrics on metricWeb, with fs_avail measuring
// the file system holding dir.
func InitMetricWeb(metricWeb, dir string) {
	// Some Go internal metrics
	expvar.Publish("go:numgoroutine", mw.NewGauge("1m1s", "5m5s", "1h1m"))
	expvar.Publish("go:numcgocall", mw.NewGauge("1m1s", "5m5s", "1h1m"))
	expvar.Publish("go:alloc", mw.NewGauge("1m1s", "5m5s", "1h1m"))
	expvar.Publish("go:alloctotal", mw.NewGauge("1m1s", "5m5s", "1h1m"))

	collect(dir)
	go func() {
		for range time.Tick(time.Minute) {
			collect(dir)
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
}
