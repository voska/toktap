package proxy

import (
	"crypto/tls"
	"net"
	"net/http"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

// NewChromeTransport returns an http.RoundTripper that mimics Chrome's TLS
// fingerprint (JA3/JA4) using uTLS. It uses an http2.Transport so the
// connection works when the server negotiates HTTP/2 via ALPN (which
// chatgpt.com / Cloudflare does).
func NewChromeTransport() http.RoundTripper {
	return &http2.Transport{
		DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}

			conn, err := net.Dial(network, addr)
			if err != nil {
				return nil, err
			}

			config := &utls.Config{
				ServerName: host,
			}
			tlsConn := utls.UClient(conn, config, utls.HelloChrome_Auto)
			if err := tlsConn.Handshake(); err != nil {
				_ = conn.Close()
				return nil, err
			}

			return tlsConn, nil
		},
	}
}
