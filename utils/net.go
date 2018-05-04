/*
 * MIT License
 *
 * Copyright (c) 2016-2018. ThunderDB
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package utils

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
)

var (
	NotEnoughPorts = errors.New("not enough ports in port range")
)

func testPort(bindAddr string, port int) bool {
	addr := net.JoinHostPort(bindAddr, fmt.Sprint(port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	defer ln.Close()
	return true
}

func GetRandomPorts(bindAddr string, minPort, maxPort, count int) (ports []int, err error) {
	ports = make([]int, 0, count)

	if count == 0 {
		return
	}

	if minPort == 0 {
		minPort = 1
	}

	if minPort > maxPort {
		err = NotEnoughPorts
		return
	}

	pivotPort := rand.Intn(maxPort-minPort) + minPort

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

	err = NotEnoughPorts

	return
}
