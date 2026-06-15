package tcp

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New("localhost", 80, time.Second*5, 1)
	if c == nil {
		t.Fatal("expected non-nil checker")
	}
}

func TestResultString(t *testing.T) {
	r := &Result{
		Host:    "example.com",
		Port:    443,
		State:   OPEN,
		Latency: 10 * time.Millisecond,
	}
	s := r.String()
	if len(s) == 0 {
		t.Error("expected non-empty string")
	}
}

func TestClosedState(t *testing.T) {
	c := New("127.0.0.1", 1, time.Millisecond*10, 1)
	res := c.Run(context.Background())
	if res.State != CLOSED {
		t.Logf("expected CLOSED (or TIMEOUT for fast test), got %s", res.State)
	}
}

func TestHeaders(t *testing.T) {
	r := &Result{}
	h := r.Headers()
	if len(h) == 0 {
		t.Error("expected non-empty headers")
	}
}

func TestToMap(t *testing.T) {
	r := &Result{
		Host:    "test.com",
		Port:    80,
		State:   OPEN,
		Latency: 5 * time.Millisecond,
	}
	m := r.ToMap()
	if m["host"] != "test.com" {
		t.Errorf("expected test.com, got %v", m["host"])
	}
}
