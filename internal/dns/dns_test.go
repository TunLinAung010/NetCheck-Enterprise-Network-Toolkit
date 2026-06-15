package dns

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	r := New("google.com", "", time.Second*5)
	if r == nil {
		t.Fatal("expected non-nil resolver")
	}
}

func TestDNSLookup(t *testing.T) {
	r := New("google.com", "", time.Second*5)
	res := r.Run(context.Background())
	if res.Error != "" {
		t.Logf("DNS lookup had error: %s", res.Error)
	}
	if len(res.Records) == 0 && res.Error == "" {
		t.Error("expected records or error")
	}
}

func TestResultString(t *testing.T) {
	r := &Result{
		Domain:       "test.com",
		ResponseTime: 10 * time.Millisecond,
	}
	s := r.String()
	if len(s) == 0 {
		t.Error("expected non-empty string")
	}
}
