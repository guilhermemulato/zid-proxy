package main

// This file contains the SNI-based detection mode.
// This is a fallback mode that uses hostname matching instead of full DPI.
// For full DPI with nDPI library, see the bridge mode implementation.

import (
	"log"
	"net"
	"time"

	"github.com/guilherme/zid-proxy/internal/appid"
)

// SNIHandler handles SNI-based detection requests.
// It registers detected apps in the flow cache based on hostname lookups.
type SNIHandler struct {
	flowCache *appid.FlowCache
	detector  *appid.Detector
}

// NewSNIHandler creates a new SNI handler.
func NewSNIHandler(flowCache *appid.FlowCache, detector *appid.Detector) *SNIHandler {
	return &SNIHandler{
		flowCache: flowCache,
		detector:  detector,
	}
}

// RegisterFlow registers a flow with detected app based on hostname.
// This is called by zid-proxy when it extracts SNI from a connection.
func (h *SNIHandler) RegisterFlow(srcIP, dstIP net.IP, srcPort, dstPort uint16, hostname string) string {
	// Detect app by hostname
	app, confidence := h.detector.DetectByHostname(hostname)
	if app == nil {
		return ""
	}

	now := time.Now()

	// Create flow key
	key := appid.FlowKey{
		SrcIP:    srcIP.String(),
		DstIP:    dstIP.String(),
		SrcPort:  srcPort,
		DstPort:  dstPort,
		Protocol: 6, // TCP
	}

	// Register in cache
	flow := &appid.FlowInfo{
		Key:         key,
		AppName:     app.Name,
		AppCategory: string(app.Category),
		Confidence:  confidence,
		FirstSeen:   now,
		LastSeen:    now,
	}

	h.flowCache.Set(flow)

	log.Printf("Detected app: %s (%.0f%%) for %s -> %s:%d",
		app.Name, confidence*100, srcIP, hostname, dstPort)

	return app.Name
}

// Note: In a full implementation, there would be additional modes:
//
// 1. Bridge Mode (inline DPI):
//    - Uses libpcap or AF_PACKET to capture traffic
//    - Runs nDPI on packet payloads
//    - Can detect apps even without SNI (encrypted traffic analysis)
//
// 2. Mirror Mode (passive DPI):
//    - Receives mirrored traffic from a switch SPAN port
//    - Non-intrusive, doesn't affect traffic flow
//    - Uses nDPI for detection
//
// 3. Netfilter Queue Mode:
//    - Uses NFQUEUE to receive packets from iptables
//    - Can make allow/block decisions inline
//    - Requires kernel support
//
// For now, we use SNI mode which works with the existing zid-proxy
// by matching hostnames to known app patterns.
