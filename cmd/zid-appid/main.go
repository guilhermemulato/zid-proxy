// zid-appid is a Deep Packet Inspection daemon for application identification.
// It detects applications like Netflix, YouTube, Facebook based on traffic analysis
// and provides this information to zid-proxy via Unix socket.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guilherme/zid-proxy/internal/appid"
)

const (
	defaultSocketPath    = "/var/run/zid-appid.sock"
	defaultPidFile       = "/var/run/zid-appid.pid"
	defaultRulesFile     = "/usr/local/etc/zid-proxy/appid_rules.txt"
	defaultMaxFlows      = 10000
	defaultFlowTTL       = 5 * time.Minute
	defaultGCInterval    = 30 * time.Second
)

var (
	version = "1.0.0"
)

func main() {
	// Parse command line flags
	socketPath := flag.String("socket", defaultSocketPath, "Unix socket path")
	pidFile := flag.String("pid", defaultPidFile, "PID file path")
	rulesFile := flag.String("rules", defaultRulesFile, "AppID rules file path")
	maxFlows := flag.Int("max-flows", defaultMaxFlows, "Maximum number of flows to track")
	flowTTL := flag.Duration("flow-ttl", defaultFlowTTL, "Flow TTL (idle timeout)")
	gcInterval := flag.Duration("gc-interval", defaultGCInterval, "Garbage collection interval")
	showVersion := flag.Bool("version", false, "Show version and exit")

	flag.Parse()

	if *showVersion {
		fmt.Printf("zid-appid version %s\n", version)
		os.Exit(0)
	}

	log.Printf("zid-appid %s starting...", version)

	// Write PID file
	if err := writePidFile(*pidFile); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}
	defer os.Remove(*pidFile)

	// Initialize components
	flowCache := appid.NewFlowCache(*maxFlows, *flowTTL)
	detector := appid.NewDetector()
	ruleSet := appid.NewAppRuleSet(*rulesFile)

	// Load rules (optional - file may not exist yet)
	if err := ruleSet.Load(); err != nil {
		log.Printf("Warning: failed to load AppID rules: %v", err)
	} else {
		log.Printf("Loaded %d AppID rules", ruleSet.Count())
	}

	// Start Unix socket server
	server := appid.NewServer(*socketPath, flowCache, detector)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Printf("Listening on Unix socket: %s", *socketPath)

	// Start garbage collection goroutine
	ctx, cancel := context.WithCancel(context.Background())
	go gcLoop(ctx, flowCache, *gcInterval)

	// Log app detection stats
	log.Printf("Application detector initialized with %d app definitions", len(detector.ListApps()))

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGHUP:
			// Reload rules
			log.Println("Received SIGHUP, reloading rules...")
			if err := ruleSet.Reload(); err != nil {
				log.Printf("Failed to reload rules: %v", err)
			} else {
				log.Printf("Reloaded %d AppID rules", ruleSet.Count())
			}

		case syscall.SIGINT, syscall.SIGTERM:
			log.Println("Shutting down...")
			cancel()
			server.Stop()
			log.Println("zid-appid stopped")
			return
		}
	}
}

// writePidFile writes the current process ID to a file.
func writePidFile(path string) error {
	pid := os.Getpid()
	return os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0644)
}

// gcLoop periodically runs garbage collection on the flow cache.
func gcLoop(ctx context.Context, cache *appid.FlowCache, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			removed := cache.GC(time.Now())
			if removed > 0 {
				log.Printf("GC: removed %d expired flows", removed)
			}
		}
	}
}
