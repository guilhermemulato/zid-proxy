package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/guilherme/zid-proxy/internal/agentui"
	"github.com/guilherme/zid-proxy/internal/gateway"
)

const (
	heartbeatTimeout = 5 * time.Second
	heartbeatPath    = "/api/v1/agent/heartbeat"
)

type heartbeatPayload struct {
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	Version  string `json:"agent_version,omitempty"`
}

// runHeartbeat runs the heartbeat loop, sending periodic updates to the pfSense server.
// It respects context cancellation for graceful shutdown.
func runHeartbeat(ctx context.Context, logMgr *agentui.LogManager, statusMgr *agentui.StatusManager, cfgMgr *agentui.ConfigManager, version string) {
	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")
	if username == "" {
		username = os.Getenv("USER")
	}

	logMgr.Addf("Heartbeat service started (hostname: %s, user: %s)", hostname, username)

	client := &http.Client{Timeout: heartbeatTimeout}
	cfgCh := cfgMgr.Subscribe()
	defer cfgMgr.Unsubscribe(cfgCh)

	// Send first heartbeat immediately
	sendHeartbeat(ctx, client, logMgr, statusMgr, hostname, username, cfgMgr.Get(), version)

	timer := time.NewTimer(nextInterval(cfgMgr.Get()))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			sendHeartbeat(ctx, client, logMgr, statusMgr, hostname, username, cfgMgr.Get(), version)
			timer.Reset(nextInterval(cfgMgr.Get()))
		case <-cfgCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(nextInterval(cfgMgr.Get()))
		case <-ctx.Done():
			logMgr.Add("Heartbeat service stopped")
			return
		}
	}
}

func sendHeartbeat(ctx context.Context, client *http.Client, logMgr *agentui.LogManager, statusMgr *agentui.StatusManager, hostname, username string, cfg agentui.Config, version string) {
	targets := discoverTargets(cfg)
	if len(targets) == 0 {
		logMgr.Add("Heartbeat failed: no pfSense targets available")
		statusMgr.Set(agentui.HeartbeatStatus{
			State:     agentui.HeartbeatFail,
			Message:   "no pfSense targets available",
			Timestamp: time.Now(),
		})
		return
	}

	payload := heartbeatPayload{
		Hostname: hostname,
		Username: username,
		Version:  version,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logMgr.Addf("Heartbeat failed: marshal error: %v", err)
		return
	}

	for _, url := range targets {
		if ctx.Err() != nil {
			return // context cancelled
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			host := extractHost(url)
			logMgr.Addf("Heartbeat failed: %s: %v", host, err)
			statusMgr.Set(agentui.HeartbeatStatus{
				State:     agentui.HeartbeatFail,
				Target:    host,
				Message:   err.Error(),
				Timestamp: time.Now(),
			})
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			host := extractHost(url)
			logMgr.Addf("Heartbeat OK: %s", host)
			statusMgr.Set(agentui.HeartbeatStatus{
				State:     agentui.HeartbeatOK,
				Target:    host,
				Message:   "ok",
				Timestamp: time.Now(),
			})
			return
		}

		host := extractHost(url)
		logMgr.Addf("Heartbeat rejected: %s: status %d", host, resp.StatusCode)
		statusMgr.Set(agentui.HeartbeatStatus{
			State:     agentui.HeartbeatFail,
			Target:    host,
			Message:   fmt.Sprintf("status %d", resp.StatusCode),
			Timestamp: time.Now(),
		})
	}
}

// discoverTargets returns a list of URLs to try for heartbeat, in order of preference.
func discoverTargets(cfg agentui.Config) []string {
	var targets []string

	// Try gateway first
	if gw, err := gateway.Default(); err == nil && gw != nil {
		url := fmt.Sprintf("http://%s:%d%s", gw.String(), cfg.Port, heartbeatPath)
		targets = append(targets, url)
	}

	// DNS fallback
	if strings.TrimSpace(cfg.DNSFallback) != "" {
		url := fmt.Sprintf("http://%s:%d%s", cfg.DNSFallback, cfg.Port, heartbeatPath)
		targets = append(targets, url)
	}

	return targets
}

// extractHost extracts the host portion from a URL for logging.
func extractHost(url string) string {
	// Simple extraction: remove http:// and everything after the port
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	if idx := strings.Index(url, "/"); idx > 0 {
		url = url[:idx]
	}
	return url
}

func nextInterval(cfg agentui.Config) time.Duration {
	interval := time.Duration(cfg.IntervalSeconds) * time.Second
	if interval <= 0 {
		return 30 * time.Second
	}
	return interval
}
