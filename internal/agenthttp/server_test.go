package agenthttp

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guilherme/zid-proxy/internal/agent"
)

func TestHeartbeat_UsesRemoteAddrIP(t *testing.T) {
	reg := agent.NewRegistry(10 * time.Second)
	var called bool
	var gotIP, gotMachine, gotUser string
	s := New(reg, func(srcIP, machine, username string) {
		called = true
		gotIP = srcIP
		gotMachine = machine
		gotUser = username
	})

	req := httptest.NewRequest(http.MethodPost, "http://example/api/v1/agent/heartbeat", bytes.NewBufferString(`{"hostname":"pc","username":"bob","ip":"1.2.3.4"}`))
	req.RemoteAddr = "192.168.1.55:12345"
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	m, u, ok := reg.Lookup("192.168.1.55", time.Now())
	if !ok || m != "pc" || u != "bob" {
		t.Fatalf("lookup=%v machine=%q user=%q", ok, m, u)
	}
	if !called || gotIP != "192.168.1.55" || gotMachine != "pc" || gotUser != "bob" {
		t.Fatalf("callback called=%v ip=%q machine=%q user=%q", called, gotIP, gotMachine, gotUser)
	}
}
