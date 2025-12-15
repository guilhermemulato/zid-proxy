package sni

import (
	"testing"
)

// Sample ClientHello message (without TLS record header) for testing
// This is a minimal ClientHello with SNI extension for "localhost"
var sampleClientHello = buildSampleClientHello()

func buildSampleClientHello() []byte {
	// Build SNI extension for "localhost"
	hostname := []byte("localhost")
	sniExtension := []byte{
		0x00, 0x00, // Type: server_name
	}
	// SNI extension data
	sniData := []byte{
		0x00, byte(len(hostname) + 3), // Server Name List Length
		0x00,                   // Name Type: host_name
		0x00, byte(len(hostname)), // Name Length
	}
	sniData = append(sniData, hostname...)

	// Extension length
	sniExtension = append(sniExtension, 0x00, byte(len(sniData)))
	sniExtension = append(sniExtension, sniData...)

	// Extensions block
	extensions := sniExtension
	extensionsLen := len(extensions)

	// Build the body after handshake header
	body := []byte{
		// Client Version: TLS 1.2
		0x03, 0x03,
	}
	// Client Random (32 bytes)
	body = append(body, make([]byte, 32)...)
	// Session ID Length: 0
	body = append(body, 0x00)
	// Cipher Suites Length: 2, one cipher suite
	body = append(body, 0x00, 0x02, 0x00, 0x9c)
	// Compression Methods Length: 1, null compression
	body = append(body, 0x01, 0x00)
	// Extensions length (2 bytes)
	body = append(body, byte(extensionsLen>>8), byte(extensionsLen))
	// Extensions
	body = append(body, extensions...)

	// Build handshake message
	bodyLen := len(body)
	msg := []byte{
		0x01, // ClientHello
		byte(bodyLen >> 16), byte(bodyLen >> 8), byte(bodyLen), // Length
	}
	msg = append(msg, body...)

	return msg
}

func TestExtractSNI(t *testing.T) {
	hostname, err := ExtractSNI(sampleClientHello)
	if err != nil {
		t.Fatalf("ExtractSNI failed: %v", err)
	}
	if hostname != "localhost" {
		t.Errorf("Expected hostname 'localhost', got '%s'", hostname)
	}
}

func TestExtractSNI_NotClientHello(t *testing.T) {
	data := []byte{0x02, 0x00, 0x00, 0x00} // ServerHello type
	_, err := ExtractSNI(data)
	if err != ErrNotClientHello {
		t.Errorf("Expected ErrNotClientHello, got %v", err)
	}
}

func TestExtractSNI_EmptyData(t *testing.T) {
	_, err := ExtractSNI([]byte{})
	if err != ErrBufferTooSmall {
		t.Errorf("Expected ErrBufferTooSmall, got %v", err)
	}
}

func TestExtractSNI_NoExtensions(t *testing.T) {
	// Minimal ClientHello without extensions
	data := []byte{
		0x01,             // ClientHello
		0x00, 0x00, 0x26, // Length
		0x03, 0x03, // Version
		// Random (32 bytes)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00,             // Session ID length: 0
		0x00, 0x02,       // Cipher suites length: 2
		0x00, 0x9c,       // One cipher suite
		0x01,             // Compression methods length: 1
		0x00,             // Null compression
	}
	_, err := ExtractSNI(data)
	if err != ErrNoSNI {
		t.Errorf("Expected ErrNoSNI, got %v", err)
	}
}

func TestParseSNIExtension(t *testing.T) {
	// SNI extension data for "test.example.org"
	data := []byte{
		0x00, 0x13, // Server Name List Length: 19
		0x00,       // Name Type: host_name
		0x00, 0x10, // Name Length: 16
		// "test.example.org"
		0x74, 0x65, 0x73, 0x74, 0x2e, 0x65, 0x78, 0x61,
		0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x6f, 0x72, 0x67,
	}

	hostname, err := parseSNIExtension(data)
	if err != nil {
		t.Fatalf("parseSNIExtension failed: %v", err)
	}
	if hostname != "test.example.org" {
		t.Errorf("Expected 'test.example.org', got '%s'", hostname)
	}
}

func TestParseSNIExtension_EmptyData(t *testing.T) {
	_, err := parseSNIExtension([]byte{})
	if err != ErrInvalidSNI {
		t.Errorf("Expected ErrInvalidSNI, got %v", err)
	}
}

func BenchmarkExtractSNI(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ExtractSNI(sampleClientHello)
	}
}
