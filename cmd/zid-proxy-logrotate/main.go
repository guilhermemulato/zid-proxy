package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/guilherme/zid-proxy/internal/logrotate"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	logPath := flag.String("log", "/var/log/zid-proxy.log", "Log file path")
	keepDays := flag.Int("keep-days", 7, "How many rotated daily logs to keep (>=1)")
	pidFile := flag.String("pid", "/var/run/zid-proxy.pid", "PID file to signal after rotating")
	sendHup := flag.Bool("hup", false, "Send SIGHUP to the PID in -pid after rotating")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("zid-proxy-logrotate version %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	rotated, err := logrotate.Run(logrotate.Options{
		LogPath:  *logPath,
		KeepDays: *keepDays,
		Now:      time.Now(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(2)
	}

	if rotated && *sendHup {
		if err := hupFromPidFile(*pidFile); err != nil {
			fmt.Fprintf(os.Stderr, "WARN: %v\n", err)
		}
	}
}

func hupFromPidFile(pidFile string) error {
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		return err
	}
	pidStr := strings.TrimSpace(string(raw))
	if pidStr == "" {
		return fmt.Errorf("pid file is empty: %s", pidFile)
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid < 2 {
		return fmt.Errorf("invalid pid in %s: %q", pidFile, pidStr)
	}
	if err := syscall.Kill(pid, syscall.SIGHUP); err != nil {
		return fmt.Errorf("send SIGHUP to pid %d: %w", pid, err)
	}
	return nil
}
