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
	"os"

	"golang.org/x/sys/unix"
)

// DiskAvailable returns the bytes that unprivileged processes can still
// allocate on the file system holding dir.
func DiskAvailable(dir string) (uint64, error) {
	var fs unix.Statfs_t
	if err := unix.Statfs(dir, &fs); err != nil {
		return 0, &os.PathError{Op: "statfs", Path: dir, Err: err}
	}
	return uint64(fs.Bavail) * uint64(fs.Bsize), nil
}
