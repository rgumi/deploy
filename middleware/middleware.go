package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func LogRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		before := time.Now()

		defer func() {
			delta := time.Since(before)

			log.Infof("%s %s %s %v", r.RemoteAddr, r.Method, r.URL, delta)
		}()
		handler.ServeHTTP(w, r)
	})
}

// SetRequestID sets a unique ID for each request. Allows for better tracing
func SetRequestID(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		r.Header.Set("X-Request-ID", id)
		w.Header().Set("X-Request-ID", id)

		handler.ServeHTTP(w, r)
	})
}

func CorsHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Methods, Content-Type")
		handler.ServeHTTP(w, r)
	})
}
