package middleware

import (
	"time"

	"github.com/valyala/fasthttp"

	log "github.com/sirupsen/logrus"
)

func LogRequest(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		before := time.Now()

		defer func() {
			log.Infof("%s \"%s %s %s\" %v",
				ctx.RemoteAddr(), ctx.Method(), ctx.URI().String(),
				string(ctx.Request.Header.UserAgent()), time.Since(before),
			)
		}()
		handler(ctx)
	}
}
