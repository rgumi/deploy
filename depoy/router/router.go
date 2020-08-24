package router

import (
	"fmt"
	"net/http"
	"strings"

	radix "github.com/armon/go-radix"
	log "github.com/sirupsen/logrus"
)

func defaultErrorHandler(w http.ResponseWriter, r *http.Request, e error) {
	w.WriteHeader(500)
	w.Write([]byte(e.Error()))
}

func defaultNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
}

type Router struct {
	tree            map[string]*radix.Tree
	ErrorHandler    func(w http.ResponseWriter, r *http.Request, e error)
	NotFoundHandler func(w http.ResponseWriter, r *http.Request)
}

func NewRouter() *Router {
	return &Router{
		tree:            make(map[string]*radix.Tree),
		ErrorHandler:    defaultErrorHandler,
		NotFoundHandler: defaultNotFoundHandler,
	}
}

func (r *Router) CheckIfHandleExists(method, prefix string) (bool, error) {
	var err error

	method, prefix, err = checkMethodAndPrefix(method, prefix)
	if err != nil {
		return false, err
	}

	// if no tree exists with given method, initialize it
	if r.tree[method] == nil {
		r.tree[method] = radix.New()

		// if no tree existed, no handle can exist for it
		return false, nil
	}

	if _, exists := r.tree[method].Get(prefix); exists {
		// handle already exists with this method
		return true, fmt.Errorf("Handle already exists for method %s and prefix %s", method, prefix)
	}
	// Handle does not exist
	return false, nil
}

func checkMethodAndPrefix(m, p string) (method, prefix string, err error) {
	method = strings.ToUpper(m)
	prefix = strings.ToLower(p)
	err = nil

	// method cannot be empty
	if method == "" {
		err = fmt.Errorf("Method cannot be empty")
	}

	// Prefix needs to be not empty and start with a /
	if prefix == "" || string(prefix[0]) != "/" {
		err = fmt.Errorf("Prefix cannot be empty and must start with a \"/\"")
	}
	return
}

func (r *Router) AddHandler(method, prefix string, handler http.HandlerFunc) error {
	var err error

	method, prefix, err = checkMethodAndPrefix(method, prefix)
	if err != nil {
		return err
	}

	// check if the prefix & method combination already exists
	_, err = r.CheckIfHandleExists(method, prefix)
	if err != nil {
		return err
	}

	log.Debugf("Adding new Handle {Method:%s Prefix: %s} to Router %v", method, prefix, r)
	if _, updated := r.tree[method].Insert(prefix, handler); updated {
		return fmt.Errorf("Updated an entry")
	}

	return nil
}

func (r *Router) RemoveHandle(method, prefix string) error {
	var err error

	method, prefix, err = checkMethodAndPrefix(method, prefix)
	if err != nil {
		return err
	}

	// check if the prefix & method combination already exists
	_, err = r.CheckIfHandleExists(method, prefix)
	if err == nil {
		return fmt.Errorf("Handle does not exist")
	}

	if _, deleted := r.tree[method].Delete(prefix); !deleted {
		return fmt.Errorf("Could not delete handle")
	}

	// delete successful
	return nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	// recovers if any goroutine results in an unhandeled error (internal server error)
	// and returns 500 and the error message to the downstream client
	// therefore no connections can hangup and server crashes are very unlikely
	defer func() {
		if e := recover(); e != nil {
			log.Errorf("Recovered in Router: %v", e)
			r.ErrorHandler(w, req, e.(error))
		}
	}()

	if _, found := r.tree[req.Method]; found {
		if _, h, found := r.tree[req.Method].LongestPrefix(req.URL.String()); found {
			h.(http.HandlerFunc)(w, req)
			return
		}
	}
	r.NotFoundHandler(w, req)
}

func (r *Router) ServeFiles(path string, root http.FileSystem) {
	fileServer := http.FileServer(root)

	r.AddHandler("GET", path, func(w http.ResponseWriter, req *http.Request) {
		// get the file from the URL => remove handler prefix from URL
		req.URL.Path = strings.Split(req.URL.Path, path)[1]
		fileServer.ServeHTTP(w, req)
	})
}
