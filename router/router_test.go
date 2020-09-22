package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testHandle(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

var router *Router

func Test_New(t *testing.T) {

	router = NewRouter()

	router.AddHandler("get", "/hello", testHandle)

	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Error(err)
	}

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func Test_Redundant_Handle(t *testing.T) {
	err := router.AddHandler("get", "/hello", testHandle)
	if err == nil {
		t.Error("Inserted an already existing handle")
	}
}

func Test_CaseSensitity(t *testing.T) {
	err := router.AddHandler("get", "/HellO", testHandle)
	if err == nil {
		t.Error("Inserted an already existing handle")
	}
}

func Test_MethodsOnSamePrefix(t *testing.T) {
	var err error
	var method string

	method = "post"
	err = router.AddHandler(method, "/hello", testHandle)
	if err != nil {
		t.Errorf("Unable to add %s to existing prefix", method)
		t.Errorf(err.Error())
	}

	method = "UPDATE"
	err = router.AddHandler(method, "/hello", testHandle)
	if err != nil {
		t.Errorf("Unable to add %s to existing prefix", method)
		t.Errorf(err.Error())
	}

	method = "PUT"
	err = router.AddHandler(method, "/hello", testHandle)
	if err != nil {
		t.Errorf("Unable to add %s to existing prefix", method)
		t.Errorf(err.Error())
	}
}

func Test_DefaultNotFound(t *testing.T) {

	req, err := http.NewRequest("GET", "/doesnotexist", nil)
	if err != nil {
		t.Error(err)
	}

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}
}

func Test_DefaultErrorHandler(t *testing.T) {

	req, err := http.NewRequest("GET", "/doesnotexist", nil)
	if err != nil {
		t.Error(err)
	}

	rr := httptest.NewRecorder()
	testError := fmt.Errorf("Test error message")
	router.ErrorHandler(rr, req, testError)
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}

	expected := testError.Error()
	if rr.Body.String() != expected {
		t.Log("Response body:" + rr.Body.String())
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
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

func Test_NoHandler(t *testing.T) {
	router := NewRouter()

	req, err := http.NewRequest("GET", "/hello", nil)
	if err != nil {
		t.Error(err)
	}

	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}
}
