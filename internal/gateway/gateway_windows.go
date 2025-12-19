//go:build windows

package gateway

import (
	"errors"
	"net"
	"os/exec"
	"strings"
)

func Default() (net.IP, error) {
	// Use `route print -4` and parse the default route line:
	//   0.0.0.0          0.0.0.0      192.168.1.1     192.168.1.50     25
	out, err := exec.Command("route", "print", "-4").CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "0.0.0.0") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if fields[0] != "0.0.0.0" || fields[1] != "0.0.0.0" {
			continue
		}
		ip := net.ParseIP(fields[2])
		if ip == nil {
			continue
		}
		return ip, nil
	}

	return nil, errors.New("default gateway not found")
}
