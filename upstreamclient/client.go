package upstreamclient

import (
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"github.com/rgumi/depoy/metrics"

	log "github.com/sirupsen/logrus"
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
	timeAccuracy := time.Duration(*flag.Int64("client.timeAccuracy", 500, "time accuracy in milliseconds")) * time.Millisecond

	// start time goroutine
	go func(deviation time.Duration) {
		var ct time.Time
		ct = time.Now()
		currentTime = &ct
		for {
			select {
			case ct = <-time.After(deviation):
				currentTime = &ct
			}
		}
	}(timeAccuracy)

}

type UpstreamClient interface {
	Send(req *http.Request, m *metrics.Metrics) (*http.Response, *metrics.Metrics, error)
	GetClient() *http.Transport
}

type upstreamClient struct {
	Client   *http.Transport
	UseProxy bool
	Proxy    string
}

func NewDefaultClient() UpstreamClient {
	client := NewClient(MaxIdleConns, MaxIdleConnsPerHost, 5000, 30000, "", TLSVerfiy)
	return client
}

func NewClient(maxIdleConns, maxIdleConnsPerHost int,
	timeout, idleConnTimeout time.Duration,
	proxy string,
	tlsVerify bool) UpstreamClient {

	client := new(upstreamClient)

	if proxy != "" {
		client.UseProxy = true
		client.Proxy = proxy
	} else {
		client.UseProxy = false
		client.Proxy = ""
	}

	client.configClient(maxIdleConns, maxIdleConnsPerHost, timeout, idleConnTimeout, 0, 0, true)

	return client
}

func (u *upstreamClient) GetClient() *http.Transport {
	return u.Client
}

func (uc *upstreamClient) getProxy(_ *http.Request) (*url.URL, error) {
	if uc.Proxy == "" {
		log.Debug("No proxy is configured. Returning nil for http.Client")
		return nil, nil
	}
	proxy, err := url.Parse(uc.Proxy)
	if err != nil {
		log.Error("Unable to set proxy")
		return nil, err
	}
	log.Debug("Successfully created Proxy URL. Returning it for http.Client")
	return proxy, nil
}

func (uc *upstreamClient) configClient(
	maxIdleConns, maxIdleConnsPerHost int,
	timeout,
	idleConnTimeout,
	responseHeaderTimeout,
	tlsHandshakeTimeout time.Duration,
	tlsVerify bool) {

	// if a proxy is required it will be set
	// if not Proxy attribute of http.Transport will be set to nil
	// to avoid checking
	var proxy func(*http.Request) (*url.URL, error)
	if uc.UseProxy {
		proxy = uc.getProxy
	} else {
		proxy = nil
	}

	uc.Client = &http.Transport{
		Proxy: proxy,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		MaxIdleConns:          maxIdleConns,
		IdleConnTimeout:       idleConnTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: tlsVerify,
		},
		ForceAttemptHTTP2:  false,
		DisableCompression: true,
		DisableKeepAlives:  DisableKeepAlives,
	}
}

// Send a HTTP message to the upstream client and collect metrics
func (u *upstreamClient) Send(req *http.Request, m *metrics.Metrics) (*http.Response, *metrics.Metrics, error) {
	trace := &httptrace.ClientTrace{
		ConnectDone: func(network, addr string, err error) {
			if err == nil {
				m.UpstreamRequestTime = time.Since(*currentTime).Milliseconds()
			}
		},
		GotFirstResponseByte: func() {
			m.UpstreamResponseTime = time.Since(*currentTime).Milliseconds()
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	resp, err := u.Client.RoundTrip(req)
	return resp, m, err
}
