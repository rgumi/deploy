package router

import (
	"fmt"
	"strings"

	"github.com/valyala/fasthttp"

	radix "github.com/armon/go-radix"
	log "github.com/sirupsen/logrus"
)

func defaultErrorHandler(ctx *fasthttp.RequestCtx, e error) {
	ctx.Response.SetStatusCode(500)
	ctx.Response.SetBody([]byte(e.Error()))
}

func defaultNotFoundHandler(ctx *fasthttp.RequestCtx) {
	ctx.Response.SetStatusCode(404)
}

type Router struct {
	tree            map[string]*radix.Tree
	ErrorHandler    func(ctx *fasthttp.RequestCtx, e error)
	NotFoundHandler func(ctx *fasthttp.RequestCtx)
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

	// method cannot be empty
	if method == "" {
		err = fmt.Errorf("Method cannot be empty")
	}
	// Prefix needs to be not empty and start with a /
	if prefix == "" || string(prefix[0]) != "/" {
		err = fmt.Errorf("Prefix cannot be empty and must start with a \"/\"")
	}
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

func (r *Router) Handle(method, prefix string, handler fasthttp.RequestHandler) error {
	var err error
	httpMethod := strings.ToUpper(method)
	// check if the prefix & method combination already exists
	_, err = r.CheckIfHandleExists(httpMethod, prefix)
	if err != nil {
		return err
	}
	log.Debugf("Adding new Handle {Method:%s Prefix: %s} to Router", httpMethod, prefix)
	if _, updated := r.tree[httpMethod].Insert(prefix, handler); updated {
		return fmt.Errorf("Updated an entry")
	}
	return nil
}

func (r *Router) RemoveHandle(method, prefix string) error {
	var err error
	httpMethod := strings.ToUpper(method)
	// check if the prefix & method combination already exists
	_, err = r.CheckIfHandleExists(httpMethod, prefix)
	if err == nil {
		return fmt.Errorf("Handle does not exist")
	}

	if _, deleted := r.tree[httpMethod].Delete(prefix); !deleted {
		return fmt.Errorf("Could not delete handle")
	}

	// delete successful
	return nil
}

func (r *Router) ServeHTTP(ctx *fasthttp.RequestCtx) {
	defer func() {
		if err := recover(); err != nil {
			r.ErrorHandler(ctx, err.(error))
		}
	}()
	method := string(ctx.Method())
	if _, found := r.tree[method]; found {
		if _, h, found := r.tree[method].LongestPrefix(string(ctx.URI().Path())); found {
			h.(fasthttp.RequestHandler)(ctx)
			return
		}
	}
	r.NotFoundHandler(ctx)
}
