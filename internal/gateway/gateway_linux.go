//go:build linux

package gateway

import (
	"bufio"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
)

func Default() (net.IP, error) {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Skip header
	if !scanner.Scan() {
		return nil, errors.New("empty route table")
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		// Iface Destination Gateway Flags RefCnt Use Metric Mask ...
		if len(fields) < 4 {
			continue
		}
		dest := fields[1]
		gwHex := fields[2]
		flagsHex := fields[3]
		if dest != "00000000" {
			continue
		}
		flags, err := strconv.ParseUint(flagsHex, 16, 64)
		if err != nil {
			continue
		}
		// RTF_GATEWAY = 0x2
		if flags&0x2 == 0 {
			continue
		}
		gwVal, err := strconv.ParseUint(gwHex, 16, 32)
		if err != nil {
			continue
		}
		ip := net.IPv4(byte(gwVal), byte(gwVal>>8), byte(gwVal>>16), byte(gwVal>>24))
		return ip, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("default gateway not found")
}
