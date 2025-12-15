package rules

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// RuleType represents the action to take for a matching rule
type RuleType string

const (
	RuleAllow RuleType = "ALLOW"
	RuleBlock RuleType = "BLOCK"
)

// Rule represents a single access rule
type Rule struct {
	Type     RuleType
	SourceIP *net.IPNet
	Hostname string // Supports wildcards like *.example.com
}

// RuleSet manages a collection of access rules
type RuleSet struct {
	mu       sync.RWMutex
	rules    []Rule
	filePath string
}

// NewRuleSet creates a new RuleSet that loads rules from the given file path
func NewRuleSet(filePath string) *RuleSet {
	return &RuleSet{
		filePath: filePath,
		rules:    make([]Rule, 0),
	}
}

// Load reads and parses rules from the configured file
func (rs *RuleSet) Load() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	return rs.loadInternal()
}

// Reload reloads rules from the file (thread-safe)
func (rs *RuleSet) Reload() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Clear existing rules
	rs.rules = rs.rules[:0]

	return rs.loadInternal()
}

// loadInternal loads rules without locking (caller must hold lock)
func (rs *RuleSet) loadInternal() error {
	file, err := os.Open(rs.filePath)
	if err != nil {
		return fmt.Errorf("failed to open rules file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		rule, err := parseRule(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}

		rs.rules = append(rs.rules, rule)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading rules file: %w", err)
	}

	return nil
}

// parseRule parses a single rule line in format: TYPE;IP_OR_CIDR;HOSTNAME
func parseRule(line string) (Rule, error) {
	parts := strings.Split(line, ";")
	if len(parts) != 3 {
		return Rule{}, fmt.Errorf("invalid rule format: expected TYPE;IP_OR_CIDR;HOSTNAME")
	}

	ruleType := strings.ToUpper(strings.TrimSpace(parts[0]))
	ipStr := strings.TrimSpace(parts[1])
	hostname := strings.ToLower(strings.TrimSpace(parts[2]))

	// Validate rule type
	var rt RuleType
	switch ruleType {
	case "ALLOW":
		rt = RuleAllow
	case "BLOCK":
		rt = RuleBlock
	default:
		return Rule{}, fmt.Errorf("invalid rule type: %s (must be ALLOW or BLOCK)", ruleType)
	}

	// Parse IP/CIDR
	ipNet, err := parseIPOrCIDR(ipStr)
	if err != nil {
		return Rule{}, fmt.Errorf("invalid IP/CIDR: %w", err)
	}

	return Rule{
		Type:     rt,
		SourceIP: ipNet,
		Hostname: hostname,
	}, nil
}

// parseIPOrCIDR parses an IP address or CIDR notation
func parseIPOrCIDR(s string) (*net.IPNet, error) {
	// Try CIDR first
	if strings.Contains(s, "/") {
		_, ipNet, err := net.ParseCIDR(s)
		if err != nil {
			return nil, err
		}
		return ipNet, nil
	}

	// Try as plain IP
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", s)
	}

	// Convert to /32 or /128 CIDR
	bits := 32
	if ip.To4() == nil {
		bits = 128
	}

	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(bits, bits),
	}, nil
}

// Match checks if a connection from srcIP to hostname matches any rule
// Returns the action to take and whether a rule was matched
// Priority: ALLOW > BLOCK
// Default: ALLOW if no rule matches
func (rs *RuleSet) Match(srcIP net.IP, hostname string) (action RuleType, matched bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	hostname = strings.ToLower(hostname)
	var blockMatched bool

	for _, rule := range rs.rules {
		if rs.matchRule(rule, srcIP, hostname) {
			if rule.Type == RuleAllow {
				// ALLOW takes priority - return immediately
				return RuleAllow, true
			}
			blockMatched = true
		}
	}

	if blockMatched {
		return RuleBlock, true
	}

	// Default: ALLOW if no rule matches
	return RuleAllow, false
}

// matchRule checks if a single rule matches the given connection
func (rs *RuleSet) matchRule(rule Rule, srcIP net.IP, hostname string) bool {
	// Check IP match
	if !rule.SourceIP.Contains(srcIP) {
		return false
	}

	// Check hostname match (with wildcard support)
	return matchHostname(rule.Hostname, hostname)
}

// matchHostname matches a hostname against a pattern with wildcard support
// Supports: *.example.com, example.com, *.sub.example.com
func matchHostname(pattern, hostname string) bool {
	if pattern == hostname {
		return true
	}

	// Wildcard matching
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // Remove *, keep .example.com
		// Match exact suffix or the domain itself without subdomain
		if strings.HasSuffix(hostname, suffix) {
			return true
		}
		// Also match the bare domain (e.g., "*.example.com" should match "example.com")
		if hostname == pattern[2:] {
			return true
		}
	}

	return false
}

// RuleCount returns the number of loaded rules
func (rs *RuleSet) RuleCount() int {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return len(rs.rules)
}

// String returns a string representation of the rules for debugging
func (rs *RuleSet) String() string {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("RuleSet (%d rules):\n", len(rs.rules)))
	for i, r := range rs.rules {
		sb.WriteString(fmt.Sprintf("  %d: %s %s %s\n", i+1, r.Type, r.SourceIP, r.Hostname))
	}
	return sb.String()
}
