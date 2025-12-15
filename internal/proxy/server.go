package proxy

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/guilherme/zid-proxy/internal/logger"
	"github.com/guilherme/zid-proxy/internal/rules"
)

// Config holds server configuration
type Config struct {
	ListenAddr   string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		ListenAddr:   ":443",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

// Server is the main proxy server
type Server struct {
	config   Config
	rules    *rules.RuleSet
	logger   logger.Interface
	listener net.Listener

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new Server
func New(cfg Config, ruleSet *rules.RuleSet, log logger.Interface) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		config: cfg,
		rules:  ruleSet,
		logger: log,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the proxy server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.ListenAddr, err)
	}
	s.listener = listener

	log.Printf("zid-proxy listening on %s", s.config.ListenAddr)

	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	log.Println("Shutting down server...")

	// Cancel context to signal handlers to stop
	s.cancel()

	// Close listener to stop accepting new connections
	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for all handlers to finish
	s.wg.Wait()

	// Flush and close logger
	if err := s.logger.Flush(); err != nil {
		log.Printf("Warning: failed to flush logger: %v", err)
	}

	log.Println("Server stopped")
	return nil
}

// Reload triggers a reload of the rules
func (s *Server) Reload() error {
	log.Println("Reloading rules...")
	if err := s.rules.Reload(); err != nil {
		return fmt.Errorf("failed to reload rules: %w", err)
	}
	log.Printf("Rules reloaded successfully (%d rules)", s.rules.RuleCount())
	return nil
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return // Server is shutting down
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection processes a single connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	handler := &Handler{
		server:       s,
		clientConn:   conn,
		readTimeout:  s.config.ReadTimeout,
		writeTimeout: s.config.WriteTimeout,
	}

	handler.Handle()
}

// ListenAddr returns the actual listen address (useful when port 0 is used)
func (s *Server) ListenAddr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.config.ListenAddr
}
