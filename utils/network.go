package utils

import (
	"net"
)

// GetRandomUnusedPort returns a random unused port
func GetRandomUnusedPort() int {
	listener, _ := net.Listen("tcp", ":0")
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
