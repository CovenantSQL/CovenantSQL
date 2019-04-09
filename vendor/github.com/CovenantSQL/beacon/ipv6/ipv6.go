package ipv6

import (
	"fmt"
	"net"

	"github.com/pkg/errors"
)

func ToIPv6(in []byte) (ips []net.IP, err error) {
	if len(in)%net.IPv6len != 0 {
		return nil, errors.New("must be n * 16 length")
	}
	ipCount := len(in) / net.IPv6len
	ips = make([]net.IP, ipCount)
	for i := 0; i < ipCount; i++ {
		ips[i] = make(net.IP, net.IPv6len)
		copy(ips[i], in[i*net.IPv6len:(i+1)*net.IPv6len])
	}
	return
}

func FromIPv6(ips []net.IP) (out []byte, err error) {
	ipCount := len(ips)
	out = make([]byte, ipCount*net.IPv6len)
	for i := 0; i < ipCount; i++ {
		copy(out[i*net.IPv6len:(i+1)*net.IPv6len], ips[i])
	}

	return
}

func FromDomain(domain string) (out []byte, err error) {
	var ips []net.IP
	allIPv6 := make([]net.IP, 0, 4)
	for i := 0; ; i++ {
		ips, err = net.LookupIP(fmt.Sprintf("%02d.%s", i, domain))
		if err != nil {
			if i > 0 {
				break
			} else {
				return
			}
		} else {
			if len(ips) == 0 {
				return nil, errors.New("empty IP list")
			}
			if len(ips[0]) != net.IPv6len {
				return nil, errors.Errorf("unexpected IP: %s", ips[0])
			}
			allIPv6 = append(allIPv6, ips[0])
		}

	}
	out, err = FromIPv6(allIPv6)
	if err != nil {
		return nil, errors.Errorf("convert from IPv6 failed: %v", err)
	}
	return
}
