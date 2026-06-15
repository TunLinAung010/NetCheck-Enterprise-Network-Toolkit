package ping

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	p := New("localhost", 4, time.Second, time.Second*5)
	if p == nil {
		t.Fatal("expected non-nil pinger")
	}
}

func TestResultProcess(t *testing.T) {
	r := &Result{
		Host: "localhost",
		IP:   "127.0.0.1",
		Sent: 4,
	}
	rtts := []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond, 40 * time.Millisecond}
	p := &Pinger{}
	result := p.processResult(r, rtts)
	if result.Min != 10*time.Millisecond {
		t.Errorf("expected min 10ms, got %v", result.Min)
	}
	if result.Max != 40*time.Millisecond {
		t.Errorf("expected max 40ms, got %v", result.Max)
	}
	if result.Avg != 25*time.Millisecond {
		t.Errorf("expected avg 25ms, got %v", result.Avg)
	}
	if result.PacketLoss != 0 {
		t.Errorf("expected loss 0, got %f", result.PacketLoss)
	}
}

func TestResultString(t *testing.T) {
	r := &Result{
		Host:       "example.com",
		IP:         "93.184.216.34",
		Sent:       4,
		Received:   4,
		Min:        10 * time.Millisecond,
		Max:        30 * time.Millisecond,
		Avg:        20 * time.Millisecond,
		Jitter:     5 * time.Millisecond,
		PacketLoss: 0,
	}
	s := r.String()
	if len(s) == 0 {
		t.Error("expected non-empty string")
	}
}

func TestHeaders(t *testing.T) {
	r := &Result{}
	h := r.Headers()
	if len(h) == 0 {
		t.Error("expected non-empty headers")
	}
}
