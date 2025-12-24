package appid

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// RuleType represents the type of an AppID rule.
type RuleType string

const (
	RuleAllowApp RuleType = "ALLOW_APP"
	RuleBlockApp RuleType = "BLOCK_APP"
)

// AppRule represents a rule that matches group + application.
type AppRule struct {
	Type      RuleType // ALLOW_APP or BLOCK_APP
	GroupName string   // Group name (e.g., "acesso_restrito")
	AppName   string   // App name (e.g., "netflix") or "*" for all
}

// GroupMember represents an IP or CIDR that belongs to a group.
type GroupMember struct {
	GroupName string
	Network   *net.IPNet
}

// AppRuleSet contains the parsed AppID rules.
type AppRuleSet struct {
	mu       sync.RWMutex
	rules    []AppRule
	groups   map[string][]*net.IPNet // groupName -> list of member networks
	filePath string
}

// NewAppRuleSet creates a new AppID rule set.
func NewAppRuleSet(filePath string) *AppRuleSet {
	return &AppRuleSet{
		rules:    make([]AppRule, 0),
		groups:   make(map[string][]*net.IPNet),
		filePath: filePath,
	}
}

// Load loads the rules from the file.
func (rs *AppRuleSet) Load() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	file, err := os.Open(rs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, start with empty rules
			return nil
		}
		return fmt.Errorf("failed to open rules file: %w", err)
	}
	defer file.Close()

	rs.rules = make([]AppRule, 0)

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove inline comments
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		// Parse rule: TYPE;GROUP;APP_NAME
		parts := strings.Split(line, ";")
		if len(parts) != 3 {
			return fmt.Errorf("invalid rule format at line %d: expected TYPE;GROUP;APP_NAME", lineNum)
		}

		ruleType := RuleType(strings.ToUpper(strings.TrimSpace(parts[0])))
		groupName := strings.TrimSpace(parts[1])
		appName := strings.ToLower(strings.TrimSpace(parts[2]))

		if ruleType != RuleAllowApp && ruleType != RuleBlockApp {
			return fmt.Errorf("invalid rule type at line %d: %s (expected ALLOW_APP or BLOCK_APP)", lineNum, ruleType)
		}

		if groupName == "" {
			return fmt.Errorf("empty group name at line %d", lineNum)
		}

		if appName == "" {
			return fmt.Errorf("empty app name at line %d", lineNum)
		}

		rs.rules = append(rs.rules, AppRule{
			Type:      ruleType,
			GroupName: groupName,
			AppName:   appName,
		})
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading rules file: %w", err)
	}

	return nil
}

// Reload reloads the rules from the file.
func (rs *AppRuleSet) Reload() error {
	return rs.Load()
}

// SetGroups sets the group membership data (loaded from zid-proxy groups).
func (rs *AppRuleSet) SetGroups(groups map[string][]*net.IPNet) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.groups = groups
}

// FindGroup returns the group name for a given source IP.
func (rs *AppRuleSet) FindGroup(srcIP net.IP) string {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	for groupName, networks := range rs.groups {
		for _, network := range networks {
			if network.Contains(srcIP) {
				return groupName
			}
		}
	}
	return ""
}

// Match checks if there's a matching rule for the given group and app.
// Returns: action (ALLOW/BLOCK), matched (bool)
// Priority: ALLOW_APP > BLOCK_APP
func (rs *AppRuleSet) Match(groupName, appName string) (RuleType, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if groupName == "" || appName == "" {
		return "", false
	}

	appName = strings.ToLower(appName)
	var hasAllow, hasBlock bool

	for _, rule := range rs.rules {
		if rule.GroupName != groupName {
			continue
		}

		// Check if app matches (exact match or wildcard)
		if rule.AppName != appName && rule.AppName != "*" {
			continue
		}

		switch rule.Type {
		case RuleAllowApp:
			hasAllow = true
		case RuleBlockApp:
			hasBlock = true
		}
	}

	// ALLOW has priority over BLOCK
	if hasAllow {
		return RuleAllowApp, true
	}
	if hasBlock {
		return RuleBlockApp, true
	}

	return "", false
}

// MatchForIP combines group lookup and rule matching.
// Returns: action, matched, groupName
func (rs *AppRuleSet) MatchForIP(srcIP net.IP, appName string) (RuleType, bool, string) {
	groupName := rs.FindGroup(srcIP)
	if groupName == "" {
		return "", false, ""
	}

	action, matched := rs.Match(groupName, appName)
	return action, matched, groupName
}

// Count returns the number of loaded rules.
func (rs *AppRuleSet) Count() int {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return len(rs.rules)
}

// ListRules returns all rules (for debugging/display).
func (rs *AppRuleSet) ListRules() []AppRule {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	result := make([]AppRule, len(rs.rules))
	copy(result, rs.rules)
	return result
}
