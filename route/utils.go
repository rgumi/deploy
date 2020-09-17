package route

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

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

func sendResponse(resp *http.Response, w http.ResponseWriter) int {
	log.Debug("Sending response to downstream client")
	b, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	copyHeaders(resp.Header, w.Header())

	w.Header().Add("Server", "Depoy")
	w.WriteHeader(resp.StatusCode)
	w.Write(b)
	log.Debug("Successfully send response to downstream client")
	return len(b)
}

func formateRequest(old *http.Request, addr string, body []byte) (*http.Request, error) {

	reader := bytes.NewReader(body)

	new, _ := http.NewRequest(old.Method, addr, reader)

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

// GGT receives list of ints of which the ggT needs to be found
// in []int needs to be initialized as if len(in) < 2 the first value
// will be returned
func GGT(in []uint8) uint8 {
	count := len(in)
	if count < 2 {
		return in[0]
	}
	ggt := ggT(in[0], in[1])

	for i := 2; i < count; i++ {
		ggt = ggT(ggt, in[i])
	}
	return ggt
}

// https://de.wikipedia.org/wiki/Euklidischer_Algorithmus
func ggT(a, b uint8) uint8 {
	for b != 0 {
		t := b
		b = a % b
		a = t
	}
	return a
}

func addCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	expire := time.Now().Add(ttl)
	cookie := http.Cookie{
		Name:    name,
		Value:   value,
		Expires: expire,
	}
	http.SetCookie(w, &cookie)
}

func checkCookie(req *http.Request, name string) (string, time.Time) {
	cookie, err := req.Cookie(name)
	if err != nil {
		return "", time.Time{}
	}
	return cookie.Value, cookie.Expires
}
