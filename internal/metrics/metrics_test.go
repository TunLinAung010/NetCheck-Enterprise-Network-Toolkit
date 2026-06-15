package metrics

import (
	"testing"
	"time"
)

func TestSetAvailability(t *testing.T) {
	SetAvailability("test", "ping", true)
	SetAvailability("test", "ping", false)
}

func TestSetLatency(t *testing.T) {
	SetLatency("test", "ping", 100*time.Millisecond)
}

func TestSetPacketLoss(t *testing.T) {
	SetPacketLoss("test", 50.0)
}

func TestObserveDuration(t *testing.T) {
	ObserveDuration("ping", 100*time.Millisecond)
}

func TestIncCheck(t *testing.T) {
	IncCheck("ping", "success")
	IncCheck("ping", "failure")
}
