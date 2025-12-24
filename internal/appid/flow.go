// Package appid provides Deep Packet Inspection (DPI) functionality for application identification.
// It uses nDPI library to detect applications like Netflix, YouTube, Facebook, etc.
package appid

import (
	"net"
	"sync"
	"time"
)

// FlowKey represents a unique identifier for a network flow (5-tuple).
type FlowKey struct {
	SrcIP    string
	DstIP    string
	SrcPort  uint16
	DstPort  uint16
	Protocol uint8 // 6=TCP, 17=UDP
}

// FlowInfo contains information about a detected flow.
type FlowInfo struct {
	Key         FlowKey
	AppName     string    // Detected application name (e.g., "netflix", "youtube")
	AppCategory string    // Category (e.g., "streaming_media", "social_networking")
	Confidence  float32   // Detection confidence (0.0 to 1.0)
	FirstSeen   time.Time // When the flow was first seen
	LastSeen    time.Time // Last activity timestamp
	BytesIn     uint64    // Bytes received
	BytesOut    uint64    // Bytes sent
	PacketCount uint64    // Total packets processed
}

// FlowCache maintains a thread-safe cache of detected flows.
type FlowCache struct {
	mu       sync.RWMutex
	flows    map[FlowKey]*FlowInfo
	maxFlows int
	ttl      time.Duration
}

// NewFlowCache creates a new flow cache with the specified parameters.
func NewFlowCache(maxFlows int, ttl time.Duration) *FlowCache {
	return &FlowCache{
		flows:    make(map[FlowKey]*FlowInfo),
		maxFlows: maxFlows,
		ttl:      ttl,
	}
}

// Get retrieves flow information by key.
func (c *FlowCache) Get(key FlowKey) (*FlowInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	flow, ok := c.flows[key]
	return flow, ok
}

// GetByIP retrieves the most recent flow for a source IP.
func (c *FlowCache) GetByIP(srcIP net.IP) (*FlowInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	srcIPStr := srcIP.String()
	var mostRecent *FlowInfo

	for _, flow := range c.flows {
		if flow.Key.SrcIP == srcIPStr {
			if mostRecent == nil || flow.LastSeen.After(mostRecent.LastSeen) {
				mostRecent = flow
			}
		}
	}

	return mostRecent, mostRecent != nil
}

// Set adds or updates a flow in the cache.
func (c *FlowCache) Set(flow *FlowInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict old entries
	if len(c.flows) >= c.maxFlows {
		c.evictOldest()
	}

	c.flows[flow.Key] = flow
}

// Update updates an existing flow's statistics.
func (c *FlowCache) Update(key FlowKey, bytesIn, bytesOut uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if flow, ok := c.flows[key]; ok {
		flow.LastSeen = time.Now()
		flow.BytesIn += bytesIn
		flow.BytesOut += bytesOut
		flow.PacketCount++
	}
}

// SetApp sets the detected application for a flow.
func (c *FlowCache) SetApp(key FlowKey, appName, category string, confidence float32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if flow, ok := c.flows[key]; ok {
		flow.AppName = appName
		flow.AppCategory = category
		flow.Confidence = confidence
	}
}

// evictOldest removes the oldest entries to make room for new ones.
// Must be called with lock held.
func (c *FlowCache) evictOldest() {
	// Find and remove entries that exceed TTL
	now := time.Now()
	for key, flow := range c.flows {
		if now.Sub(flow.LastSeen) > c.ttl {
			delete(c.flows, key)
		}
	}

	// If still over limit, remove oldest entries
	if len(c.flows) >= c.maxFlows {
		var oldestKey FlowKey
		var oldestTime time.Time

		for key, flow := range c.flows {
			if oldestTime.IsZero() || flow.LastSeen.Before(oldestTime) {
				oldestKey = key
				oldestTime = flow.LastSeen
			}
		}

		if !oldestTime.IsZero() {
			delete(c.flows, oldestKey)
		}
	}
}

// GC removes expired entries from the cache.
func (c *FlowCache) GC(now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for key, flow := range c.flows {
		if now.Sub(flow.LastSeen) > c.ttl {
			delete(c.flows, key)
			removed++
		}
	}

	return removed
}

// Stats returns statistics about the cache.
func (c *FlowCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	appCounts := make(map[string]int)
	for _, flow := range c.flows {
		if flow.AppName != "" {
			appCounts[flow.AppName]++
		}
	}

	return map[string]interface{}{
		"flows_total":   len(c.flows),
		"apps_detected": appCounts,
	}
}

// List returns all flows (for debugging).
func (c *FlowCache) List() []*FlowInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*FlowInfo, 0, len(c.flows))
	for _, flow := range c.flows {
		result = append(result, flow)
	}
	return result
}
