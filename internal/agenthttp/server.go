package agenthttp

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/guilherme/zid-proxy/internal/agent"
)

type Server struct {
	registry *agent.Registry
	onBeat   func(srcIP, machine, username string)
}

func New(registry *agent.Registry, onBeat func(srcIP, machine, username string)) *Server {
	return &Server{registry: registry, onBeat: onBeat}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/v1/agent/heartbeat", s.heartbeat)
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

type heartbeatRequest struct {
	Hostname string `json:"hostname"`
	Machine  string `json:"machine"`
	Username string `json:"username"`
	User     string `json:"user"`
}

func (s *Server) heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 8*1024))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	var req heartbeatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	machine := strings.TrimSpace(req.Hostname)
	if machine == "" {
		machine = strings.TrimSpace(req.Machine)
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = strings.TrimSpace(req.User)
	}

	srcIP, err := remoteIP(r.RemoteAddr)
	if err != nil {
		http.Error(w, "invalid remote addr", http.StatusBadRequest)
		return
	}

	now := time.Now()
	if s.registry != nil {
		s.registry.Update(srcIP, machine, username, now)
	}
	if s.onBeat != nil {
		s.onBeat(srcIP, machine, username)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}` + "\n"))
}

func remoteIP(remoteAddr string) (string, error) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// Some environments may provide an address without a port.
		ip := net.ParseIP(strings.TrimSpace(remoteAddr))
		if ip == nil {
			return "", errors.New("invalid remote addr")
		}
		return ip.String(), nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", errors.New("invalid remote addr ip")
	}
	return ip.String(), nil
}
