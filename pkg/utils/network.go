package utils

import (
	"fmt"
	"net"
)

const localAddr = "localhost:0"

// NextPort asks kernel for a free open port that is ready to use.
func NextPort(network string) (int, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
		return nextTCPPort()
	case "udp", "udp4", "udp6":
		return nextUDPPort()
	default:
		return 0, fmt.Errorf("unsupported network")
	}

}

func nextTCPPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		return 0, nil
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, nil
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

func nextUDPPort() (int, error) {
	addr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return 0, nil
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, nil
	}
	defer conn.Close()

	return l.LocalAddr(), nil
}
