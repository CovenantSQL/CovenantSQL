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
	"encoding/binary"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// MemAvailable returns the bytes in free, purgeable and file-backed pages,
// which macOS reclaims without swapping.
func MemAvailable() (uint64, error) {
	var pages uint64
	for _, name := range []string{
		"vm.page_free_count",
		"vm.page_purgeable_count",
		"vm.page_pageable_external_count",
	} {
		n, err := unix.SysctlUint32(name)
		if err != nil {
			return 0, fmt.Errorf("sysctl %s: %w", name, err)
		}
		pages += uint64(n)
	}
	return pages * uint64(os.Getpagesize()), nil
}

// LoadAvg returns the 1, 5 and 15 minute load averages from the vm.loadavg
// sysctl.
func LoadAvg() (l [3]float64, err error) {
	// struct loadavg { fixpt_t ldavg[3]; long fscale; }
	b, err := unix.SysctlRaw("vm.loadavg")
	if err != nil {
		return l, fmt.Errorf("sysctl vm.loadavg: %w", err)
	}
	if len(b) != 24 {
		return l, fmt.Errorf("sysctl vm.loadavg: got %d bytes, want 24", len(b))
	}
	scale := float64(binary.NativeEndian.Uint64(b[16:]))
	for i := range l {
		l[i] = float64(binary.NativeEndian.Uint32(b[4*i:])) / scale
	}
	return l, nil
}
