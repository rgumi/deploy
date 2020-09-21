package upstreamclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

	"github.com/rgumi/depoy/metrics"

	log "github.com/sirupsen/logrus"
)

type ProxyFunction func(*http.Request) (*url.URL, error)

type upstreamClient struct {
	Client *http.Client
	// http[s]://host:port
	UseProxy bool
	Proxy    string
}
type UpstreamClient interface {
	Send(req *http.Request) (*http.Response, metrics.Metrics, error)
	GetClient() *http.Client
}

func NewDefaultClient() UpstreamClient {

	client := NewClient(100, 5000, 30000, "", false)
	return client
}

func NewClient(maxIdleConns int,
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

	client.configClient(maxIdleConns, timeout, idleConnTimeout, 0, 0, true)
	return client
}

func (u *upstreamClient) GetClient() *http.Client {
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
	maxIdleConns int,
	timeout,
	idleConnTimeout,
	responseHeaderTimeout,
	tlsHandshakeTimeout time.Duration,
	tlsVerify bool) {

	// if a proxy is required it will be set
	// if not Proxy attribute of http.Transport will be set to nil
	// to avoid checking
	var proxy ProxyFunction
	if uc.UseProxy {
		proxy = uc.getProxy
	} else {
		proxy = nil
	}

	uc.Client = &http.Client{
		Transport: &http.Transport{
			// Use the default proxy as configured in the system environment
			Proxy: proxy,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConnsPerHost:   maxIdleConns,
			MaxIdleConns:          maxIdleConns,
			IdleConnTimeout:       idleConnTimeout,
			ResponseHeaderTimeout: responseHeaderTimeout,
			TLSHandshakeTimeout:   tlsHandshakeTimeout,
			DisableCompression:    true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: tlsVerify,
			},
		},
		// do not follow redirects as they are forwarded to the downstream client
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: timeout,
	}
}

// Send a HTTP message to the upstream client and collect metrics
func (u *upstreamClient) Send(req *http.Request) (*http.Response, metrics.Metrics, error) {
	var m metrics.Metrics
	startOfRequest := time.Now()

	trace := &httptrace.ClientTrace{
		//GotConn: func(connInfo httptrace.GotConnInfo) {
		//	m.UpstreamAddr = connInfo.Conn.RemoteAddr().String()
		//},
		ConnectDone: func(network, addr string, err error) {
			if err == nil {
				m.UpstreamRequestTime = time.Since(startOfRequest).Milliseconds()
			}
		},

		GotFirstResponseByte: func() {
			m.UpstreamResponseTime = time.Since(startOfRequest).Milliseconds()
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Start Sending
	log.Trace("Sending request to upstream host")
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, m, err
	}
	log.Tracef("Successfully received response from upstream host: %d", resp.StatusCode)
	m.ResponseStatus = resp.StatusCode
	return resp, m, nil
}
