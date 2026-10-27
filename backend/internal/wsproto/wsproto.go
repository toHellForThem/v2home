// Package wsproto implements the small subset of RFC 6455 that v2home needs:
// a server-side handshake, text/binary frames, ping/pong and close frames.
//
// It is intentionally dependency-free. If the tunnel engine ever needs
// compression, binary streaming or a battle-tested client, replace this
// package with github.com/coder/websocket or github.com/gorilla/websocket.
package wsproto

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// GUID is the magic value from RFC 6455 section 1.3.
	GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	// MaxMessageSize caps a single (possibly fragmented) message.
	MaxMessageSize = 1 << 20 // 1 MiB
)

// Opcodes.
const (
	OpContinuation byte = 0x0
	OpText         byte = 0x1
	OpBinary       byte = 0x2
	OpClose        byte = 0x8
	OpPing         byte = 0x9
	OpPong         byte = 0xA
)

// CloseNormal is the RFC 6455 "normal closure" status code.
const CloseNormal uint16 = 1000

var (
	ErrBadHandshake = errors.New("wsproto: bad handshake")
	ErrProtocol     = errors.New("wsproto: protocol error")
	ErrTooLarge     = errors.New("wsproto: message too large")
)

// AcceptKey computes the Sec-WebSocket-Accept value for a client key.
func AcceptKey(key string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, key)
	_, _ = io.WriteString(h, GUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Upgrade validates the handshake, hijacks the connection and writes the
// 101 Switching Protocols response.
//
// On a validation error Upgrade writes an HTTP 400 response itself, so the
// caller must not write to w after a non-nil error.
func Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	if err := validateHandshake(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, err
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket: hijacking not supported", http.StatusInternalServerError)
		return nil, errors.New("wsproto: response writer does not support hijacking")
	}

	netConn, brw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("wsproto: hijack: %w", err)
	}

	// The HTTP server may have set deadlines; clear them for the long-lived
	// WebSocket connection.
	_ = netConn.SetDeadline(time.Time{})

	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + AcceptKey(r.Header.Get("Sec-WebSocket-Key")) + "\r\n" +
		"\r\n"

	if _, err := brw.WriteString(response); err != nil {
		_ = netConn.Close()
		return nil, fmt.Errorf("wsproto: write handshake: %w", err)
	}
	if err := brw.Flush(); err != nil {
		_ = netConn.Close()
		return nil, fmt.Errorf("wsproto: flush handshake: %w", err)
	}

	return NewConn(netConn, brw.Reader, brw.Writer), nil
}

func validateHandshake(r *http.Request) error {
	if r.Method != http.MethodGet {
		return fmt.Errorf("%w: method must be GET", ErrBadHandshake)
	}
	if !headerContainsToken(r.Header, "Connection", "upgrade") {
		return fmt.Errorf("%w: missing Connection: Upgrade", ErrBadHandshake)
	}
	if !headerContainsToken(r.Header, "Upgrade", "websocket") {
		return fmt.Errorf("%w: missing Upgrade: websocket", ErrBadHandshake)
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return fmt.Errorf("%w: unsupported Sec-WebSocket-Version", ErrBadHandshake)
	}
	if r.Header.Get("Sec-WebSocket-Key") == "" {
		return fmt.Errorf("%w: missing Sec-WebSocket-Key", ErrBadHandshake)
	}
	return nil
}

func headerContainsToken(h http.Header, name, token string) bool {
	for _, value := range h.Values(name) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

// Conn is a server-side WebSocket connection.
type Conn struct {
	conn net.Conn
	br   *bufio.Reader
	bw   *bufio.Writer

	writeMu   sync.Mutex
	closeOnce sync.Once
}

// NewConn wraps an already-upgraded connection.
func NewConn(conn net.Conn, br *bufio.Reader, bw *bufio.Writer) *Conn {
	if br == nil {
		br = bufio.NewReader(conn)
	}
	if bw == nil {
		bw = bufio.NewWriter(conn)
	}
	return &Conn{conn: conn, br: br, bw: bw}
}

// WriteText sends a single unfragmented text frame.
func (c *Conn) WriteText(payload []byte) error {
	return c.writeFrame(OpText, payload)
}

// WritePong replies to a ping.
func (c *Conn) WritePong(payload []byte) error {
	return c.writeFrame(OpPong, payload)
}

// WriteClose sends a close frame.
func (c *Conn) WriteClose(code uint16, reason string) error {
	payload := make([]byte, 2+len(reason))
	binary.BigEndian.PutUint16(payload, code)
	copy(payload[2:], reason)
	return c.writeFrame(OpClose, payload)
}

func (c *Conn) writeFrame(opcode byte, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := writeFrame(c.bw, opcode, payload); err != nil {
		return err
	}
	return c.bw.Flush()
}

// ReadMessage returns the next data message or control frame. Control frames
// (ping, pong, close) are returned immediately; fragmented data messages are
// reassembled.
func (c *Conn) ReadMessage() (byte, []byte, error) {
	var (
		messageOpcode byte
		message       []byte
	)

	for {
		fin, opcode, payload, err := c.readFrame()
		if err != nil {
			return 0, nil, err
		}

		switch opcode {
		case OpPing, OpPong, OpClose:
			return opcode, payload, nil
		case OpText, OpBinary:
			if messageOpcode != 0 {
				return 0, nil, ErrProtocol
			}
			messageOpcode = opcode
			message = append(message, payload...)
		case OpContinuation:
			if messageOpcode == 0 {
				return 0, nil, ErrProtocol
			}
			message = append(message, payload...)
		default:
			return 0, nil, ErrProtocol
		}

		if len(message) > MaxMessageSize {
			return 0, nil, ErrTooLarge
		}
		if fin && messageOpcode != 0 {
			return messageOpcode, message, nil
		}
	}
}

func (c *Conn) readFrame() (fin bool, opcode byte, payload []byte, err error) {
	var header [2]byte
	if _, err = io.ReadFull(c.br, header[:]); err != nil {
		return false, 0, nil, err
	}

	fin = header[0]&0x80 != 0
	opcode = header[0] & 0x0F
	masked := header[1]&0x80 != 0
	length := int64(header[1] & 0x7F)

	switch length {
	case 126:
		var extended [2]byte
		if _, err = io.ReadFull(c.br, extended[:]); err != nil {
			return false, 0, nil, err
		}
		length = int64(binary.BigEndian.Uint16(extended[:]))
	case 127:
		var extended [8]byte
		if _, err = io.ReadFull(c.br, extended[:]); err != nil {
			return false, 0, nil, err
		}
		unsigned := binary.BigEndian.Uint64(extended[:])
		if unsigned > MaxMessageSize {
			return false, 0, nil, ErrTooLarge
		}
		length = int64(unsigned)
	}

	if length < 0 || length > MaxMessageSize {
		return false, 0, nil, ErrTooLarge
	}
	if opcode >= 0x8 && (!fin || length > 125) {
		return false, 0, nil, ErrProtocol
	}

	var mask [4]byte
	if masked {
		if _, err = io.ReadFull(c.br, mask[:]); err != nil {
			return false, 0, nil, err
		}
	}

	payload = make([]byte, length)
	if _, err = io.ReadFull(c.br, payload); err != nil {
		return false, 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return fin, opcode, payload, nil
}

func writeFrame(w io.Writer, opcode byte, payload []byte) error {
	var header [10]byte
	header[0] = 0x80 | opcode // FIN + opcode

	length := len(payload)
	var headerLen int
	switch {
	case length < 126:
		header[1] = byte(length)
		headerLen = 2
	case length <= 65535:
		header[1] = 126
		binary.BigEndian.PutUint16(header[2:4], uint16(length))
		headerLen = 4
	default:
		header[1] = 127
		binary.BigEndian.PutUint64(header[2:10], uint64(length))
		headerLen = 10
	}

	if _, err := w.Write(header[:headerLen]); err != nil {
		return err
	}
	if length == 0 {
		return nil
	}
	_, err := w.Write(payload)
	return err
}

// Close sends a normal close frame and closes the underlying connection.
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		_ = c.conn.SetWriteDeadline(time.Now().Add(time.Second))
		_ = c.WriteClose(CloseNormal, "")
		err = c.conn.Close()
	})
	return err
}
