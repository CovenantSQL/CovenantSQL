package utils

import (
	"net"
	"errors"
	"math/rand"
	"fmt"
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
