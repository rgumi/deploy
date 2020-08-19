package upstreamclient

import (
	"testing"
	"time"
)

func Test_UpstreamClient(t *testing.T) {

	// get new instance of client

	uc := New()
	for i := 0; i < 5; i++ {
		go uc.Send(nil, nil, "get", "http://localhost:9090")
	}
	time.Sleep(5 * time.Second)
}
