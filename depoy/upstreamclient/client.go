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

func NewClient() UpstreamClient {

	client := new(upstreamClient)
	client.UseProxy = false
	client.configClient(100, 10000, 30000, 10000, 10000, true)
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
	maxIdleConns,
	timeoutMs,
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

			DisableCompression: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: tlsVerify,
			},
		},
		Timeout: time.Duration(timeoutMs) * time.Millisecond,
	}
}

// Send a HTTP message to the upstream client and collect metrics
func (u *upstreamClient) Send(req *http.Request) (*http.Response, Metric, error) {
	var m Metric
	startOfRequest := time.Now()

	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			m.UpstreamAddr = connInfo.Conn.RemoteAddr().String()
		},
		/*
			DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
				m.DNSDoneTime = time.Since(startOfRequest).Milliseconds()
			},
			ConnectDone: func(network, addr string, err error) {
				if err == nil {
					m.ConnDoneTime = time.Since(startOfRequest).Microseconds()
				}
			},
		*/
		GotFirstResponseByte: func() {
			m.GotFirstResponseByteTime = time.Since(startOfRequest).Milliseconds()
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Start Sending
	log.Debug("Sending request to upstream host")
	resp, err := u.Client.Do(req)
	if err != nil {
		return nil, m, err
	}

	m.UpstreamStatus = resp.StatusCode
	return resp, m, nil
}
