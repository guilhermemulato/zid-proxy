package activeips

import (
	"testing"
	"time"
)

func TestTracker_AggregatesByIP(t *testing.T) {
	tr := New(Options{IdleTimeout: 10 * time.Second, MaxIPs: 100})
	now := time.Unix(1000, 0).UTC()

	tr.ConnStart("192.168.1.100", now)
	tr.AddBytes("192.168.1.100", 10, 20, now.Add(1*time.Second))
	tr.AddBytes("192.168.1.100", 5, 0, now.Add(2*time.Second))

	snap := tr.Snapshot(now.Add(3 * time.Second))
	if len(snap.IPs) != 1 {
		t.Fatalf("got %d ips, want 1", len(snap.IPs))
	}
	ip := snap.IPs[0]
	if ip.SrcIP != "192.168.1.100" {
		t.Fatalf("src_ip=%q", ip.SrcIP)
	}
	if ip.BytesIn != 15 || ip.BytesOut != 20 || ip.BytesTotal != 35 {
		t.Fatalf("bytes in/out/total=%d/%d/%d", ip.BytesIn, ip.BytesOut, ip.BytesTotal)
	}
	if ip.ActiveConns != 1 {
		t.Fatalf("active_conns=%d, want 1", ip.ActiveConns)
	}
}

func TestTracker_GC_RemovesIdle(t *testing.T) {
	tr := New(Options{IdleTimeout: 5 * time.Second, MaxIPs: 100})
	now := time.Unix(1000, 0).UTC()

	tr.AddBytes("192.168.1.10", 1, 1, now)
	tr.ConnStart("192.168.1.20", now)
	tr.ConnEnd("192.168.1.20", now)

	tr.GC(now.Add(6 * time.Second))
	snap := tr.Snapshot(now.Add(6 * time.Second))
	if len(snap.IPs) != 0 {
		t.Fatalf("got %d ips, want 0", len(snap.IPs))
	}
}

func TestTracker_SetIdentity_PersistsInSnapshot(t *testing.T) {
	tr := New(Options{IdleTimeout: 10 * time.Second, MaxIPs: 100})
	now := time.Unix(1000, 0).UTC()

	tr.AddBytes("192.168.1.10", 1, 1, now)
	tr.SetIdentity("192.168.1.10", "pc-01", "alice", now)

	snap := tr.Snapshot(now.Add(1 * time.Second))
	if len(snap.IPs) != 1 {
		t.Fatalf("got %d ips, want 1", len(snap.IPs))
	}
	if snap.IPs[0].Machine != "pc-01" || snap.IPs[0].Username != "alice" {
		t.Fatalf("machine/user=%q/%q", snap.IPs[0].Machine, snap.IPs[0].Username)
	}
}

func TestTracker_IdentityTTL_ClearsAfterTimeout(t *testing.T) {
	tr := New(Options{IdleTimeout: 10 * time.Second, MaxIPs: 100, IdentityTTL: 2 * time.Second})
	now := time.Unix(1000, 0).UTC()

	tr.AddBytes("192.168.1.10", 1, 1, now)
	tr.SetIdentity("192.168.1.10", "pc-01", "alice", now)

	snap := tr.Snapshot(now.Add(3 * time.Second))
	if len(snap.IPs) != 1 {
		t.Fatalf("got %d ips, want 1", len(snap.IPs))
	}
	if snap.IPs[0].Machine != "" || snap.IPs[0].Username != "" {
		t.Fatalf("expected identity cleared, got %q/%q", snap.IPs[0].Machine, snap.IPs[0].Username)
	}
}
