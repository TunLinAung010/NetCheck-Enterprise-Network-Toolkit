package utils

import (
	"testing"
	"time"
)

func TestStatsEmpty(t *testing.T) {
	min, max, avg, jitter, loss, sent := Stats(nil)
	if min != 0 || max != 0 || avg != 0 {
		t.Error("expected zero values for empty input")
	}
	if loss != 100 {
		t.Errorf("expected loss 100, got %f", loss)
	}
	if sent != 0 {
		t.Errorf("expected sent 0, got %d", sent)
	}
	_ = jitter
}

func TestStats(t *testing.T) {
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
	}
	min, max, avg, jitter, loss, sent := Stats(durations)
	if min != 10*time.Millisecond {
		t.Errorf("expected min 10ms, got %v", min)
	}
	if max != 30*time.Millisecond {
		t.Errorf("expected max 30ms, got %v", max)
	}
	if avg != 20*time.Millisecond {
		t.Errorf("expected avg 20ms, got %v", avg)
	}
	if loss != 0 {
		t.Errorf("expected loss 0, got %f", loss)
	}
	if sent != 3 {
		t.Errorf("expected sent 3, got %d", sent)
	}
	_ = jitter
}
