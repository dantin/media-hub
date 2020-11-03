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
		return availableTCPPort()
	default:
		return 0, fmt.Errorf("unsupported network")
	}

}

func availableTCPPort() (int, error) {
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
