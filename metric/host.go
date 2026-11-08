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
	"github.com/mackerelio/go-osstat/loadavg"
	"github.com/mackerelio/go-osstat/memory"
	"golang.org/x/sys/unix"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

// Host holds the resources of the host a node runs on.
type Host struct {
	// MemAvailable is the memory in bytes that is not in use, which
	// go-osstat derives from MemAvailable on Linux and from free and cached
	// pages on darwin.
	MemAvailable         uint64
	Load1, Load5, Load15 float64
	// DiskAvailable is the space in bytes that an unprivileged process can
	// still write on the file system holding the directory given to ReadHost.
	DiskAvailable uint64
}

// ReadHost reads the host's memory, load averages and the space available
// on the file system holding dir. It logs what it cannot read and leaves
// that zero.
func ReadHost(dir string) (h Host) {
	if m, err := memory.Get(); err != nil {
		log.WithError(err).Error("read memory stats failed")
	} else {
		h.MemAvailable = m.Total - m.Used
	}
	if l, err := loadavg.Get(); err != nil {
		log.WithError(err).Error("read load averages failed")
	} else {
		h.Load1, h.Load5, h.Load15 = l.Loadavg1, l.Loadavg5, l.Loadavg15
	}
	var fs unix.Statfs_t
	if err := unix.Statfs(dir, &fs); err != nil {
		log.WithError(err).WithField("dir", dir).Error("read file system stats failed")
	} else {
		h.DiskAvailable = uint64(fs.Bavail) * uint64(fs.Bsize)
	}
	return
}
