package rules

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestParseRule(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantType RuleType
		wantErr  bool
	}{
		{
			name:     "valid BLOCK rule",
			line:     "BLOCK;192.168.1.0/24;*.facebook.com",
			wantType: RuleBlock,
			wantErr:  false,
		},
		{
			name:     "valid ALLOW rule",
			line:     "ALLOW;10.0.0.1;example.com",
			wantType: RuleAllow,
			wantErr:  false,
		},
		{
			name:     "lowercase type",
			line:     "block;192.168.1.0/24;*.example.com",
			wantType: RuleBlock,
			wantErr:  false,
		},
		{
			name:    "invalid type",
			line:    "DENY;192.168.1.0/24;*.example.com",
			wantErr: true,
		},
		{
			name:    "invalid format - missing field",
			line:    "BLOCK;192.168.1.0/24",
			wantErr: true,
		},
		{
			name:    "invalid IP",
			line:    "BLOCK;invalid-ip;example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := parseRule(tt.line)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rule.Type != tt.wantType {
				t.Errorf("expected type %s, got %s", tt.wantType, rule.Type)
			}
		})
	}
}

func TestMatchHostname(t *testing.T) {
	tests := []struct {
		pattern  string
		hostname string
		want     bool
	}{
		{"example.com", "example.com", true},
		{"example.com", "www.example.com", false},
		{"*.example.com", "www.example.com", true},
		{"*.example.com", "sub.www.example.com", true},
		{"*.example.com", "example.com", true},
		{"*.example.com", "otherexample.com", false},
		{"*.sub.example.com", "www.sub.example.com", true},
		{"*.sub.example.com", "example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.hostname, func(t *testing.T) {
			got := matchHostname(tt.pattern, tt.hostname)
			if got != tt.want {
				t.Errorf("matchHostname(%q, %q) = %v, want %v", tt.pattern, tt.hostname, got, tt.want)
			}
		})
	}
}

func TestRuleSetMatch(t *testing.T) {
	// Create a temporary rules file
	content := `# Test rules
BLOCK;192.168.1.0/24;*.facebook.com
BLOCK;192.168.1.0/24;*.twitter.com
ALLOW;192.168.1.100;*.facebook.com
BLOCK;10.0.0.50;*.netflix.com
`
	tmpFile := createTempRulesFile(t, content)
	defer os.Remove(tmpFile)

	rs := NewRuleSet(tmpFile)
	if err := rs.Load(); err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	tests := []struct {
		name       string
		srcIP      string
		hostname   string
		wantAction RuleType
		wantMatch  bool
	}{
		{
			name:       "ALLOW takes priority over BLOCK",
			srcIP:      "192.168.1.100",
			hostname:   "www.facebook.com",
			wantAction: RuleAllow,
			wantMatch:  true,
		},
		{
			name:       "BLOCK for other IPs in subnet",
			srcIP:      "192.168.1.50",
			hostname:   "www.facebook.com",
			wantAction: RuleBlock,
			wantMatch:  true,
		},
		{
			name:       "BLOCK twitter for subnet",
			srcIP:      "192.168.1.50",
			hostname:   "api.twitter.com",
			wantAction: RuleBlock,
			wantMatch:  true,
		},
		{
			name:       "default ALLOW for non-matching IP",
			srcIP:      "10.0.0.1",
			hostname:   "www.google.com",
			wantAction: RuleAllow,
			wantMatch:  false,
		},
		{
			name:       "default ALLOW for non-matching hostname",
			srcIP:      "192.168.1.50",
			hostname:   "www.google.com",
			wantAction: RuleAllow,
			wantMatch:  false,
		},
		{
			name:       "BLOCK specific IP for netflix",
			srcIP:      "10.0.0.50",
			hostname:   "www.netflix.com",
			wantAction: RuleBlock,
			wantMatch:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcIP := net.ParseIP(tt.srcIP)
			action, matched, group := rs.Match(srcIP, tt.hostname)
			if action != tt.wantAction {
				t.Errorf("action = %s, want %s", action, tt.wantAction)
			}
			if matched != tt.wantMatch {
				t.Errorf("matched = %v, want %v", matched, tt.wantMatch)
			}
			if group != "" {
				t.Errorf("expected empty group for legacy rules, got %q", group)
			}
		})
	}
}

func TestRuleSetMatch_GroupedRules_FirstGroupWins(t *testing.T) {
	content := `# Groups are evaluated in order; first membership match wins
GROUP;acesso_liberado
MEMBER;192.168.1.0/24
ALLOW;*.facebook.com

GROUP;acesso_restrito
MEMBER;192.168.1.50
BLOCK;*.facebook.com
`
	tmpFile := createTempRulesFile(t, content)
	defer os.Remove(tmpFile)

	rs := NewRuleSet(tmpFile)
	if err := rs.Load(); err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	srcIP := net.ParseIP("192.168.1.50")
	action, matched, group := rs.Match(srcIP, "www.facebook.com")
	if action != RuleAllow || !matched || group != "acesso_liberado" {
		t.Fatalf("got action=%s matched=%v group=%q; want ALLOW true acesso_liberado", action, matched, group)
	}
}

func TestRuleSetMatch_GroupedRules_DefaultAllowWithinGroup(t *testing.T) {
	content := `GROUP;acesso_controlado
MEMBER;10.0.0.0/8
BLOCK;*.netflix.com
`
	tmpFile := createTempRulesFile(t, content)
	defer os.Remove(tmpFile)

	rs := NewRuleSet(tmpFile)
	if err := rs.Load(); err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	srcIP := net.ParseIP("10.1.2.3")
	action, matched, group := rs.Match(srcIP, "www.google.com")
	if action != RuleAllow || matched {
		t.Fatalf("got action=%s matched=%v; want ALLOW false", action, matched)
	}
	if group != "acesso_controlado" {
		t.Fatalf("got group=%q; want acesso_controlado", group)
	}
}

func TestParseRule_StripsInlineComment(t *testing.T) {
	rule, err := parseRule("BLOCK;192.168.1.0/24;*.facebook.com # social")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Hostname != "*.facebook.com" {
		t.Fatalf("got hostname=%q; want %q", rule.Hostname, "*.facebook.com")
	}
}

func TestRuleSetReload(t *testing.T) {
	content1 := `BLOCK;192.168.1.0/24;*.example.com`
	tmpFile := createTempRulesFile(t, content1)
	defer os.Remove(tmpFile)

	rs := NewRuleSet(tmpFile)
	if err := rs.Load(); err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	if rs.RuleCount() != 1 {
		t.Errorf("expected 1 rule, got %d", rs.RuleCount())
	}

	// Update the file
	content2 := `BLOCK;192.168.1.0/24;*.example.com
BLOCK;10.0.0.0/8;*.test.com
`
	if err := os.WriteFile(tmpFile, []byte(content2), 0644); err != nil {
		t.Fatalf("failed to update rules file: %v", err)
	}

	// Reload
	if err := rs.Reload(); err != nil {
		t.Fatalf("failed to reload rules: %v", err)
	}

	if rs.RuleCount() != 2 {
		t.Errorf("expected 2 rules after reload, got %d", rs.RuleCount())
	}
}

func TestParseIPOrCIDR(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"192.168.1.0/24", false},
		{"10.0.0.1", false},
		{"::1", false},
		{"2001:db8::/32", false},
		{"invalid", true},
		{"192.168.1.0/33", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseIPOrCIDR(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func createTempRulesFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "rules.txt")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp rules file: %v", err)
	}
	return tmpFile
}

func BenchmarkRuleSetMatch(b *testing.B) {
	content := `BLOCK;192.168.1.0/24;*.facebook.com
BLOCK;192.168.1.0/24;*.twitter.com
BLOCK;192.168.1.0/24;*.instagram.com
ALLOW;192.168.1.100;*.facebook.com
BLOCK;10.0.0.0/8;*.netflix.com
BLOCK;172.16.0.0/12;*.youtube.com
`
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "rules.txt")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		b.Fatalf("failed to create temp rules file: %v", err)
	}

	rs := NewRuleSet(tmpFile)
	if err := rs.Load(); err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}

	srcIP := net.ParseIP("192.168.1.50")
	hostname := "www.facebook.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.Match(srcIP, hostname)
	}
}
