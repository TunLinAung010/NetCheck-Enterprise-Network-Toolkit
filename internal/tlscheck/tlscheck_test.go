package tlscheck

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New("example.com", 443, time.Second*10)
	if c == nil {
		t.Fatal("expected non-nil checker")
	}
}

func TestResultString(t *testing.T) {
	r := &Result{
		Host:          "example.com",
		Subject:       "Example",
		Issuer:        "CA",
		DaysRemaining: 90,
	}
	s := r.String()
	if len(s) == 0 {
		t.Error("expected non-empty string")
	}
}
