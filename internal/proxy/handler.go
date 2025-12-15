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
		if err == sni.ErrNotTLS || err == sni.ErrNoSNI {
			log.Printf("Non-TLS or no SNI from %s, blocking", clientIP)
			h.sendRST()
			return
		}
		log.Printf("Failed to read ClientHello from %s: %v", clientIP, err)
		return
	}

	// Clear deadline
	h.clientConn.SetReadDeadline(time.Time{})

	// Match against rules
	action, matched := h.server.rules.Match(clientIP, hostname)

	// Convert to logger action
	var logAction logger.Action
	if action == rules.RuleBlock {
		logAction = logger.ActionBlock
	} else {
		logAction = logger.ActionAllow
	}

	// Log the connection
	h.server.logger.LogConnection(clientIP.String(), hostname, logAction)

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
