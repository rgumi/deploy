package gateway

import (
	"depoy/route"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
)

var route1 = &route.Route{}
var route2 = &route.Route{}
var route3 = &route.Route{}
var g = NewGateway(":10010")

func testHandle(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(200)
}

func init() {
	log.SetLevel(log.DebugLevel)

	route1.Name = "route1"
	route1.Host = "example.com"
	route1.Methods = []string{"GET", "POST", "UPDATE"}
	route1.Prefix = "/"
	route1.Handler = testHandle

	route2.Name = "route2"
	route2.Host = "example.com"
	route2.Methods = []string{"POST", "UPDATE"}
	route2.Prefix = "/route2"
	route2.Handler = testHandle

	route3.Name = "route3"
	route3.Host = "*"
	route3.Methods = []string{"GET"}
	route3.Prefix = "/"
	route3.Handler = testHandle
}

func Test_RegisterRoutes(t *testing.T) {
	var exists bool

	// Add route1 to gateway
	if err := g.RegisterRoute(route1); err != nil {
		t.Error(err)
	}

	// Check if route2 exists => should not exist as its not registered yet
	exists, _ = g.Router["example.com"].CheckIfHandleExists("POST", "/route2")
	if exists {
		// only error if route exists
		t.Error("Handler should not exist for router")
	}

	// Add route2 to gateway
	if err := g.RegisterRoute(route2); err != nil {
		t.Error(err)
	}

	// Add route2 to gateway again => should result in err as route exists already
	if err := g.RegisterRoute(route2); err == nil {
		t.Error(err)
	}

	// Check if route2 exists => should exist as it was just registered
	exists, _ = g.Router["example.com"].CheckIfHandleExists("POST", "/route2")
	if !exists {
		t.Error("Handler should exist for router")
	}

	// Check if random route exists => should not exist as its not registered
	exists, _ = g.Router["example.com"].CheckIfHandleExists("POST", "/doesnotexist")
	if exists {
		t.Error("Handler should ´not exist for router")
	}
}

func Test_RemoveRoute(t *testing.T) {
	var exists bool

	// Check if route2 exists => should exist as it was just registered
	exists, _ = g.Router["example.com"].CheckIfHandleExists("POST", "/route2")
	if !exists {
		t.Error("Handler should exist for router")
	}

	g.RemoveRoute(route2.Name)

	if g.Routes[route2.Name] != nil {
		t.Error("Route should not exist as it was just deleted")
	}

	exists, _ = g.Router["example.com"].CheckIfHandleExists("POST", "/route2")
	if exists {
		t.Error("Handler should not exist for router as it was just deleted")
	}
}

func Test_HandlerOfGatewayWithHost(t *testing.T) {

	var rr *httptest.ResponseRecorder
	var req *http.Request
	var err error

	// with host
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/", nil)
	req.Host = "example.com"
	if err != nil {
		t.Error(err)
	}

	g.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

}

func Test_HandlerOfGatewayWithoutHost(t *testing.T) {

	var rr *httptest.ResponseRecorder
	var req *http.Request
	var err error

	g.RegisterRoute(route3)
	// without host
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Error(err)
	}

	g.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func Test_HandlerOfGatewayNoHandler(t *testing.T) {

	var rr *httptest.ResponseRecorder
	var req *http.Request
	var err error

	g.RemoveRoute(route3.Name)

	// without host
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Error(err)
	}

	g.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}
}
