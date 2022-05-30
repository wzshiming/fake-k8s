package utils

import (
	"fmt"
	"net"
)

// GetUnusedPort returns an unused port
func GetUnusedPort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("get unused port error: %s", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
