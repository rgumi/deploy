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
	SkipTLSVerify                     bool
	DisableKeepAlives                 bool
	currentTime                       *time.Time
)

func init() {
	flag.IntVar(&MaxIdleConnsPerHost, "client.idleHostConns", 1024, "defines the maxIdleConnsPerHost")
	flag.IntVar(&MaxIdleConns, "client.idleConns", 1024, "defines the maxIdleConns")
	flag.BoolVar(&SkipTLSVerify, "client.tlsVerify", true, "defines if tls verification should be skipped")
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
				InsecureSkipVerify: SkipTLSVerify,
			},
			MaxConnsPerHost:           maxIdleConnsPerHost,
			MaxIdleConnDuration:       idleTimeout,
			MaxConnDuration:           0, // unlimited
			MaxIdemponentCallAttempts: 2,
		},
	}

}

func (c *Upstreamclient) Send(req *fasthttp.Request, m *metrics.Metrics) (*fasthttp.Response, error) {
	resp := fasthttp.AcquireResponse()
	start := time.Now()
	if err := c.client.Do(req, resp); err != nil {
		return nil, err
	}
	m.UpstreamResponseTime = time.Since(start).Milliseconds()
	return resp, nil
}
