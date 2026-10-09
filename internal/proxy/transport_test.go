package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

func TestChromeTransportCancelsTLSHandshake(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	transport := NewChromeTransport().(*http2.Transport)
	if transport.DialTLSContext == nil {
		t.Fatal("context-aware TLS dial callback is missing")
	}
	result := make(chan error, 1)
	go func() {
		conn, err := transport.DialTLSContext(ctx, "tcp", listener.Addr().String(), &tls.Config{})
		if conn != nil {
			conn.Close()
		}
		result <- err
	}()

	// Accept TCP but leave the TLS handshake stalled.
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var hello [5]byte
	if _, err := io.ReadFull(conn, hello[:]); err != nil {
		t.Fatalf("reading TLS ClientHello: %v", err)
	}
	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("TLS dial error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		conn.Close()
		<-result
		t.Fatal("cancelled dial remained blocked in TLS handshake")
	}
	if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, conn); err != nil {
		t.Fatalf("TLS connection remained open after cancellation: %v", err)
	}
}
