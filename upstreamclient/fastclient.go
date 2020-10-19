package upstreamclient

import (
	"crypto/tls"
	"flag"
	"time"

	"github.com/rgumi/depoy/metrics"
	"github.com/valyala/fasthttp"
)

var (
	MaxIdleConnsPerHost, MaxIdleConns int
	TLSVerfiy                         bool
	DisableKeepAlives                 bool
	currentTime                       *time.Time
)

func init() {
	flag.IntVar(&MaxIdleConnsPerHost, "client.idleHostConns", 1024, "defines the maxIdleConnsPerHost")
	flag.IntVar(&MaxIdleConns, "client.idleConns", 1024, "defines the maxIdleConns")
	flag.BoolVar(&TLSVerfiy, "client.tlsVerify", false, "defines if tls should be verified")
	flag.BoolVar(&DisableKeepAlives, "client.keepAlives", true, "defines if http-keep-alive")
}

type Upstreamclient struct {
	client *fasthttp.Client
}

func NewUpstreamclient(
	readTimeout, writeTimeout, idleTimeout time.Duration,
	maxIdleConnsPerHost int, tlsVerify bool) *Upstreamclient {

	return &Upstreamclient{
		client: &fasthttp.Client{
			NoDefaultUserAgentHeader:      true,
			DisablePathNormalizing:        false,
			DisableHeaderNamesNormalizing: false,
			ReadTimeout:                   readTimeout,
			WriteTimeout:                  writeTimeout,
			TLSConfig: &tls.Config{
				InsecureSkipVerify: tlsVerify,
			},
			MaxConnsPerHost:           maxIdleConnsPerHost,
			MaxIdleConnDuration:       idleTimeout,
			MaxConnDuration:           0, // unlimited
			MaxIdemponentCallAttempts: 2,
		},
	}

}

func (c *Upstreamclient) Send(req *fasthttp.Request, m *metrics.Metrics) (*fasthttp.Response, error) {
	start := time.Now()
	resp := fasthttp.AcquireResponse()
	if err := c.client.Do(req, resp); err != nil {
		return nil, err
	}
	m.UpstreamResponseTime = time.Since(start).Milliseconds()
	return resp, nil
}
