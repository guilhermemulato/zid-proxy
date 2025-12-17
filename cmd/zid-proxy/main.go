package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guilherme/zid-proxy/internal/config"
	"github.com/guilherme/zid-proxy/internal/logger"
	"github.com/guilherme/zid-proxy/internal/proxy"
	"github.com/guilherme/zid-proxy/internal/rules"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Parse command-line flags
	cfg := config.Default()

	flag.StringVar(&cfg.ListenAddr, "listen", cfg.ListenAddr, "Address to listen on (e.g., :443 or 0.0.0.0:8443)")
	flag.StringVar(&cfg.RulesFile, "rules", cfg.RulesFile, "Path to access rules file")
	flag.StringVar(&cfg.LogFile, "log", cfg.LogFile, "Path to log file")
	flag.StringVar(&cfg.PidFile, "pid", cfg.PidFile, "Path to PID file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("zid-proxy version %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	log.Printf("zid-proxy version %s starting...", Version)
	log.Printf("Configuration: listen=%s rules=%s log=%s", cfg.ListenAddr, cfg.RulesFile, cfg.LogFile)

	// Write PID file
	if err := writePidFile(cfg.PidFile); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}
	defer removePidFile(cfg.PidFile)

	// Initialize logger
	accessLogger, err := logger.New(cfg.LogFile)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer accessLogger.Close()

	// Start periodic log flush (every 1 second for near real-time logs)
	flushDone := startFlushTicker(accessLogger, 1*time.Second)
	defer close(flushDone)

	// Load rules
	ruleSet := rules.NewRuleSet(cfg.RulesFile)
	if err := ruleSet.Load(); err != nil {
		log.Fatalf("Failed to load rules: %v", err)
	}
	log.Printf("Loaded %d rules from %s", ruleSet.RuleCount(), cfg.RulesFile)

	// Create proxy server
	proxyCfg := proxy.Config{
		ListenAddr:   cfg.ListenAddr,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
	server := proxy.New(proxyCfg, ruleSet, accessLogger)

	// Start server
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Setup signal handlers
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	// Wait for signals
	for sig := range sigChan {
		switch sig {
		case syscall.SIGHUP:
			log.Println("Received SIGHUP, reloading rules...")
			if err := server.Reload(); err != nil {
				log.Printf("Failed to reload rules: %v", err)
			}
		case syscall.SIGTERM, syscall.SIGINT:
			log.Printf("Received %s, shutting down...", sig)
			if err := server.Stop(); err != nil {
				log.Printf("Error during shutdown: %v", err)
			}
			log.Println("Goodbye!")
			return
		}
	}
}

// writePidFile writes the current process PID to the specified file
func writePidFile(path string) error {
	pid := os.Getpid()
	return os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0644)
}

// removePidFile removes the PID file
func removePidFile(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to remove PID file: %v", err)
	}
}

// startFlushTicker starts a goroutine that periodically flushes the logger
func startFlushTicker(logger *logger.Logger, interval time.Duration) chan struct{} {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := logger.Flush(); err != nil {
					log.Printf("Warning: failed to flush log: %v", err)
				}
			case <-done:
				return
			}
		}
	}()
	return done
}
