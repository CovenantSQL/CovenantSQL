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
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	// ErrNotEnoughPorts defines error indicating random port allocation failure.
	ErrNotEnoughPorts = errors.New("not enough ports in port range")

	allocatedPorts sync.Map
	allocateLock   sync.Mutex
)

func testPortConnectable(addr string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		log.Infof("test dial to %s failed", addr)
		return false
	}

	conn.Close()
	return true
}

func testPort(bindAddr string, port int, excludeAllocated bool) bool {
	addr := net.JoinHostPort(bindAddr, fmt.Sprint(port))

	if excludeAllocated {
		if _, ok := allocatedPorts.Load(addr); ok {
			return false
		}
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// WaitToConnect returns only when port is ready to connect or canceled by context.
func WaitToConnect(ctx context.Context, bindAddr string, ports []int, interval time.Duration) (err error) {
	for {
	continueCheckC:
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		case <-time.After(interval):
			for _, port := range ports {
				addr := net.JoinHostPort(bindAddr, fmt.Sprint(port))
				if !testPortConnectable(addr, 100*time.Millisecond) {
					goto continueCheckC
				}
			}
			return
		}
	}
}

// WaitForPorts returns only when port is ready to listen or canceled by context.
func WaitForPorts(ctx context.Context, bindAddr string, ports []int, interval time.Duration) (err error) {
	for {
	continueCheck:
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		case <-time.After(interval):
			for _, port := range ports {
				if !testPort(bindAddr, port, false) {
					goto continueCheck
				}
			}
			return
		}
	}
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
		if testPort(bindAddr, i, true) {
			ports = append(ports, i)
		}

		if len(ports) == count {
			return
		}
	}

	for i := minPort; i < pivotPort; i++ {
		if testPort(bindAddr, i, true) {
			ports = append(ports, i)
		}

		if len(ports) == count {
			return
		}
	}

	err = ErrNotEnoughPorts

	return
}
