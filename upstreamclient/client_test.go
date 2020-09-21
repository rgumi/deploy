package upstreamclient

import (
	"testing"
	"time"
)

func Test_NewClient(t *testing.T) {
	uc := NewClient(100, 5*time.Second, 30*time.Second, "", false)
	client := uc.GetClient()
	expectedTimeout := 5000 * time.Millisecond
	if client.Timeout != expectedTimeout {
		t.Errorf("Client is not configured correctly. Got: %v. Expected: %v", client.Timeout, expectedTimeout)
	}
}
