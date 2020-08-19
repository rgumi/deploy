package upstreamclient

import (
	"net"
	"time"
)

type Metric struct {
	UpstreamStatus int
	UpstreamIP     []net.IPAddr
	UpstreamTime   time.Duration
}

func (m Metric) GetStatus() int {
	return m.UpstreamStatus
}

func (m Metric) GetUpstreamIP() []net.IPAddr {
	return m.UpstreamIP
}

func (m Metric) GetUpstreamTime() time.Duration {
	return m.UpstreamTime
}
