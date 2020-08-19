package upstreamclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"

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
	Send(req *http.Request) (*http.Response, Metric, error)
	GetClient() *http.Client
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
	maxIdleConns,
	idleConnTimeoutMs,
	responseHeaderTimeoutMs,
	tlsHandshakeTimeoutMs int,
	tlsVerify bool,
) {

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
			IdleConnTimeout:       time.Duration(idleConnTimeoutMs) * time.Millisecond,
			ResponseHeaderTimeout: time.Duration(responseHeaderTimeoutMs) * time.Millisecond,
			TLSHandshakeTimeout:   time.Duration(tlsHandshakeTimeoutMs) * time.Millisecond,
			DisableCompression:    true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: tlsVerify,
			},
		},
	}
}

func New() UpstreamClient {

	client := new(upstreamClient)
	client.UseProxy = false
	client.configClient(100, 30000, 1000, 1000, true)
	return client
}

// Send a HTTP message to the upstream client and collect metrics
func (u *upstreamClient) Send(req *http.Request) (*http.Response, Metric, error) {
	var m Metric

	trace := &httptrace.ClientTrace{
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			m.UpstreamIP = dnsInfo.Addrs
			log.Debugf("DNS Info: %+v", dnsInfo)
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			log.Debugf("Got Conn: %+v", connInfo)
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Start Sending
	log.Debug("Sending request to upstream host")
	start := time.Now()
	resp, err := u.Client.Do(req)
	if err != nil {
		log.Errorf("Upstream request connection failure")
		return nil, Metric{}, err
	}

	connTime := time.Now().Sub(start)

	log.Debugf("%s %s%s %s %s %d %d %v", req.Proto, req.Method, req.RemoteAddr, req.URL,
		req.Header.Get("X-Forwarded-For"), resp.StatusCode, resp.ContentLength,
		connTime)

	m.UpstreamTime = connTime
	m.UpstreamStatus = resp.StatusCode
	return resp, m, nil
}
