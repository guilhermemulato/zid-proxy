package appid

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestAppRuleSet_Load(t *testing.T) {
	// Create temp file with test rules
	tempDir := t.TempDir()
	rulesFile := filepath.Join(tempDir, "appid_rules.txt")

	content := `# ZID Proxy AppID Rules
# Format: TYPE;GROUP;APP_NAME

BLOCK_APP;acesso_restrito;netflix
BLOCK_APP;acesso_restrito;youtube
ALLOW_APP;acesso_restrito;microsoft_teams
BLOCK_APP;visitantes;*
`

	if err := os.WriteFile(rulesFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test rules: %v", err)
	}

	rs := NewAppRuleSet(rulesFile)
	if err := rs.Load(); err != nil {
		t.Fatalf("failed to load rules: %v", err)
	}

	if rs.Count() != 4 {
		t.Errorf("expected 4 rules, got %d", rs.Count())
	}

	rules := rs.ListRules()
	if len(rules) != 4 {
		t.Errorf("expected 4 rules in list, got %d", len(rules))
	}

	// Check first rule
	if rules[0].Type != RuleBlockApp {
		t.Error("expected first rule to be BLOCK_APP")
	}
	if rules[0].GroupName != "acesso_restrito" {
		t.Errorf("expected group 'acesso_restrito', got '%s'", rules[0].GroupName)
	}
	if rules[0].AppName != "netflix" {
		t.Errorf("expected app 'netflix', got '%s'", rules[0].AppName)
	}
}

func TestAppRuleSet_Match(t *testing.T) {
	rs := NewAppRuleSet("")
	rs.rules = []AppRule{
		{Type: RuleBlockApp, GroupName: "acesso_restrito", AppName: "netflix"},
		{Type: RuleBlockApp, GroupName: "acesso_restrito", AppName: "youtube"},
		{Type: RuleAllowApp, GroupName: "acesso_restrito", AppName: "microsoft_teams"},
		{Type: RuleBlockApp, GroupName: "visitantes", AppName: "*"},
	}

	tests := []struct {
		group   string
		app     string
		wantAction RuleType
		wantMatch  bool
	}{
		// acesso_restrito rules
		{"acesso_restrito", "netflix", RuleBlockApp, true},
		{"acesso_restrito", "youtube", RuleBlockApp, true},
		{"acesso_restrito", "microsoft_teams", RuleAllowApp, true},
		{"acesso_restrito", "spotify", "", false}, // no match

		// visitantes wildcard
		{"visitantes", "netflix", RuleBlockApp, true},
		{"visitantes", "youtube", RuleBlockApp, true},
		{"visitantes", "anything", RuleBlockApp, true},

		// unknown group
		{"unknown_group", "netflix", "", false},

		// empty inputs
		{"", "netflix", "", false},
		{"acesso_restrito", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.group+"_"+tt.app, func(t *testing.T) {
			action, matched := rs.Match(tt.group, tt.app)
			if matched != tt.wantMatch {
				t.Errorf("Match(%s, %s) matched = %v, want %v", tt.group, tt.app, matched, tt.wantMatch)
			}
			if action != tt.wantAction {
				t.Errorf("Match(%s, %s) action = %v, want %v", tt.group, tt.app, action, tt.wantAction)
			}
		})
	}
}

func TestAppRuleSet_Priority(t *testing.T) {
	// Test that ALLOW has priority over BLOCK
	rs := NewAppRuleSet("")
	rs.rules = []AppRule{
		{Type: RuleBlockApp, GroupName: "test", AppName: "netflix"},
		{Type: RuleAllowApp, GroupName: "test", AppName: "netflix"}, // ALLOW should win
	}

	action, matched := rs.Match("test", "netflix")
	if !matched {
		t.Error("expected match")
	}
	if action != RuleAllowApp {
		t.Errorf("expected ALLOW_APP to have priority, got %v", action)
	}
}

func TestAppRuleSet_FindGroup(t *testing.T) {
	rs := NewAppRuleSet("")

	_, net1, _ := net.ParseCIDR("192.168.1.0/24")
	_, net2, _ := net.ParseCIDR("10.0.0.0/8")

	rs.SetGroups(map[string][]*net.IPNet{
		"office":    {net1},
		"datacenter": {net2},
	})

	tests := []struct {
		ip        string
		wantGroup string
	}{
		{"192.168.1.100", "office"},
		{"192.168.1.1", "office"},
		{"10.0.0.1", "datacenter"},
		{"10.255.255.255", "datacenter"},
		{"172.16.0.1", ""}, // no match
		{"8.8.8.8", ""},     // no match
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			group := rs.FindGroup(net.ParseIP(tt.ip))
			if group != tt.wantGroup {
				t.Errorf("FindGroup(%s) = %s, want %s", tt.ip, group, tt.wantGroup)
			}
		})
	}
}

func TestAppRuleSet_MatchForIP(t *testing.T) {
	rs := NewAppRuleSet("")

	_, net1, _ := net.ParseCIDR("192.168.1.0/24")
	rs.SetGroups(map[string][]*net.IPNet{
		"office": {net1},
	})

	rs.rules = []AppRule{
		{Type: RuleBlockApp, GroupName: "office", AppName: "netflix"},
	}

	// IP in group, app blocked
	action, matched, groupName := rs.MatchForIP(net.ParseIP("192.168.1.100"), "netflix")
	if !matched {
		t.Error("expected match")
	}
	if action != RuleBlockApp {
		t.Errorf("expected BLOCK_APP, got %v", action)
	}
	if groupName != "office" {
		t.Errorf("expected group 'office', got '%s'", groupName)
	}

	// IP not in any group
	action, matched, groupName = rs.MatchForIP(net.ParseIP("172.16.0.1"), "netflix")
	if matched {
		t.Error("expected no match for IP not in any group")
	}
	if groupName != "" {
		t.Errorf("expected empty group, got '%s'", groupName)
	}
}
