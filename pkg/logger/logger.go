package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

type Logger struct {
	mu       sync.Mutex
	level    Level
	out      io.Writer
	jsonMode bool
}

var defaultLogger = New(INFO)

func New(level Level) *Logger {
	return &Logger{
		level: level,
		out:   os.Stderr,
	}
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

func (l *Logger) SetJSONMode(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.jsonMode = enabled
}

func (l *Logger) Log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	if l.jsonMode {
		entry := fmt.Sprintf(`{"time":"%s","level":"%s","message":"%s"}%s`,
			ts, levelNames[level], fmt.Sprintf(msg, args...), "\n")
		fmt.Fprint(l.out, entry)
	} else {
		entry := fmt.Sprintf("%s [%s] %s%s", ts, levelNames[level], fmt.Sprintf(msg, args...), "\n")
		fmt.Fprint(l.out, entry)
	}
}

func (l *Logger) Debug(msg string, args ...interface{})  { l.Log(DEBUG, msg, args...) }
func (l *Logger) Info(msg string, args ...interface{})   { l.Log(INFO, msg, args...) }
func (l *Logger) Warn(msg string, args ...interface{})   { l.Log(WARN, msg, args...) }
func (l *Logger) Error(msg string, args ...interface{})  { l.Log(ERROR, msg, args...) }
func (l *Logger) Fatalf(msg string, args ...interface{}) { l.Log(FATAL, msg, args...); os.Exit(1) }

func SetLevel(level Level)                       { defaultLogger.SetLevel(level) }
func SetOutput(w io.Writer)                      { defaultLogger.SetOutput(w) }
func SetJSONMode(enabled bool)                   { defaultLogger.SetJSONMode(enabled) }
func Debug(msg string, args ...interface{})      { defaultLogger.Debug(msg, args...) }
func Info(msg string, args ...interface{})       { defaultLogger.Info(msg, args...) }
func Warn(msg string, args ...interface{})       { defaultLogger.Warn(msg, args...) }
func Error(msg string, args ...interface{})      { defaultLogger.Error(msg, args...) }
func Fatalf(msg string, args ...interface{})     { defaultLogger.Fatalf(msg, args...) }

func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

func init() {
	log.SetFlags(0)
	log.SetOutput(io.Discard)
}
