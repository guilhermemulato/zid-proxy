package appid

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Client is a Unix socket client for communicating with zid-appid daemon.
type Client struct {
	mu         sync.Mutex
	socketPath string
	conn       net.Conn
	reader     *bufio.Reader
	timeout    time.Duration
}

// LookupResult contains the result of an AppID lookup.
type LookupResult struct {
	AppName    string  `json:"app_name"`
	Confidence float32 `json:"confidence"`
	Found      bool    `json:"found"`
}

// StatsResult contains AppID daemon statistics.
type StatsResult struct {
	FlowsTotal   int            `json:"flows_total"`
	AppsDetected map[string]int `json:"apps_detected"`
}

// NewClient creates a new AppID client.
func NewClient(socketPath string, timeout time.Duration) *Client {
	return &Client{
		socketPath: socketPath,
		timeout:    timeout,
	}
}

// connect establishes a connection to the daemon.
func (c *Client) connect() error {
	if c.conn != nil {
		return nil
	}

	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to appid daemon: %w", err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

// Close closes the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.reader = nil
		return err
	}
	return nil
}

// sendCommand sends a command and returns the response.
func (c *Client) sendCommand(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.connect(); err != nil {
		return "", err
	}

	// Set deadline
	c.conn.SetDeadline(time.Now().Add(c.timeout))

	// Send command
	_, err := fmt.Fprintf(c.conn, "%s\n", cmd)
	if err != nil {
		c.conn.Close()
		c.conn = nil
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	response, err := c.reader.ReadString('\n')
	if err != nil {
		c.conn.Close()
		c.conn = nil
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// Lookup queries the daemon for the app associated with a flow.
// Command format: LOOKUP srcIP dstIP proto srcPort dstPort
func (c *Client) Lookup(srcIP, dstIP string, proto uint8, srcPort, dstPort uint16) (*LookupResult, error) {
	protoStr := "TCP"
	if proto == 17 {
		protoStr = "UDP"
	}

	cmd := fmt.Sprintf("LOOKUP %s %s %s %d %d", srcIP, dstIP, protoStr, srcPort, dstPort)
	response, err := c.sendCommand(cmd)
	if err != nil {
		return nil, err
	}

	return c.parseResponse(response)
}

// LookupIP queries the daemon for the most recent app for a source IP.
// Command format: LOOKUP_IP srcIP
func (c *Client) LookupIP(srcIP string) (*LookupResult, error) {
	cmd := fmt.Sprintf("LOOKUP_IP %s", srcIP)
	response, err := c.sendCommand(cmd)
	if err != nil {
		return nil, err
	}

	return c.parseResponse(response)
}

// parseResponse parses the daemon response.
// Formats:
//   - OK app_name [confidence]
//   - UNKNOWN
//   - ERROR message
func (c *Client) parseResponse(response string) (*LookupResult, error) {
	parts := strings.Fields(response)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty response from daemon")
	}

	switch parts[0] {
	case "OK":
		result := &LookupResult{Found: true}
		if len(parts) >= 2 {
			result.AppName = parts[1]
		}
		if len(parts) >= 3 {
			fmt.Sscanf(parts[2], "%f", &result.Confidence)
		} else {
			result.Confidence = 1.0
		}
		return result, nil

	case "UNKNOWN":
		return &LookupResult{Found: false}, nil

	case "ERROR":
		errMsg := "unknown error"
		if len(parts) > 1 {
			errMsg = strings.Join(parts[1:], " ")
		}
		return nil, fmt.Errorf("daemon error: %s", errMsg)

	default:
		return nil, fmt.Errorf("unexpected response: %s", response)
	}
}

// Stats retrieves statistics from the daemon.
func (c *Client) Stats() (*StatsResult, error) {
	response, err := c.sendCommand("STATS")
	if err != nil {
		return nil, err
	}

	var result StatsResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse stats: %w", err)
	}

	return &result, nil
}

// Apps retrieves the list of supported applications.
func (c *Client) Apps() ([]string, error) {
	response, err := c.sendCommand("APPS")
	if err != nil {
		return nil, err
	}

	var apps []string
	if err := json.Unmarshal([]byte(response), &apps); err != nil {
		return nil, fmt.Errorf("failed to parse apps list: %w", err)
	}

	return apps, nil
}

// Ping checks if the daemon is responsive.
func (c *Client) Ping() error {
	_, err := c.sendCommand("PING")
	return err
}

// IsAvailable checks if the daemon socket exists and is accessible.
func (c *Client) IsAvailable() bool {
	err := c.Ping()
	return err == nil
}
