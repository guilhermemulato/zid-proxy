//go:build !linux && !windows

package gateway

import (
	"errors"
	"net"
)

func Default() (net.IP, error) {
	return nil, errors.New("default gateway discovery not supported on this OS")
}
