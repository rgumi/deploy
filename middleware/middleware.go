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

			log.Infof("%s \"%s %s %s\" %v",
				r.RemoteAddr, r.Method, r.URL.Path,
				r.Proto, delta,
			)
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
