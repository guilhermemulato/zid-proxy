package activeips

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Tracker struct {
	mu   sync.Mutex
	ips  map[string]*ipStats
	opts Options
}

type Options struct {
	IdleTimeout time.Duration
	MaxIPs      int
	IdentityTTL time.Duration
}

type ipStats struct {
	SrcIP        string
	FirstSeen    time.Time
	LastActivity time.Time
	BytesIn      uint64
	BytesOut     uint64
	ActiveConns  int
	Machine      string
	Username     string
	IdentitySeen time.Time
}

type Snapshot struct {
	Version        int          `json:"version"`
	GeneratedAt    string       `json:"generated_at"`
	IdleTimeoutSec int          `json:"idle_timeout_sec"`
	IPs            []IPSnapshot `json:"ips"`
}

type IPSnapshot struct {
	SrcIP            string `json:"src_ip"`
	Machine          string `json:"machine,omitempty"`
	Username         string `json:"username,omitempty"`
	FirstSeen        string `json:"first_seen"`
	LastActivity     string `json:"last_activity"`
	IdentitySeen     string `json:"identity_seen,omitempty"`
	IdleSeconds      int    `json:"idle_seconds"`
	IdentityIdleSecs int    `json:"identity_idle_seconds,omitempty"`
	BytesIn          uint64 `json:"bytes_in"`
	BytesOut         uint64 `json:"bytes_out"`
	BytesTotal       uint64 `json:"bytes_total"`
	ActiveConns      int    `json:"active_conns"`
}

func New(opts Options) *Tracker {
	if opts.IdleTimeout <= 0 {
		opts.IdleTimeout = 120 * time.Second
	}
	if opts.MaxIPs <= 0 {
		opts.MaxIPs = 5000
	}
	if opts.IdentityTTL < 0 {
		opts.IdentityTTL = 0
	}
	return &Tracker{
		ips:  make(map[string]*ipStats),
		opts: opts,
	}
}

func normalizeSrcIP(srcIP string) string {
	srcIP = net.ParseIP(srcIP).String()
	if srcIP == "<nil>" {
		return ""
	}
	return srcIP
}

func sanitizeIdentityField(s string) string {
	if s == "" {
		return ""
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			continue
		}
		out = append(out, r)
		if len(out) >= 128 {
			break
		}
	}
	return strings.TrimSpace(string(out))
}

func (t *Tracker) ConnStart(srcIP string, now time.Time) {
	srcIP = normalizeSrcIP(srcIP)
	if srcIP == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	s := t.ips[srcIP]
	if s == nil {
		s = &ipStats{
			SrcIP:        srcIP,
			FirstSeen:    now,
			LastActivity: now,
		}
		t.ips[srcIP] = s
	}
	s.ActiveConns++
	if now.After(s.LastActivity) {
		s.LastActivity = now
	}
}

func (t *Tracker) ConnEnd(srcIP string, now time.Time) {
	srcIP = normalizeSrcIP(srcIP)
	if srcIP == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	s := t.ips[srcIP]
	if s == nil {
		return
	}
	if s.ActiveConns > 0 {
		s.ActiveConns--
	}
	if now.After(s.LastActivity) {
		s.LastActivity = now
	}
}

func (t *Tracker) AddBytes(srcIP string, bytesIn, bytesOut uint64, now time.Time) {
	srcIP = normalizeSrcIP(srcIP)
	if srcIP == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	s := t.ips[srcIP]
	if s == nil {
		s = &ipStats{
			SrcIP:        srcIP,
			FirstSeen:    now,
			LastActivity: now,
		}
		t.ips[srcIP] = s
	}
	s.BytesIn += bytesIn
	s.BytesOut += bytesOut
	if now.After(s.LastActivity) {
		s.LastActivity = now
	}
}

// SetIdentity updates machine/user for an already-tracked IP.
// It intentionally does not create a new tracked IP entry (Active IPs list is traffic-based).
func (t *Tracker) SetIdentity(srcIP, machine, username string, now time.Time) {
	srcIP = normalizeSrcIP(srcIP)
	if srcIP == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	s := t.ips[srcIP]
	if s == nil {
		log.Printf("[ACTIVEIPS] Warning: Attempt to set identity for non-tracked IP %s (machine=%q, username=%q). IP must have traffic first.", srcIP, machine, username)
		return
	}
	machine = sanitizeIdentityField(machine)
	username = sanitizeIdentityField(username)
	if machine != "" {
		s.Machine = machine
	}
	if username != "" {
		s.Username = username
	}
	if !now.IsZero() {
		s.IdentitySeen = now
	}
	log.Printf("[ACTIVEIPS] Identity set for IP %s: machine=%q username=%q", srcIP, machine, username)
}

func (t *Tracker) GC(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	idle := t.opts.IdleTimeout
	for ip, s := range t.ips {
		if s.ActiveConns > 0 {
			continue
		}
		if now.Sub(s.LastActivity) > idle {
			delete(t.ips, ip)
		}
	}

	// Optional size cap: keep most recently active entries.
	if len(t.ips) <= t.opts.MaxIPs {
		return
	}
	type pair struct {
		ip   string
		last time.Time
	}
	pairs := make([]pair, 0, len(t.ips))
	for ip, s := range t.ips {
		pairs = append(pairs, pair{ip: ip, last: s.LastActivity})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].last.After(pairs[j].last) })
	for i := t.opts.MaxIPs; i < len(pairs); i++ {
		delete(t.ips, pairs[i].ip)
	}
}

func (t *Tracker) Snapshot(now time.Time) Snapshot {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := Snapshot{
		Version:        1,
		GeneratedAt:    now.UTC().Format(time.RFC3339),
		IdleTimeoutSec: int(t.opts.IdleTimeout.Seconds()),
		IPs:            make([]IPSnapshot, 0, len(t.ips)),
	}
	for _, s := range t.ips {
		if t.opts.IdentityTTL > 0 {
			if s.IdentitySeen.IsZero() || now.Sub(s.IdentitySeen) > t.opts.IdentityTTL {
				if s.Machine != "" || s.Username != "" {
					log.Printf("[ACTIVEIPS] Clearing identity for IP %s (TTL expired, last seen: %v, TTL: %v)", s.SrcIP, s.IdentitySeen, t.opts.IdentityTTL)
				}
				s.Machine = ""
				s.Username = ""
			}
		}

		idle := int(now.Sub(s.LastActivity).Seconds())
		if idle < 0 {
			idle = 0
		}
		total := s.BytesIn + s.BytesOut

		snap := IPSnapshot{
			SrcIP:        s.SrcIP,
			Machine:      s.Machine,
			Username:     s.Username,
			FirstSeen:    s.FirstSeen.UTC().Format(time.RFC3339),
			LastActivity: s.LastActivity.UTC().Format(time.RFC3339),
			IdleSeconds:  idle,
			BytesIn:      s.BytesIn,
			BytesOut:     s.BytesOut,
			BytesTotal:   total,
			ActiveConns:  s.ActiveConns,
		}

		// Add identity timestamp and idle if available
		if !s.IdentitySeen.IsZero() {
			snap.IdentitySeen = s.IdentitySeen.UTC().Format(time.RFC3339)
			identityIdle := int(now.Sub(s.IdentitySeen).Seconds())
			if identityIdle < 0 {
				identityIdle = 0
			}
			snap.IdentityIdleSecs = identityIdle
		}

		out.IPs = append(out.IPs, snap)
	}

	sort.Slice(out.IPs, func(i, j int) bool {
		// Most recent activity first; tie-break by bytes total.
		if out.IPs[i].LastActivity != out.IPs[j].LastActivity {
			return out.IPs[i].LastActivity > out.IPs[j].LastActivity
		}
		return out.IPs[i].BytesTotal > out.IPs[j].BytesTotal
	})

	return out
}

func WriteSnapshotAtomic(path string, snap Snapshot) error {
	b, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
