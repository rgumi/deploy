package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	buf              bytes.Buffer
	request          *http.Request
	responseRecorder *httptest.ResponseRecorder
	handler          http.HandlerFunc
)

func init() {
	log.SetOutput(&buf)

	request = httptest.NewRequest("GET", "/", nil)
	responseRecorder = httptest.NewRecorder()
	handler = http.HandlerFunc(waiterHandler)
}

func waiterHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(2 * time.Second)
	w.WriteHeader(200)
	w.Write(nil)
}

func Test_LogRequest(t *testing.T) {

	// expect this to take 2 seconds
	LogRequest(handler).ServeHTTP(responseRecorder, request)

	fmt.Println(buf.String())
	// check if log is written
	if !strings.Contains(buf.String(), "GET / ") {
		t.Error("Logger did not write to log output")
	}
}

func Test_SetRequestID(t *testing.T) {
	SetRequestID(handler).ServeHTTP(responseRecorder, request)

	var idRequest, idResponse string

	if idRequest = request.Header.Get("X-Request-ID"); idRequest == "" {
		t.Error("SetRequestID did not set X-Request-ID in Request")
	}

	if idResponse = responseRecorder.Header().Get("X-Request-ID"); idResponse == "" {
		t.Error("SetRequestID did not set X-Request-ID in Response")
	}

	if idRequest != idResponse {
		t.Error("IDs do not match")
	}
}
