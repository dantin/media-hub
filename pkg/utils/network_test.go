package utils

import (
	"fmt"
	"net"
	"testing"
)

func TestNextPort(t *testing.T) {
	port, err := NextPort("tcp")
	if err != nil {
		t.Error(err)
	}

	if port == 0 {
		t.Error("port == 0")
	}

	lt, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Error(err)
	}
	defer lt.Close()

	port, err = NextPort("udp")
	if err != nil {
		t.Error(err)
	}

	if port == 0 {
		t.Error("port == 0")
	}
}
