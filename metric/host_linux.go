/*
 * Copyright 2026 The CovenantSQL Authors.
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
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MemAvailable returns MemAvailable from /proc/meminfo: the bytes of memory
// available to new work without swapping.
func MemAvailable() (uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		// MemAvailable:    8165432 kB
		fields := strings.Fields(s.Text())
		if len(fields) == 3 && fields[0] == "MemAvailable:" && fields[2] == "kB" {
			kb, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parse /proc/meminfo: %w", err)
			}
			return kb * 1024, nil
		}
	}
	if err = s.Err(); err != nil {
		return 0, err
	}
	return 0, errors.New("no MemAvailable in /proc/meminfo")
}

// LoadAvg returns the 1, 5 and 15 minute load averages from /proc/loadavg.
func LoadAvg() (l [3]float64, err error) {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return l, err
	}
	if _, err = fmt.Sscan(string(b), &l[0], &l[1], &l[2]); err != nil {
		return [3]float64{}, fmt.Errorf("parse /proc/loadavg: %w", err)
	}
	return l, nil
}
