package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/Adeelp1/vigil-gateway/internal/middleware"
)

// Handler is the final step in the middleware chain: it forwards the request
// to the upstream server and relays the response back to the client.
type Handler struct {
	upstreamAddr string
	client       *http.Client
}

// hopByHopHeaders must not be forward to upstream per RFC 7230.
var hopByHopHeaders = []string{
	"Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Athorization",
	"TE", "Trailers", "Transfer-Encoding", "Upgrade",
}

func NewHandler(upstreamAddr string, readTimeout, writeTimeout time.Duration) *Handler {
	transport := &http.Transport{
		// Pre-connect a pool of TCP connections to the upstream.
		MaxIdleConns:        512,
		MaxIdleConnsPerHost: 256,
		IdleConnTimeout:     90 * time.Second,
		// Keep-alive on the upstream leg — avoids SYN/SYN-ACK on every request.
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	return &Handler{
		upstreamAddr: upstreamAddr,
		client: &http.Client{
			Transport: transport,
			Timeout:   readTimeout + writeTimeout,
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Build the upstream URL from the original request path + query.
	target := fmt.Sprintf("http://%s%s", h.upstreamAddr, r.RequestURI)

	// Clone the request, replacing the URL with the upstream one.
	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		http.Error(w, "bad gatway", http.StatusBadGateway)
		return
	}

	// Copy client headers, stripping hop-by-hop.
	copyHeaders(upstreamReq.Header, r.Header)

	// Append client IP to X-Forwarded-For.
	// Correct implementation: preserve existing chain, append ours.
	clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if prior := r.Header.Get("X-Forwarded-For"); prior != "" {
		upstreamReq.Header.Set("X-Forwarded-For", prior+", "+clientIP)
	} else {
		upstreamReq.Header.Set("X-Forwarded-For", clientIP)
	}

	upstreamReq.Header.Set("X-Forwarded-HOST", r.Host)
	upstreamReq.Header.Set("X-Real-IP", clientIP)

	// Forward to upstream.
	resp, err := h.client.Do(upstreamReq)
	if err != nil {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Relay upstream response headers to client.
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	// Stream the body — no buffering, O(1) memory regardless of body size.
	// io.Copy uses a 32 KB buffer internally, which is a good trade-off.
	if _, err := io.Copy(w, resp.Body); err != nil {
		// Client disconnect mid-stream
		_ = middleware.GetRequestID(r.Context()) // log here
	}
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		if isHopByHop(key) {
			continue
		}
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}

func isHopByHop(header string) bool {
	for _, h := range hopByHopHeaders {
		if http.CanonicalHeaderKey(header) == h {
			return true
		}
	}
	return false
}
