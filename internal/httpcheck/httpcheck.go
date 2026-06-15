package httpcheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"
)

type Result struct {
	URL           string
	StatusCode    int
	StatusText    string
	ResponseTime  time.Duration
	DNSLookup     time.Duration
	TCPConnect    time.Duration
	TLSHandshake  time.Duration
	Headers       map[string]string
	RedirectCount int
	BodySize      int64
	Error         string
}

type Checker struct {
	url         string
	timeout     time.Duration
	followRedirects bool
}

func New(url string, timeout time.Duration, followRedirects bool) *Checker {
	return &Checker{
		url:             url,
		timeout:         timeout,
		followRedirects: followRedirects,
	}
}

func (c *Checker) Run(ctx context.Context) *Result {
	result := &Result{URL: c.url}

	req, err := http.NewRequestWithContext(ctx, "GET", c.url, nil)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	var dnsStart, dnsEnd, tcpStart, tcpEnd, tlsStart, tlsEnd time.Time

	trace := &httptrace.ClientTrace{
		DNSStart:             func(_ httptrace.DNSStartInfo) { dnsStart = time.Now() },
		DNSDone:              func(_ httptrace.DNSDoneInfo) { dnsEnd = time.Now() },
		ConnectStart:         func(_, _ string) { tcpStart = time.Now() },
		ConnectDone:          func(_, _ string, _ error) { tcpEnd = time.Now() },
		TLSHandshakeStart:    func() { tlsStart = time.Now() },
		TLSHandshakeDone:     func(_ tls.ConnectionState, _ error) { tlsEnd = time.Now() },
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	client := &http.Client{
		Timeout: c.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			result.RedirectCount = len(via)
			if !c.followRedirects && len(via) > 0 {
				return http.ErrUseLastResponse
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Error = fmt.Sprintf("request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.StatusText = http.StatusText(resp.StatusCode)
	result.Headers = make(map[string]string)

	for key := range resp.Header {
		if len(key) < 50 {
			result.Headers[key] = resp.Header.Get(key)
		}
	}

	if !dnsStart.IsZero() && !dnsEnd.IsZero() {
		result.DNSLookup = dnsEnd.Sub(dnsStart)
	}
	if !tcpStart.IsZero() && !tcpEnd.IsZero() {
		result.TCPConnect = tcpEnd.Sub(tcpStart)
	}
	if !tlsStart.IsZero() && !tlsEnd.IsZero() {
		result.TLSHandshake = tlsEnd.Sub(tlsStart)
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	result.BodySize = int64(len(body))

	return result
}

func (r *Result) String() string {
	s := fmt.Sprintf("HTTP Check for %s:\n", r.URL)
	if r.Error != "" {
		s += fmt.Sprintf("  Error: %s\n", r.Error)
		return s
	}
	s += fmt.Sprintf("  Status: %d %s\n", r.StatusCode, r.StatusText)
	s += fmt.Sprintf("  Response Time: %v\n", r.ResponseTime)
	s += fmt.Sprintf("  DNS Lookup: %v\n", r.DNSLookup)
	s += fmt.Sprintf("  TCP Connect: %v\n", r.TCPConnect)
	if r.TLSHandshake > 0 {
		s += fmt.Sprintf("  TLS Handshake: %v\n", r.TLSHandshake)
	}
	s += fmt.Sprintf("  Redirects: %d\n", r.RedirectCount)
	s += fmt.Sprintf("  Body Size: %d bytes\n", r.BodySize)
	if len(r.Headers) > 0 {
		s += fmt.Sprintf("  Headers:\n")
		for k, v := range r.Headers {
			if len(v) < 100 {
				s += fmt.Sprintf("    %s: %s\n", k, v)
			}
		}
	}
	return s
}


