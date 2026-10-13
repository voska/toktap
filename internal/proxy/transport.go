package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	utls "github.com/refraction-networking/utls"
)

// NewChromeTransport returns an http.RoundTripper that mimics Chrome's TLS
// fingerprint (JA3/JA4) using uTLS. It enables HTTP/2 so the
// connection works when the server negotiates HTTP/2 via ALPN (which
// chatgpt.com / Cloudflare does).
func NewChromeTransport() http.RoundTripper {
	transport := &http.Transport{ForceAttemptHTTP2: true}
	transport.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}

		conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		config := &utls.Config{
			ServerName: host,
		}
		if transport.TLSClientConfig != nil {
			config.RootCAs = transport.TLSClientConfig.RootCAs
		}
		tlsConn := utls.UClient(conn, config, utls.HelloChrome_Auto)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return nil, err
		}

		state := tlsConn.ConnectionState()
		return &chromeTLSConn{Conn: tlsConn, state: tls.ConnectionState{
			Version:                     state.Version,
			HandshakeComplete:           state.HandshakeComplete,
			DidResume:                   state.DidResume,
			CipherSuite:                 state.CipherSuite,
			NegotiatedProtocol:          state.NegotiatedProtocol,
			ServerName:                  state.ServerName,
			PeerCertificates:            state.PeerCertificates,
			VerifiedChains:              state.VerifiedChains,
			SignedCertificateTimestamps: state.SignedCertificateTimestamps,
			OCSPResponse:                state.OCSPResponse,
			TLSUnique:                   state.TLSUnique,
			ECHAccepted:                 state.ECHAccepted,
		}}, nil
	}
	return transport
}

// chromeTLSConn exposes uTLS ALPN state using the interface net/http expects.
type chromeTLSConn struct {
	net.Conn
	state tls.ConnectionState
}

func (c *chromeTLSConn) ConnectionState() tls.ConnectionState {
	return c.state
}
