package route

import "depoy/upstreamclient"

type Target struct {
	ID         int64
	Name       string
	Addr       string
	MetricChan chan upstreamclient.Metric
	Active     bool
}
