package appid

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

// Server is the Unix socket server for the AppID daemon.
type Server struct {
	socketPath string
	listener   net.Listener
	flowCache  *FlowCache
	detector   *Detector
	wg         sync.WaitGroup
	quit       chan struct{}
}

// NewServer creates a new AppID server.
func NewServer(socketPath string, flowCache *FlowCache, detector *Detector) *Server {
	return &Server{
		socketPath: socketPath,
		flowCache:  flowCache,
		detector:   detector,
		quit:       make(chan struct{}),
	}
}

// Start starts the Unix socket server.
func (s *Server) Start() error {
	// Remove existing socket if present
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}

	// Set socket permissions
	if err := os.Chmod(s.socketPath, 0660); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.listener = listener
	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop stops the server.
func (s *Server) Stop() error {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()

	// Remove socket file
	os.Remove(s.socketPath)
	return nil
}

// acceptLoop accepts incoming connections.
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Printf("AppID server: accept error: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a client connection.
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		select {
		case <-s.quit:
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			return // Client disconnected
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		response := s.handleCommand(line)
		fmt.Fprintf(conn, "%s\n", response)
	}
}

// handleCommand processes a command and returns the response.
func (s *Server) handleCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return "ERROR empty command"
	}

	switch strings.ToUpper(parts[0]) {
	case "PING":
		return "PONG"

	case "LOOKUP":
		return s.handleLookup(parts[1:])

	case "LOOKUP_IP":
		return s.handleLookupIP(parts[1:])

	case "LOOKUP_HOST":
		return s.handleLookupHost(parts[1:])

	case "STATS":
		return s.handleStats()

	case "APPS":
		return s.handleApps()

	default:
		return fmt.Sprintf("ERROR unknown command: %s", parts[0])
	}
}

// handleLookup handles: LOOKUP srcIP dstIP proto srcPort dstPort
func (s *Server) handleLookup(args []string) string {
	if len(args) < 5 {
		return "ERROR usage: LOOKUP srcIP dstIP proto srcPort dstPort"
	}

	srcIP := args[0]
	dstIP := args[1]
	proto := args[2]
	var srcPort, dstPort uint16
	fmt.Sscanf(args[3], "%d", &srcPort)
	fmt.Sscanf(args[4], "%d", &dstPort)

	var protoNum uint8 = 6 // TCP
	if strings.ToUpper(proto) == "UDP" {
		protoNum = 17
	}

	key := FlowKey{
		SrcIP:    srcIP,
		DstIP:    dstIP,
		SrcPort:  srcPort,
		DstPort:  dstPort,
		Protocol: protoNum,
	}

	flow, found := s.flowCache.Get(key)
	if !found || flow.AppName == "" {
		return "UNKNOWN"
	}

	return fmt.Sprintf("OK %s %.2f", flow.AppName, flow.Confidence)
}

// handleLookupIP handles: LOOKUP_IP srcIP
func (s *Server) handleLookupIP(args []string) string {
	if len(args) < 1 {
		return "ERROR usage: LOOKUP_IP srcIP"
	}

	srcIP := net.ParseIP(args[0])
	if srcIP == nil {
		return "ERROR invalid IP address"
	}

	flow, found := s.flowCache.GetByIP(srcIP)
	if !found || flow.AppName == "" {
		return "UNKNOWN"
	}

	return fmt.Sprintf("OK %s %.2f", flow.AppName, flow.Confidence)
}

// handleLookupHost handles: LOOKUP_HOST hostname
// This is used for SNI-based detection when full DPI isn't available.
func (s *Server) handleLookupHost(args []string) string {
	if len(args) < 1 {
		return "ERROR usage: LOOKUP_HOST hostname"
	}

	hostname := args[0]
	app, confidence := s.detector.DetectByHostname(hostname)
	if app == nil {
		return "UNKNOWN"
	}

	return fmt.Sprintf("OK %s %.2f", app.Name, confidence)
}

// handleStats returns statistics.
func (s *Server) handleStats() string {
	stats := s.flowCache.Stats()
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Sprintf("ERROR failed to marshal stats: %v", err)
	}
	return string(data)
}

// handleApps returns the list of supported applications.
func (s *Server) handleApps() string {
	apps := s.detector.ListAppNames()
	data, err := json.Marshal(apps)
	if err != nil {
		return fmt.Sprintf("ERROR failed to marshal apps: %v", err)
	}
	return string(data)
}
