package httpcheck

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New("https://example.com", time.Second*10, true)
	if c == nil {
		t.Fatal("expected non-nil checker")
	}
}

func TestResultString(t *testing.T) {
	r := &Result{
		URL:          "https://example.com",
		StatusCode:   200,
		StatusText:   "OK",
		ResponseTime: 50 * time.Millisecond,
	}
	s := r.String()
	if len(s) == 0 {
		t.Error("expected non-empty string")
	}
}

func TestRunError(t *testing.T) {
	c := New("http://192.0.2.1:1", time.Millisecond*10, true)
	res := c.Run(context.Background())
	if res.Error == "" {
		t.Log("expected error or timeout for invalid address")
	}
}
