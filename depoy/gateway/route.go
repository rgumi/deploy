package gateway

import (
	"github.com/julienschmidt/httprouter"
)

// Route defines a mapping between a downstream URI and the upstream (restful) service
type Route struct {
	Src               string
	Dest              string
	UpstreamTimeoutMs int
	TimeoutMs         int
	IdleTimeout       int
	Methods           []string
	Chain             httprouter.Handle
}
