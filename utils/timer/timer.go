/*
 * Copyright 2019 The CovenantSQL Authors.
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

package timer

import (
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// Timer defines a stop watch timer for performance analysis.
type Timer struct {
	sync.Mutex
	start  time.Time
	names  []string
	pivots []time.Time
}

// NewTimer returns a new stop watch timer instance.
func NewTimer() *Timer {
	return &Timer{
		start: time.Now(),
	}
}

// Add records a time pivot.
func (t *Timer) Add(name string) {
	t.Lock()
	defer t.Unlock()

	t.names = append(t.names, name)
	t.pivots = append(t.pivots, time.Now())
}

// ToLogFields returns analysis results as log fields.
func (t *Timer) ToLogFields() log.Fields {
	var (
		m = t.ToMap()
		f = log.Fields{}
	)

	for k, v := range m {
		f[k] = v
	}

	return f
}

// ToMap returns analysis results as time duration map.
func (t *Timer) ToMap() map[string]time.Duration {
	t.Lock()
	defer t.Unlock()

	// calc
	lp := len(t.pivots)
	m := make(map[string]time.Duration, 1+lp)

	for i := 0; i != lp; i++ {
		var d time.Duration
		if i == 0 {
			d = t.pivots[i].Sub(t.start)
		} else {
			d = t.pivots[i].Sub(t.pivots[i-1])
		}

		m[t.names[i]] = d

		if i+1 == lp {
			// last one
			m["total"] = t.pivots[i].Sub(t.start)
		}
	}

	return m
}
