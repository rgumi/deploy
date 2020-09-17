package upstreamclient

import (
	"testing"
	"time"
)

func Test_NewClient(t *testing.T) {
	uc := NewDefaultClient()
	client := uc.GetClient()
	expectedTimeout := 5000 * time.Millisecond
	if client.Timeout != expectedTimeout {
		t.Errorf("Client is not configured correctly. Got: %v. Expected: %v", client.Timeout, expectedTimeout)
	}
}
