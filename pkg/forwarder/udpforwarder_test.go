package udpforwarder

import (
	"testing"
)

func TestForward(t *testing.T) {
	forwarder, err := Forward(
		"127.0.0.1:12345",
		"1.1.1.1:12345",
		DefaultTimeout,
		"foo",
	)
	if forwarder == nil || err != nil {
		t.Errorf("Failed to isntantiate Forwarder")
	}
}
