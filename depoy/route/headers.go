package route

import (
	"net/http"
	"strings"
)

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeaders(src, dest http.Header) {
	for k, v := range src {
		for _, vv := range v {
			dest.Add(k, vv)
		}
	}
}

func delHopHeaders(r http.Header) {
	for _, h := range hopHeaders {
		r.Del(h)
	}
}

func appendHostToXForwardHeader(r http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := r["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	r.Set("X-Forwarded-For", host)
}
