package upstreamclient

import (
	"testing"
	"time"
)

func Test_NewClient(t *testing.T) {
	uc := NewClient()
	client := uc.GetClient()
	expectedTimeout := time.Duration(2000) * time.Millisecond
	if client.Timeout != expectedTimeout {
		t.Errorf("Client is not configured correctly. Got: %v. Expected: %v", client.Timeout, expectedTimeout)
	}
}
