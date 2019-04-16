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

package utils

import (
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var prof struct {
	cpu *os.File
	mem *os.File
}

// StartProfile initializes the CPU and memory profile, if specified.
func StartProfile(cpuprofile, memprofile string) error {
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.WithField("file", cpuprofile).WithError(err).Error("failed to create CPU profile file")
			return err
		}
		log.WithField("file", cpuprofile).Info("writing CPU profiling to file")
		prof.cpu = f
		pprof.StartCPUProfile(prof.cpu)
	}

	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.WithField("file", memprofile).WithError(err).Error("failed to create memory profile file")
			return err
		}
		log.WithField("file", memprofile).WithError(err).Info("writing memory profiling to file")
		prof.mem = f
		runtime.MemProfileRate = 4096
	}
	return nil
}

// StopProfile closes the CPU and memory profiles if they are running.
func StopProfile() {
	if prof.cpu != nil {
		pprof.StopCPUProfile()
		prof.cpu.Close()
		log.Info("CPU profiling stopped")
	}
	if prof.mem != nil {
		pprof.WriteHeapProfile(prof.mem)
		prof.mem.Close()
		log.Info("memory profiling stopped")
	}
}
