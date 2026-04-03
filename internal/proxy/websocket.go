package proxy

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// handleWebSocket hijacks the client connection, dials the upstream over TLS,
// replays the HTTP upgrade request, and bidirectionally copies bytes until
// either side closes. No framing is inspected — it's a transparent tunnel.
func (p *Proxy) handleWebSocket(w http.ResponseWriter, r *http.Request, route *Route) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket hijack not supported", http.StatusInternalServerError)
		return
	}

	// Dial upstream.
	host := route.Upstream.Host
	if !strings.Contains(host, ":") {
		if route.Upstream.Scheme == "https" || route.Upstream.Scheme == "wss" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	var upstream net.Conn
	var err error
	if route.Upstream.Scheme == "https" || route.Upstream.Scheme == "wss" {
		upstream, err = tls.Dial("tcp", host, &tls.Config{
			ServerName: route.Upstream.Hostname(),
			NextProtos: []string{"http/1.1"},
		})
	} else {
		upstream, err = net.Dial("tcp", host)
	}
	if err != nil {
		log.Printf("websocket dial upstream [%s]: %v", route.Provider, err)
		http.Error(w, "upstream connect failed", http.StatusBadGateway)
		return
	}
	defer func() { _ = upstream.Close() }()

	// Rebuild the upgrade request for upstream.
	var reqBuf strings.Builder
	reqBuf.WriteString(r.Method + " " + r.URL.RequestURI() + " HTTP/1.1\r\n")
	reqBuf.WriteString("Host: " + route.Upstream.Host + "\r\n")
	for key, vals := range r.Header {
		skip := false
		for _, h := range stripHeaders {
			if strings.EqualFold(key, h) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		for _, v := range vals {
			reqBuf.WriteString(key + ": " + v + "\r\n")
		}
	}
	reqBuf.WriteString("\r\n")

	if _, err := io.WriteString(upstream, reqBuf.String()); err != nil {
		log.Printf("websocket write upgrade [%s]: %v", route.Provider, err)
		http.Error(w, "upstream write failed", http.StatusBadGateway)
		return
	}

	// Hijack the client connection.
	client, buf, err := hj.Hijack()
	if err != nil {
		log.Printf("websocket hijack [%s]: %v", route.Provider, err)
		return
	}
	defer func() { _ = client.Close() }()

	// Flush any buffered data from the hijacked reader to upstream.
	if n := buf.Reader.Buffered(); n > 0 {
		buffered := make([]byte, n)
		if _, err := buf.Read(buffered); err == nil {
			_, _ = upstream.Write(buffered)
		}
	}

	log.Printf("websocket connected [%s]: %s", route.Provider, r.URL.Path)
	start := time.Now()

	// Bidirectional copy.
	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(client, upstream)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(upstream, client)
		done <- struct{}{}
	}()
	<-done

	log.Printf("websocket closed [%s]: %s (%s)", route.Provider, r.URL.Path, time.Since(start).Round(time.Millisecond))
}
