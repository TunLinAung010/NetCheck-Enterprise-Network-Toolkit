package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	l := New(INFO)
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	l := New(DEBUG)
	l.SetOutput(&buf)

	l.Debug("debug %s", "test")
	l.Info("info %s", "test")
	l.Warn("warn %s", "test")
	l.Error("error %s", "test")

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Error("expected DEBUG log")
	}
	if !strings.Contains(output, "INFO") {
		t.Error("expected INFO log")
	}
}

func TestJSONMode(t *testing.T) {
	var buf bytes.Buffer
	l := New(INFO)
	l.SetOutput(&buf)
	l.SetJSONMode(true)

	l.Info("test message")

	output := buf.String()
	if !strings.Contains(output, `"level":"INFO"`) {
		t.Error("expected JSON format")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		level Level
	}{
		{"debug", DEBUG},
		{"info", INFO},
		{"warn", WARN},
		{"error", ERROR},
		{"unknown", INFO},
	}
	for _, tt := range tests {
		l := ParseLevel(tt.input)
		if l != tt.level {
			t.Errorf("ParseLevel(%s) = %d, want %d", tt.input, l, tt.level)
		}
	}
}
