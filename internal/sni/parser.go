package sni

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

const (
	// TLS record types
	recordTypeHandshake = 0x16

	// Handshake message types
	handshakeTypeClientHello = 0x01

	// Extension types
	extensionServerName = 0x0000

	// Server name types
	serverNameTypeHostname = 0x00

	// Size limits
	maxTLSRecordSize = 16384 + 5 // 16KB + header
	tlsHeaderSize    = 5
)

var (
	ErrNotTLS          = errors.New("not a TLS handshake")
	ErrNotClientHello  = errors.New("not a ClientHello message")
	ErrNoSNI           = errors.New("no SNI extension found")
	ErrInvalidSNI      = errors.New("invalid SNI extension")
	ErrBufferTooSmall  = errors.New("buffer too small")
	ErrRecordTooLarge  = errors.New("TLS record too large")
)

// PeekClientHello reads the TLS ClientHello from a connection without consuming it.
// Returns the SNI hostname and a new reader that includes the peeked data.
func PeekClientHello(conn net.Conn) (hostname string, clientHello []byte, err error) {
	// First, read the TLS record header to know how much to read
	header := make([]byte, tlsHeaderSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		return "", nil, err
	}

	// Validate it's a TLS handshake
	if header[0] != recordTypeHandshake {
		return "", header, ErrNotTLS
	}

	// Get the record length
	recordLen := int(binary.BigEndian.Uint16(header[3:5]))
	if recordLen > maxTLSRecordSize-tlsHeaderSize {
		return "", header, ErrRecordTooLarge
	}

	// Read the full record
	record := make([]byte, recordLen)
	if _, err := io.ReadFull(conn, record); err != nil {
		return "", header, err
	}

	// Combine header and record for replay
	clientHello = make([]byte, tlsHeaderSize+recordLen)
	copy(clientHello, header)
	copy(clientHello[tlsHeaderSize:], record)

	// Extract SNI from the record
	hostname, err = ExtractSNI(record)
	if err != nil {
		return "", clientHello, err
	}

	return hostname, clientHello, nil
}

// ExtractSNI extracts the server name from a TLS ClientHello message body.
// The data should be the handshake message without the TLS record header.
func ExtractSNI(data []byte) (string, error) {
	if len(data) < 1 {
		return "", ErrBufferTooSmall
	}

	// Check handshake type
	if data[0] != handshakeTypeClientHello {
		return "", ErrNotClientHello
	}

	// Skip handshake header: type(1) + length(3)
	if len(data) < 4 {
		return "", ErrBufferTooSmall
	}
	pos := 4

	// Skip client version (2 bytes)
	pos += 2
	if pos > len(data) {
		return "", ErrBufferTooSmall
	}

	// Skip client random (32 bytes)
	pos += 32
	if pos > len(data) {
		return "", ErrBufferTooSmall
	}

	// Skip session ID (variable length)
	if pos >= len(data) {
		return "", ErrBufferTooSmall
	}
	sessionIDLen := int(data[pos])
	pos += 1 + sessionIDLen
	if pos > len(data) {
		return "", ErrBufferTooSmall
	}

	// Skip cipher suites (variable length)
	if pos+2 > len(data) {
		return "", ErrBufferTooSmall
	}
	cipherSuitesLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
	pos += 2 + cipherSuitesLen
	if pos > len(data) {
		return "", ErrBufferTooSmall
	}

	// Skip compression methods (variable length)
	if pos >= len(data) {
		return "", ErrBufferTooSmall
	}
	compressionMethodsLen := int(data[pos])
	pos += 1 + compressionMethodsLen
	if pos > len(data) {
		return "", ErrBufferTooSmall
	}

	// Check if extensions are present
	if pos+2 > len(data) {
		return "", ErrNoSNI
	}

	// Get extensions length
	extensionsLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
	pos += 2
	extensionsEnd := pos + extensionsLen
	if extensionsEnd > len(data) {
		return "", ErrBufferTooSmall
	}

	// Parse extensions
	for pos+4 <= extensionsEnd {
		extType := binary.BigEndian.Uint16(data[pos : pos+2])
		extLen := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		pos += 4

		if pos+extLen > extensionsEnd {
			return "", ErrBufferTooSmall
		}

		if extType == extensionServerName {
			return parseSNIExtension(data[pos : pos+extLen])
		}

		pos += extLen
	}

	return "", ErrNoSNI
}

// parseSNIExtension parses the SNI extension data and returns the hostname.
func parseSNIExtension(data []byte) (string, error) {
	if len(data) < 2 {
		return "", ErrInvalidSNI
	}

	// Get server name list length
	listLen := int(binary.BigEndian.Uint16(data[0:2]))
	if listLen+2 > len(data) {
		return "", ErrInvalidSNI
	}

	pos := 2
	listEnd := pos + listLen

	for pos+3 <= listEnd {
		nameType := data[pos]
		nameLen := int(binary.BigEndian.Uint16(data[pos+1 : pos+3]))
		pos += 3

		if pos+nameLen > listEnd {
			return "", ErrInvalidSNI
		}

		if nameType == serverNameTypeHostname {
			return string(data[pos : pos+nameLen]), nil
		}

		pos += nameLen
	}

	return "", ErrNoSNI
}
