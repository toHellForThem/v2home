package wsproto

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func TestAcceptKey(t *testing.T) {
	const key = "dGhlIHNhbXBsZSBub25jZQ=="
	const want = "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="

	if got := AcceptKey(key); got != want {
		t.Fatalf("AcceptKey(%q) = %q, want %q", key, got, want)
	}
}

func TestReadMaskedTextFrame(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	conn := NewConn(server, bufio.NewReader(server), bufio.NewWriter(server))
	payload := []byte(`{"app":"v2home"}`)

	go func() {
		_ = writeMaskedFrame(client, OpText, payload)
	}()

	_ = server.SetReadDeadline(time.Now().Add(2 * time.Second))

	opcode, got, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if opcode != OpText {
		t.Fatalf("opcode = %#x, want %#x", opcode, OpText)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}

func TestWriteTextFrame(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	conn := NewConn(server, bufio.NewReader(server), bufio.NewWriter(server))
	payload := []byte("hello")

	go func() {
		_ = conn.WriteText(payload)
	}()

	_ = client.SetReadDeadline(time.Now().Add(2 * time.Second))

	reader := bufio.NewReader(client)
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		t.Fatalf("read header: %v", err)
	}
	if header[0] != 0x81 {
		t.Fatalf("first byte = %#x, want 0x81", header[0])
	}
	if header[1] != byte(len(payload)) {
		t.Fatalf("length byte = %d, want %d", header[1], len(payload))
	}

	body := make([]byte, len(payload))
	if _, err := io.ReadFull(reader, body); err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != string(payload) {
		t.Fatalf("body = %q, want %q", body, payload)
	}
}

func writeMaskedFrame(w io.Writer, opcode byte, payload []byte) error {
	mask := [4]byte{0x11, 0x22, 0x33, 0x44}

	header := []byte{0x80 | opcode}
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, 0x80|byte(length))
	case length <= 65535:
		header = append(header, 0x80|126, byte(length>>8), byte(length))
	default:
		header = append(header, 0x80|127)
		var extended [8]byte
		binary.BigEndian.PutUint64(extended[:], uint64(length))
		header = append(header, extended[:]...)
	}
	header = append(header, mask[:]...)

	masked := make([]byte, length)
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}

	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(masked)
	return err
}
