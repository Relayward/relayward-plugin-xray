package socket

import (
	"fmt"
	"net"
	"os"
)

func Listen(path string) (net.Listener, error) {
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("open plugin control socket: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("protect plugin control socket: %w", err)
	}
	return listener, nil
}
