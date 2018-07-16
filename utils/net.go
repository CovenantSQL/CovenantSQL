/*
 * Copyright 2018 The ThunderDB Authors.
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
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
)

var (
	// ErrNotEnoughPorts defines error indicating random port allocation failure.
	ErrNotEnoughPorts = errors.New("not enough ports in port range")

	allocatedPorts sync.Map
	allocateLock   sync.Mutex
)

func testPort(bindAddr string, port int) bool {
	addr := net.JoinHostPort(bindAddr, fmt.Sprint(port))

	if _, ok := allocatedPorts.Load(addr); ok {
		return false
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	defer ln.Close()
	return true
}

// GetRandomPorts returns available random ports, previously allocated ports will be ignored.
func GetRandomPorts(bindAddr string, minPort, maxPort, count int) (ports []int, err error) {
	allocateLock.Lock()
	defer allocateLock.Unlock()

	ports = make([]int, 0, count)

	defer func() {
		// save the allocated ports
		if err == nil {
			for _, port := range ports {
				addr := net.JoinHostPort(bindAddr, fmt.Sprint(port))
				allocatedPorts.Store(addr, true)
				addr = net.JoinHostPort("0.0.0.0", fmt.Sprint(port))
				allocatedPorts.Store(addr, true)
			}
		}
	}()

	if count == 0 {
		return
	}

	if minPort == 0 {
		minPort = 1
	}

	if minPort > maxPort {
		err = ErrNotEnoughPorts
		return
	}

	pivotPort := minPort
	if maxPort != minPort {
		pivotPort = rand.Intn(maxPort-minPort) + minPort
	}

	for i := pivotPort; i <= maxPort; i++ {
		if testPort(bindAddr, i) {
			ports = append(ports, i)
		}

		if len(ports) == count {
			return
		}
	}

	for i := minPort; i < pivotPort; i++ {
		if testPort(bindAddr, i) {
			ports = append(ports, i)
		}

		if len(ports) == count {
			return
		}
	}

	err = ErrNotEnoughPorts

	return
}
