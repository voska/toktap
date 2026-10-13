package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChromeTransportHTTP2(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Errorf("upstream protocol = %s, want HTTP/2", r.Proto)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	transport, ok := NewChromeTransport().(*http.Transport)
	if !ok {
		t.Fatal("Chrome transport must use the patched standard library HTTP/2 implementation")
	}
	defer transport.CloseIdleConnections()
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	transport.TLSClientConfig = &tls.Config{RootCAs: roots}
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.ProtoMajor != 2 || resp.TLS == nil || resp.TLS.NegotiatedProtocol != "h2" {
		t.Fatalf("response protocol = %s, TLS = %+v, want h2", resp.Proto, resp.TLS)
	}
	if len(resp.TLS.VerifiedChains) == 0 {
		t.Fatal("TLS certificate verification state was lost")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil || string(body) != "ok" {
		t.Fatalf("response = %q, error = %v", body, err)
	}
}

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
	transport := NewChromeTransport().(*http.Transport)
	if transport.DialTLSContext == nil {
		t.Fatal("context-aware TLS dial callback is missing")
	}
	result := make(chan error, 1)
	go func() {
		conn, err := transport.DialTLSContext(ctx, "tcp", listener.Addr().String())
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
