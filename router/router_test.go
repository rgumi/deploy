package router

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func testHandle(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(200)
}

var router *Router

func init() {
	router = NewRouter()
}

func Test_Handle(t *testing.T) {
	err := router.Handle("get", "/hello", testHandle)
	if err != nil {
		t.Error("Unable to insert handle")
	}
}

func Test_Redundant_Handle(t *testing.T) {
	err := router.Handle("get", "/hello", testHandle)
	if err == nil {
		t.Error("Inserted an already existing handle")
	}
}

func Test_CaseSensitity(t *testing.T) {
	err := router.Handle("get", "/HellO", testHandle)
	if err != nil {
		t.Error("Unable to insert case sensitive handle")
	}
}

func Test_MethodsOnSamePrefix(t *testing.T) {
	var err error
	var method string

	method = "post"
	err = router.Handle(method, "/hello", testHandle)
	if err != nil {
		t.Errorf("Unable to add %s to existing prefix", method)
		t.Errorf(err.Error())
	}

	method = "UPDATE"
	err = router.Handle(method, "/hello", testHandle)
	if err != nil {
		t.Errorf("Unable to add %s to existing prefix", method)
		t.Errorf(err.Error())
	}

	method = "PUT"
	err = router.Handle(method, "/hello", testHandle)
	if err != nil {
		t.Errorf("Unable to add %s to existing prefix", method)
		t.Errorf(err.Error())
	}
}

func Test_DeleteHandle(t *testing.T) {

	// handles get /hello & post /hello exist
	if err := router.RemoveHandle("get", "/hello"); err != nil {
		t.Errorf("Unable to delete existing handle")
	}

	if err := router.RemoveHandle("POST", "/hello"); err != nil {
		t.Errorf("Unable to delete existing handle")
	}

	// should not exist anymore
	if err := router.RemoveHandle("POST", "/hello"); err == nil {
		t.Errorf("Removing non-existing handle did not return error")
	}
	if err := router.RemoveHandle("OPTIONS", "/hello"); err == nil {
		t.Errorf("Removing non-existing handle did not return error")
	}
}
