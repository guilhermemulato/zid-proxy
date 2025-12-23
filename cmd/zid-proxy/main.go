package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/guilherme/zid-proxy/internal/activeips"
	"github.com/guilherme/zid-proxy/internal/agent"
	"github.com/guilherme/zid-proxy/internal/agenthttp"
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
	flag.StringVar(&cfg.ActiveIPsFile, "active-ips", cfg.ActiveIPsFile, "Active IPs JSON snapshot output path")
	activeIPsIntervalSec := flag.Int("active-ips-interval-seconds", int(cfg.ActiveIPsInterval.Seconds()), "How often to write active IPs snapshot (seconds)")
	activeIPsTimeoutSec := flag.Int("active-ips-timeout-seconds", int(cfg.ActiveIPsTimeout.Seconds()), "Idle timeout to drop IPs from snapshot (seconds)")
	flag.IntVar(&cfg.ActiveIPsMax, "active-ips-max", cfg.ActiveIPsMax, "Maximum number of tracked IPs")
	flag.StringVar(&cfg.AgentListenAddr, "agent-listen", cfg.AgentListenAddr, "Agent HTTP API listen address (e.g., 192.168.1.1:18443). Empty disables.")
	agentTTLSeconds := flag.Int("agent-ttl-seconds", int(cfg.AgentTTL.Seconds()), "Agent entry TTL (seconds)")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *activeIPsIntervalSec < 1 {
		*activeIPsIntervalSec = 1
	}
	if *activeIPsTimeoutSec < 5 {
		*activeIPsTimeoutSec = 5
	}
	cfg.ActiveIPsInterval = time.Duration(*activeIPsIntervalSec) * time.Second
	cfg.ActiveIPsTimeout = time.Duration(*activeIPsTimeoutSec) * time.Second
	// Agent TTL: minimum 10s, maximum 600s (10 minutes)
	if *agentTTLSeconds < 10 {
		*agentTTLSeconds = 10
	}
	if *agentTTLSeconds > 600 {
		*agentTTLSeconds = 600
	}
	cfg.AgentTTL = time.Duration(*agentTTLSeconds) * time.Second

	if *showVersion {
		fmt.Printf("zid-proxy version %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	log.Printf("zid-proxy version %s starting...", Version)
	log.Printf("Configuration: listen=%s rules=%s log=%s agent_listen=%s", cfg.ListenAddr, cfg.RulesFile, cfg.LogFile, cfg.AgentListenAddr)

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

	activeTracker := activeips.New(activeips.Options{
		IdleTimeout: cfg.ActiveIPsTimeout,
		MaxIPs:      cfg.ActiveIPsMax,
		IdentityTTL: cfg.AgentTTL,
	})

	agentRegistry := agent.NewRegistry(cfg.AgentTTL)

	// Periodically write snapshot to JSON (and GC idle entries)
	activeDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(cfg.ActiveIPsInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now()
				activeTracker.GC(now)
				agentRegistry.GC(now)
				snap := activeTracker.Snapshot(now)
				if err := activeips.WriteSnapshotAtomic(cfg.ActiveIPsFile, snap); err != nil {
					log.Printf("Warning: failed to write active IPs snapshot: %v", err)
				}
			case <-activeDone:
				return
			}
		}
	}()
	defer close(activeDone)

	var agentSrv *http.Server
	agentHTTPDone := make(chan struct{})
	if cfg.AgentListenAddr != "" {
		agentSrv = &http.Server{
			Addr: cfg.AgentListenAddr,
			Handler: agenthttp.New(agentRegistry, func(srcIP, machine, username string) {
				activeTracker.SetIdentity(srcIP, machine, username, time.Now())
			}).Handler(),
			ReadHeaderTimeout: 5 * time.Second,
		}
		go func() {
			defer close(agentHTTPDone)
			log.Printf("Agent HTTP API listening on %s", cfg.AgentListenAddr)
			if err := agentSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Agent HTTP API error: %v", err)
			}
		}()
	} else {
		close(agentHTTPDone)
	}

	// Create proxy server
	proxyCfg := proxy.Config{
		ListenAddr:   cfg.ListenAddr,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		ActiveIPs:    activeTracker,
		Agents:       agentRegistry,
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
			if err := accessLogger.Reopen(); err != nil {
				log.Printf("Failed to reopen log file: %v", err)
			}
		case syscall.SIGTERM, syscall.SIGINT:
			log.Printf("Received %s, shutting down...", sig)
			if err := server.Stop(); err != nil {
				log.Printf("Error during shutdown: %v", err)
			}
			if agentSrv != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = agentSrv.Shutdown(ctx)
				cancel()
			}
			<-agentHTTPDone
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
