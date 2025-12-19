package agent

import (
	"testing"
	"time"
)

func TestRegistry_UpdateLookupAndTTL(t *testing.T) {
	r := NewRegistry(2 * time.Second)
	now := time.Unix(1000, 0).UTC()

	r.Update("192.168.1.10", "pc-01", "alice", now)
	m, u, ok := r.Lookup("192.168.1.10", now.Add(500*time.Millisecond))
	if !ok || m != "pc-01" || u != "alice" {
		t.Fatalf("lookup=%v machine=%q user=%q", ok, m, u)
	}

	_, _, ok = r.Lookup("192.168.1.10", now.Add(3*time.Second))
	if ok {
		t.Fatalf("expected entry to expire")
	}
}
