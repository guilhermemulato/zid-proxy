package proxy

import (
	"bytes"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/guilherme/zid-proxy/internal/logger"
	"github.com/guilherme/zid-proxy/internal/rules"
	"github.com/guilherme/zid-proxy/internal/sni"
)

// Handler processes a single connection
type Handler struct {
	server       *Server
	clientConn   net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// Handle processes the connection
func (h *Handler) Handle() {
	// Get client IP
	clientAddr := h.clientConn.RemoteAddr().(*net.TCPAddr)
	clientIP := clientAddr.IP

	// Set read deadline for ClientHello
	h.clientConn.SetReadDeadline(time.Now().Add(h.readTimeout))

	// Extract SNI from ClientHello
	hostname, clientHello, err := sni.PeekClientHello(h.clientConn)
	if err != nil {
		if err == sni.ErrNotTLS {
			log.Printf("Non-TLS connection from %s, blocking", clientIP)
			h.sendRST()
			return
		}
		if err == sni.ErrNoSNI {
			// Allow connections without SNI from private IP ranges
			// This enables access to local resources by IP (e.g., https://192.168.1.1)
			if isPrivateIP(clientIP) {
				log.Printf("No SNI from private IP %s, allowing", clientIP)
				h.proxyConnectionNoSNI(clientAddr)
				return
			}
			log.Printf("No SNI from public IP %s, blocking", clientIP)
			h.sendRST()
			return
		}
		log.Printf("Failed to read ClientHello from %s: %v", clientIP, err)
		return
	}

	// Clear deadline
	h.clientConn.SetReadDeadline(time.Time{})

	// Match against rules
	action, matched, groupName := h.server.rules.Match(clientIP, hostname)

	// Convert to logger action
	var logAction logger.Action
	if action == rules.RuleBlock {
		logAction = logger.ActionBlock
	} else {
		logAction = logger.ActionAllow
	}

	// Log the connection
	h.server.logger.LogConnection(clientIP.String(), hostname, groupName, logAction)

	if matched {
		log.Printf("%s | %s -> %s | %s (matched rule)", clientIP, hostname, action, logAction)
	} else {
		log.Printf("%s | %s -> %s | %s (default)", clientIP, hostname, action, logAction)
	}

	if action == rules.RuleBlock {
		h.sendRST()
		return
	}

	// Allow: proxy the connection
	h.proxyConnection(hostname, clientHello)
}

// sendRST sends a TCP RST by setting linger to 0 before closing
func (h *Handler) sendRST() {
	if tcpConn, ok := h.clientConn.(*net.TCPConn); ok {
		tcpConn.SetLinger(0)
	}
	// Connection will be closed by deferred Close in handleConnection
}

// proxyConnection establishes a connection to the upstream server and proxies traffic
func (h *Handler) proxyConnection(hostname string, clientHello []byte) {
	// Connect to the original destination (the hostname from SNI)
	// We connect to port 443 as this is HTTPS traffic
	upstreamAddr := net.JoinHostPort(hostname, "443")

	dialer := &net.Dialer{
		Timeout: h.writeTimeout,
	}

	upstreamConn, err := dialer.DialContext(h.server.ctx, "tcp", upstreamAddr)
	if err != nil {
		log.Printf("Failed to connect to upstream %s: %v", upstreamAddr, err)
		h.sendRST()
		return
	}
	defer upstreamConn.Close()

	// Send the captured ClientHello to upstream
	upstreamConn.SetWriteDeadline(time.Now().Add(h.writeTimeout))
	if _, err := upstreamConn.Write(clientHello); err != nil {
		log.Printf("Failed to send ClientHello to upstream %s: %v", upstreamAddr, err)
		return
	}
	upstreamConn.SetWriteDeadline(time.Time{})

	// Bidirectional proxy
	h.bidirectionalCopy(h.clientConn, upstreamConn)
}

// isPrivateIP checks if an IP belongs to a private network (RFC 1918 + loopback)
func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",     // RFC 1918
		"172.16.0.0/12",  // RFC 1918
		"192.168.0.0/16", // RFC 1918
		"127.0.0.0/8",    // Loopback
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 private
	}

	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// proxyConnectionNoSNI handles connections from private IPs without SNI
//
// LIMITATION: True transparent proxying without SNI requires access to the
// original destination IP after NAT redirect. This information is available via:
//   - SO_ORIGINAL_DST socket option on Linux/iptables
//   - Divert sockets on FreeBSD/pf
//
// This proxy uses a simple TCP listen socket, which doesn't have access to the
// original destination after NAT Port Forward. The clientAddr parameter contains
// the CLIENT's IP address, not the destination the client was trying to reach.
//
// Therefore, this implementation cannot proxy connections without SNI.
//
// WORKAROUND (Recommended):
// Exclude specific IPs from NAT redirect in pfSense:
//   1. Firewall > NAT > Port Forward
//   2. Edit the rule that redirects port 443 to proxy
//   3. Destination: Invert match (NOT) → Single host → 192.168.1.1
//   4. Save & Apply
//
// This allows direct access to pfSense GUI and other local services by IP.
//
// Alternative: Access local services via hostname instead of IP
// (e.g., https://pfsense.local instead of https://192.168.1.1)
func (h *Handler) proxyConnectionNoSNI(clientAddr *net.TCPAddr) {
	// Close connection gracefully (no RST packet)
	// This is better than sending RST for private IP connections
	h.clientConn.Close()
}

// bidirectionalCopy copies data between two connections in both directions
func (h *Handler) bidirectionalCopy(client, upstream net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Copy from client to upstream
	go func() {
		defer wg.Done()
		io.Copy(upstream, client)
		// Signal upstream that we're done sending
		if tcpConn, ok := upstream.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	// Copy from upstream to client
	go func() {
		defer wg.Done()
		io.Copy(client, upstream)
		// Signal client that we're done sending
		if tcpConn, ok := client.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	wg.Wait()
}

// MultiReader wraps multiple readers for replaying ClientHello
type MultiReader struct {
	readers []io.Reader
	current int
}

// NewMultiReader creates a reader that first reads from clientHello, then from conn
func NewMultiReader(clientHello []byte, conn net.Conn) io.Reader {
	return io.MultiReader(bytes.NewReader(clientHello), conn)
}
