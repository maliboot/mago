package helper

import (
	"fmt"
	"net"
	"time"
)

func TelnetLikeTest(ip string, port string) error {
	address := net.JoinHostPort(ip, port)

	dialer := &net.Dialer{
		Timeout: 3 * time.Second,
	}

	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("dial tcp %s: %w", address, err)
	}
	defer conn.Close()

	return nil
}
