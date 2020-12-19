package server

import (
	"net"
	"strconv"

	"github.com/juju/errors"
)

func ParseUDPAddr(addr string) (*net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, errors.Errorf("failed to parse IP: %q", host)
	}

	portNumber, _ := strconv.Atoi(port)

	udpAddr := &net.UDPAddr{
		IP:   ip,
		Port: portNumber,
		Zone: "",
	}

	return udpAddr, nil
}

func ParseUDPAddrs(addrs []string) ([]*net.UDPAddr, error) {
	result := make([]*net.UDPAddr, 0, len(addrs))

	for _, addr := range addrs {
		udpAddr, err := ParseUDPAddr(addr)
		if err != nil {
			return nil, errors.Annotatef(err, "failed to parse UDP addr: %s", addr)
		}

		result = append(result, udpAddr)
	}

	return result, nil
}
