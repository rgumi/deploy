package middleware

import (
	"log"
	"net/http"
	"time"
)

func LogRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		before := time.Now()

		defer func() {
			delta := time.Now().Sub(before)
			log.Printf("%s %s %s %v\n", r.RemoteAddr, r.Method, r.URL, delta)
		}()
		handler.ServeHTTP(w, r)
	})
}
