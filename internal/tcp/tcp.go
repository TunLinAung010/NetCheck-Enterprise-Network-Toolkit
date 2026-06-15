package tcp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/netcheck/netcheck/pkg/logger"
)

type State string

const (
	OPEN    State = "OPEN"
	CLOSED  State = "CLOSED"
	TIMEOUT State = "TIMEOUT"
)

type Result struct {
	Host     string
	Port     int
	State    State
	Latency  time.Duration
	Min      time.Duration
	Max      time.Duration
	Avg      time.Duration
	Samples  []time.Duration
	Attempts int
}

type Checker struct {
	host    string
	port    int
	timeout time.Duration
	count   int
}

func New(host string, port int, timeout time.Duration, count int) *Checker {
	return &Checker{
		host:    host,
		port:    port,
		timeout: timeout,
		count:   count,
	}
}

func (c *Checker) Run(ctx context.Context) *Result {
	result := &Result{
		Host: c.host,
		Port: c.port,
	}

	var latencies []time.Duration
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < c.count; i++ {
		select {
		case <-ctx.Done():
			result.State = TIMEOUT
			return result
		default:
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()

			addr := net.JoinHostPort(c.host, fmt.Sprintf("%d", c.port))
			conn, err := net.DialTimeout("tcp", addr, c.timeout)
			if err != nil {
				mu.Lock()
				latencies = append(latencies, 0)
				mu.Unlock()
				return
			}
			defer conn.Close()

			latency := time.Since(start)
			mu.Lock()
			latencies = append(latencies, latency)
			mu.Unlock()
		}()
		wg.Wait()
	}

	if len(latencies) == 0 {
		result.State = TIMEOUT
		return result
	}

	result.Attempts = c.count

	var validCount int
	var total time.Duration
	result.Min = c.timeout
	for _, l := range latencies {
		if l == 0 {
			continue
		}
		validCount++
		result.Samples = append(result.Samples, l)
		if l < result.Min {
			result.Min = l
		}
		if l > result.Max {
			result.Max = l
		}
		total += l
	}

	if validCount == 0 {
		result.State = CLOSED
		return result
	}

	result.State = OPEN
	result.Avg = total / time.Duration(validCount)
	result.Latency = result.Avg

	return result
}

func (c *Checker) CheckPort(ctx context.Context, host string, port int) State {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return CLOSED
	}
	conn.Close()
	return OPEN
}

func BatchCheck(ctx context.Context, host string, ports []int, timeout time.Duration, workers int) []*Result {
	logger.Info("scanning %d TCP ports on %s with %d workers", len(ports), host, workers)
	results := make([]*Result, 0, len(ports))
	portCh := make(chan int, len(ports))
	resultCh := make(chan *Result, len(ports))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range portCh {
				select {
				case <-ctx.Done():
					return
				default:
				}
				checker := New(host, port, timeout, 1)
				res := checker.Run(ctx)
				resultCh <- res
			}
		}()
	}

	for _, port := range ports {
		portCh <- port
	}
	close(portCh)

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for res := range resultCh {
		results = append(results, res)
	}

	logger.Info("TCP scan complete for %s: %d/%d ports open", host, countOpen(results), len(ports))
	return results
}

func countOpen(results []*Result) int {
	count := 0
	for _, r := range results {
		if r.State == OPEN {
			count++
		}
	}
	return count
}

func (r *Result) String() string {
	return fmt.Sprintf("TCP %s:%d [%s] (latency=%v)", r.Host, r.Port, r.State, r.Latency)
}
