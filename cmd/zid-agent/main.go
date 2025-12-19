package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/guilherme/zid-proxy/internal/gateway"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

type heartbeatPayload struct {
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	Version  string `json:"agent_version,omitempty"`
}

func main() {
	port := flag.Int("port", 18443, "Agent API port on pfSense (default: 18443)")
	dnsHost := flag.String("dns", "zid-proxy.lan", "DNS fallback host for pfSense (default: zid-proxy.lan)")
	path := flag.String("path", "/api/v1/agent/heartbeat", "Heartbeat path")
	interval := flag.Duration("interval", 30*time.Second, "Heartbeat interval")
	once := flag.Bool("once", false, "Send one heartbeat and exit")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("zid-agent version %s (built %s)\n", Version, BuildTime)
		return
	}

	if *port < 1 || *port > 65535 {
		log.Fatalf("invalid port: %d", *port)
	}
	if *interval < 5*time.Second {
		*interval = 5 * time.Second
	}
	if !strings.HasPrefix(*path, "/") {
		*path = "/" + *path
	}

	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")
	if username == "" {
		username = os.Getenv("USER")
	}

	client := &http.Client{Timeout: 5 * time.Second}

	send := func() {
		var targets []string
		if gw, err := gateway.Default(); err == nil && gw != nil {
			targets = append(targets, fmt.Sprintf("http://%s:%d%s", gw.String(), *port, *path))
		}
		if strings.TrimSpace(*dnsHost) != "" {
			targets = append(targets, fmt.Sprintf("http://%s:%d%s", strings.TrimSpace(*dnsHost), *port, *path))
		}
		if len(targets) == 0 {
			log.Printf("no targets available (no gateway and no dns host)")
			return
		}

		payload := heartbeatPayload{
			Hostname: hostname,
			Username: username,
			Version:  Version,
		}
		b, _ := json.Marshal(payload)

		for _, url := range targets {
			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				log.Printf("heartbeat failed: url=%s err=%v", url, err)
				continue
			}
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				log.Printf("heartbeat ok: url=%s", url)
				return
			}
			log.Printf("heartbeat rejected: url=%s status=%s", url, resp.Status)
		}
	}

	send()
	if *once {
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-ticker.C:
			send()
		case <-sig:
			return
		}
	}
}
