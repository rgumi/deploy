package metrics

import (
	"depoy/upstreamclient"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

type Route interface {
	SetMetricChannel(chan upstreamclient.Metric)
}

type Metric interface {
	GetStatus() int
	GetUpstreamIP() []net.IPAddr
	GetUpstreamTime() time.Duration
}

// MetricServer contains diffrent metrics which can be collected
type MetricServer struct {
	Routes map[string]chan upstreamclient.Metric
}

// safe to db
// expose via prom
// alarming to statemgt
// register for multiple routes

func New() *MetricServer {
	return new(MetricServer)
}

func (m *MetricServer) Register(route Route) {
	ch := make(chan upstreamclient.Metric, 10)
	route.SetMetricChannel(ch)
}

func listenAndPersist(in chan upstreamclient.Metric) {
	for {
		for m := range in {
			log.Info(m)
		}
	}
}
