package proxy

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

)

func TestIsWebSocketUpgrade(t *testing.T) {
	tests := []struct {
		name       string
		headers    http.Header
		wantResult bool
	}{
		{
			name: "valid upgrade",
			headers: http.Header{
				"Upgrade":    {"websocket"},
				"Connection": {"Upgrade"},
			},
			wantResult: true,
		},
		{
			name: "case insensitive",
			headers: http.Header{
				"Upgrade":    {"WebSocket"},
				"Connection": {"upgrade, keep-alive"},
			},
			wantResult: true,
		},
		{
			name:       "no headers",
			headers:    http.Header{},
			wantResult: false,
		},
		{
			name: "upgrade without connection",
			headers: http.Header{
				"Upgrade": {"websocket"},
			},
			wantResult: false,
		},
		{
			name: "connection without upgrade",
			headers: http.Header{
				"Connection": {"Upgrade"},
			},
			wantResult: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Header: tt.headers}
			if got := isWebSocketUpgrade(r); got != tt.wantResult {
				t.Errorf("isWebSocketUpgrade() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestWebSocketTunnel(t *testing.T) {
	// Start a raw TCP server that speaks the WebSocket upgrade handshake
	// and echoes back a message.
	upstream := startEchoUpgradeServer(t)
	defer upstream.Close()

	upstreamURL, _ := url.Parse("http://" + upstream.Addr().String())
	routes := map[string]*Route{
		"test": {Provider: "test", Upstream: upstreamURL},
	}
	p := New(routes, &noopWriter{}, nil)

	// Create a test HTTP server with the proxy.
	proxyServer := httptest.NewServer(p)
	defer proxyServer.Close()

	// Dial the proxy as a raw TCP client.
	conn, err := net.Dial("tcp", strings.TrimPrefix(proxyServer.URL, "http://"))
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send a WebSocket upgrade request.
	req := "GET /test/responses HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatalf("write upgrade: %v", err)
	}

	// Read the 101 response.
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want 101", resp.StatusCode)
	}

	// Send a raw message through the tunnel.
	_, _ = conn.Write([]byte("hello"))
	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if got := string(buf[:n]); got != "hello" {
		t.Errorf("echo = %q, want %q", got, "hello")
	}
}

// startEchoUpgradeServer starts a raw TCP listener that accepts one connection,
// reads an HTTP upgrade request, sends a 101 response, then echoes bytes.
func startEchoUpgradeServer(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

		reader := bufio.NewReader(conn)
		// Read HTTP request.
		req, err := http.ReadRequest(reader)
		if err != nil {
			t.Errorf("upstream read request: %v", err)
			return
		}
		_ = req.Body.Close()

		// Send 101 Switching Protocols.
		resp := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"\r\n"
		_, _ = conn.Write([]byte(resp))

		// Echo bytes.
		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				return
			}
			_, _ = conn.Write(buf[:n])
		}
	}()
	return ln
}

