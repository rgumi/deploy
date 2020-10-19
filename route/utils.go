package route

import (
	"github.com/valyala/fasthttp"
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

func appendXForwardForHeader(req *fasthttp.Request, host string) {
	prior := string(req.Header.Peek("X-Forwarded-For"))

	if prior != "" {
		prior = prior + ", "
	}
	req.Header.Set("X-Forwarded-For", prior+host)
}

func delRequestHopHeader(src *fasthttp.Request) {
	for _, h := range hopHeaders {
		src.Header.Del(h)
	}
}

func delResponseHopHeader(src *fasthttp.Response) {
	for _, h := range hopHeaders {
		src.Header.Del(h)
	}
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
