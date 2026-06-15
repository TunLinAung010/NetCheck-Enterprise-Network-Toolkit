package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/netcheck/netcheck/internal/httpcheck"
	"github.com/netcheck/netcheck/internal/ping"
	"github.com/netcheck/netcheck/internal/tcp"
	"github.com/netcheck/netcheck/internal/tlscheck"
)

type MonitorType string

const (
	MonitorPing  MonitorType = "ping"
	MonitorTCP   MonitorType = "tcp"
	MonitorHTTP  MonitorType = "http"
	MonitorTLS   MonitorType = "tls"
)

type DataPoint struct {
	Timestamp    time.Time
	Latency      time.Duration
	Available    bool
	StatusCode   int
	ErrorMessage string
}

type MonitorResult struct {
	Target     string
	Type       MonitorType
	Port       int
	DataPoints []DataPoint
	AvgLatency time.Duration
	MinLatency time.Duration
	MaxLatency time.Duration
	Uptime     float64
	Failures   int
	Total      int
}

type Monitor struct {
	target   string
	port     int
	monType  MonitorType
	interval time.Duration
	count    int

	mu        sync.Mutex
	data      []DataPoint
	done      chan struct{}
	running   bool
}

func New(target string, port int, monType MonitorType, interval time.Duration, count int) *Monitor {
	return &Monitor{
		target:   target,
		port:     port,
		monType:  monType,
		interval: interval,
		count:    count,
		done:     make(chan struct{}),
	}
}

func (m *Monitor) Start(ctx context.Context) <-chan DataPoint {
	ch := make(chan DataPoint, 100)
	m.running = true

	go func() {
		defer close(ch)
		defer func() { m.running = false }()

		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		iterations := 0
		for {
			if m.count > 0 && iterations >= m.count {
				break
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				dp := m.check(ctx)
				m.mu.Lock()
				m.data = append(m.data, dp)
				m.mu.Unlock()

				select {
				case ch <- dp:
				default:
				}

				iterations++
			}
		}
	}()

	return ch
}

func (m *Monitor) Stop() {
	if m.running {
		close(m.done)
	}
}

func (m *Monitor) check(ctx context.Context) DataPoint {
	dp := DataPoint{
		Timestamp: time.Now(),
	}

	checkCtx, cancel := context.WithTimeout(ctx, m.interval)
	defer cancel()

	switch m.monType {
	case MonitorPing:
		p := ping.New(m.target, 1, time.Second, time.Second*5)
		res, err := p.Run(checkCtx)
		if err != nil || res.Received == 0 {
			dp.Available = false
			if err != nil {
				dp.ErrorMessage = err.Error()
			}
		} else {
			dp.Available = true
			dp.Latency = res.Avg
		}

	case MonitorTCP:
		c := tcp.New(m.target, m.port, time.Second*5, 1)
		res := c.Run(checkCtx)
		dp.Available = res.State == tcp.OPEN
		dp.Latency = res.Latency
		if !dp.Available {
			dp.ErrorMessage = fmt.Sprintf("TCP %s", res.State)
		}

	case MonitorHTTP:
		c := httpcheck.New(fmt.Sprintf("https://%s", m.target), time.Second*5, true)
		res := c.Run(checkCtx)
		dp.Available = res.StatusCode >= 200 && res.StatusCode < 500
		dp.Latency = res.ResponseTime
		dp.StatusCode = res.StatusCode
		if !dp.Available {
			dp.ErrorMessage = fmt.Sprintf("HTTP %d", res.StatusCode)
		}

	case MonitorTLS:
		c := tlscheck.New(m.target, 443, time.Second*5)
		res := c.Run(checkCtx)
		dp.Available = res.Error == ""
		if !dp.Available {
			dp.ErrorMessage = res.Error
		}
	}

	return dp
}

func (m *Monitor) Result() *MonitorResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := &MonitorResult{
		Target: m.target,
		Type:   m.monType,
		Port:   m.port,
		DataPoints: m.data,
	}

	if len(m.data) == 0 {
		return result
	}

	result.Total = len(m.data)
	var totalLatency time.Duration
	result.MinLatency = time.Hour

	for _, dp := range m.data {
		if !dp.Available {
			result.Failures++
			continue
		}
		totalLatency += dp.Latency
		if dp.Latency < result.MinLatency {
			result.MinLatency = dp.Latency
		}
		if dp.Latency > result.MaxLatency {
			result.MaxLatency = dp.Latency
		}
	}

	successCount := result.Total - result.Failures
	if successCount > 0 {
		result.AvgLatency = totalLatency / time.Duration(successCount)
	}
	if result.Total > 0 {
		result.Uptime = float64(successCount) / float64(result.Total) * 100
	}

	return result
}

func (r *MonitorResult) String() string {
	s := fmt.Sprintf("Monitoring Report for %s", r.Target)
	if r.Port > 0 {
		s += fmt.Sprintf(":%d", r.Port)
	}
	s += fmt.Sprintf(" [%s]\n", r.Type)
	s += fmt.Sprintf("  Checks: %d, Failures: %d\n", r.Total, r.Failures)
	s += fmt.Sprintf("  Uptime: %.1f%%\n", r.Uptime)
	s += fmt.Sprintf("  Latency: Avg=%v, Min=%v, Max=%v\n", r.AvgLatency, r.MinLatency, r.MaxLatency)
	return s
}
