package agent

import (
	"net"
	"strings"
	"sync"
	"time"
)

type Info struct {
	Machine  string
	Username string
	LastSeen time.Time
}

type Registry struct {
	mu  sync.Mutex
	ttl time.Duration
	ips map[string]Info
}

func NewRegistry(ttl time.Duration) *Registry {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &Registry{
		ttl: ttl,
		ips: make(map[string]Info),
	}
}

func normalizeIP(ip string) string {
	ip = strings.TrimSpace(ip)
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	return parsed.String()
}

func sanitizeField(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}

func (r *Registry) Update(srcIP, machine, username string, now time.Time) {
	srcIP = normalizeIP(srcIP)
	if srcIP == "" {
		return
	}

	info := Info{
		Machine:  sanitizeField(machine),
		Username: sanitizeField(username),
		LastSeen: now,
	}

	r.mu.Lock()
	r.ips[srcIP] = info
	r.mu.Unlock()
}

func (r *Registry) Lookup(srcIP string, now time.Time) (machine, username string, ok bool) {
	srcIP = normalizeIP(srcIP)
	if srcIP == "" {
		return "", "", false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.ips[srcIP]
	if !exists {
		return "", "", false
	}
	if r.ttl > 0 && now.Sub(info.LastSeen) > r.ttl {
		delete(r.ips, srcIP)
		return "", "", false
	}
	return info.Machine, info.Username, true
}

func (r *Registry) GC(now time.Time) {
	if r.ttl <= 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for ip, info := range r.ips {
		if now.Sub(info.LastSeen) > r.ttl {
			delete(r.ips, ip)
		}
	}
}
