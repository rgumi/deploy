package upstreamclient

type Metric struct {
	Method           string // HTTP-Method
	UpstreamStatus   int    // Status of the Response of the Upstream-Host
	DownstreamAddr   string // RemoteAddr of the downstream client
	UpstreamAddr     string // RemoteAddr of the upstream host
	ResponseSendTime int64  // Time ellapsed between receiving a response
	//	from upstream host and sending it to the downstream client
	GotFirstResponseByteTime int64 // Time ellapsed between sending the request
	// to upstream client and receiving a response
}
