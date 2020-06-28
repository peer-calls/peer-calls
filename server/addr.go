package server

import (
	"fmt"
	"net"
	"strconv"
)

func ParseUDPAddr(addr string) (*net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("Error parsing IP: %s", host)
	}

	portNumber, _ := strconv.Atoi(port)

	udpAddr := &net.UDPAddr{
		IP:   ip,
		Port: portNumber,
	}

	return udpAddr, nil
}

func ParseUDPAddrs(addrs []string) ([]*net.UDPAddr, error) {
	result := make([]*net.UDPAddr, 0, len(addrs))

	for _, addr := range addrs {
		udpAddr, err := ParseUDPAddr(addr)
		if err != nil {
			return nil, fmt.Errorf("Error parsing addr: %w", err)
		}
		result = append(result, udpAddr)
	}

	return result, nil
}
