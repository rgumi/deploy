package route

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
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

func sendResponse(resp *http.Response, w http.ResponseWriter) {
	log.Debug("Sending response to downstream client")
	b, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	/*
		if err != nil {
			log.Error("Failed to read response body of upstream response")
			log.Error(err)
			http.Error(w, "Could not return response", 500)

			return
		}
	*/
	w.WriteHeader(resp.StatusCode)
	w.Write(b)
	log.Debug("Successfully send response to downstream client")
}

func formateRequest(old *http.Request, addr string, body []byte) (*http.Request, error) {

	reader := bytes.NewReader(body)

	new, _ := http.NewRequest(old.Method, addr, reader)
	/*
		if err != nil {
			return nil, err
		}
	*/

	// setup the X-Forwarded-For header
	if clientIP, _, err := net.SplitHostPort(old.RemoteAddr); err == nil {
		appendHostToXForwardHeader(new.Header, clientIP)
	}

	// Copy all headers from the downstream request to the new upstream request
	copyHeaders(old.Header, new.Header)
	// Remove all headers from the new upstream request that are bound to downstream connection
	delHopHeaders(new.Header)

	return new, nil
}
