package mtr

import (
	"context"
	"fmt"
	"math"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/pkg/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type HopStat struct {
	Number  int
	IP      string
	Hostname string
	Sent    int
	Loss    float64
	Last    time.Duration
	Min     time.Duration
	Max     time.Duration
	Avg     time.Duration
	Jitter  time.Duration
}

type Result struct {
	Host    string
	IP      string
	Hops    []HopStat
	Cycles  int
	Running bool
	mu      sync.Mutex
}

type MTR struct {
	host     string
	cycles   int
	interval time.Duration
	timeout  time.Duration
	maxHops  int
}

func New(host string, cycles int, interval, timeout time.Duration, maxHops int) *MTR {
	return &MTR{
		host:     host,
		cycles:   cycles,
		interval: interval,
		timeout:  timeout,
		maxHops:  maxHops,
	}
}

func (m *MTR) Run(ctx context.Context) (*Result, error) {
	ip, err := net.ResolveIPAddr("ip", m.host)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve %s: %w", m.host, err)
	}

	result := &Result{
		Host: m.host,
		IP:   ip.String(),
	}

	hopData := make(map[int][]time.Duration)
	var hopMu sync.Mutex

	for cycle := 0; cycle < m.cycles; cycle++ {
		select {
		case <-ctx.Done():
			return m.finalize(result, hopData), ctx.Err()
		default:
		}

		conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		if err != nil {
			logger.Warn("failed to listen: %v", err)
			continue
		}

		for ttl := 1; ttl <= m.maxHops; ttl++ {
			select {
			case <-ctx.Done():
				conn.Close()
				return m.finalize(result, hopData), ctx.Err()
			default:
			}

			echo := icmp.Message{
				Type: ipv4.ICMPType(8),
				Code: 0,
				Body: &icmp.Echo{
					ID:   cycle + 1,
					Seq:  ttl,
					Data: []byte("NetCheck MTR"),
				},
			}

			msgBytes, _ := echo.Marshal(nil)
			ipv4.NewPacketConn(conn).SetTTL(ttl)

			start := time.Now()
			conn.WriteTo(msgBytes, ip)
			conn.SetReadDeadline(time.Now().Add(m.timeout))

			buf := make([]byte, 1500)
			_, _, err := conn.ReadFrom(buf)
			rtt := time.Since(start)

			hopMu.Lock()
			if err == nil {
				hopData[ttl] = append(hopData[ttl], rtt)
			} else {
				hopData[ttl] = append(hopData[ttl], m.timeout)
			}
			hopMu.Unlock()

			if cycle == 0 {
				hop := HopStat{Number: ttl}
				addrs, _ := net.LookupAddr(ip.String())
				if len(addrs) > 0 {
					hop.Hostname = addrs[0]
				} else {
					hop.Hostname = ip.String()
				}
				hop.IP = ip.String()

				result.mu.Lock()
				result.Hops = append(result.Hops, hop)
				result.mu.Unlock()
			}
		}

		conn.Close()

		if m.cycles > 1 {
			result.mu.Lock()
			result.Cycles = cycle + 1
			result.Running = true
			result.mu.Unlock()

			displayResult(result, hopData)
			fmt.Println("---")
		}

		time.Sleep(m.interval)
	}

	return m.finalize(result, hopData), nil
}

func (m *MTR) finalize(result *Result, hopData map[int][]time.Duration) *Result {
	result.Running = false
	result.Cycles = m.cycles

	for i := range result.Hops {
		ttl := result.Hops[i].Number
		if rtts, ok := hopData[ttl]; ok {
			stats := calculateStats(rtts, m.timeout)
			result.Hops[i].Sent = stats.sent
			result.Hops[i].Loss = stats.loss
			result.Hops[i].Last = stats.last
			result.Hops[i].Min = stats.min
			result.Hops[i].Max = stats.max
			result.Hops[i].Avg = stats.avg
			result.Hops[i].Jitter = stats.jitter
		}
	}

	return result
}

type hopStats struct {
	sent   int
	loss   float64
	last   time.Duration
	min    time.Duration
	max    time.Duration
	avg    time.Duration
	jitter time.Duration
}

func calculateStats(rtts []time.Duration, timeout time.Duration) hopStats {
	var s hopStats
	s.sent = len(rtts)
	if s.sent == 0 {
		s.loss = 100
		s.min = timeout
		s.max = timeout
		s.avg = timeout
		return s
	}

	var validRTTs []time.Duration
	for _, rtt := range rtts {
		if rtt < timeout {
			validRTTs = append(validRTTs, rtt)
		}
	}

	s.last = rtts[len(rtts)-1]
	if s.last >= timeout {
		s.last = 0
	}

	if len(validRTTs) == 0 {
		s.loss = 100
		return s
	}

	s.loss = float64(s.sent-len(validRTTs)) / float64(s.sent) * 100

	sort.Slice(validRTTs, func(i, j int) bool {
		return validRTTs[i] < validRTTs[j]
	})
	s.min = validRTTs[0]
	s.max = validRTTs[len(validRTTs)-1]

	var total time.Duration
	for _, r := range validRTTs {
		total += r
	}
	s.avg = total / time.Duration(len(validRTTs))

	var jitterTotal float64
	mean := float64(s.avg)
	for _, r := range validRTTs {
		jitterTotal += math.Abs(float64(r) - mean)
	}
	s.jitter = time.Duration(jitterTotal / float64(len(validRTTs)))

	return s
}

func displayResult(result *Result, hopData map[int][]time.Duration) {
	fmt.Printf("MTR to %s (%s) - Cycle %d\n", result.Host, result.IP, result.Cycles)
	fmt.Printf("%-4s %-20s %-8s %-8s %-8s %-8s %-8s %s\n",
		"Hop", "Host", "Loss%", "Last", "Avg", "Best", "Worst", "Jitter")
	for _, hop := range result.Hops {
		if rtts, ok := hopData[hop.Number]; ok {
			stats := calculateStats(rtts, time.Second*5)
			fmt.Printf("%-4d %-20s %-8.1f %-8v %-8v %-8v %-8v %v\n",
				hop.Number, hop.Hostname, stats.loss, stats.last, stats.avg, stats.min, stats.max, stats.jitter)
		}
	}
}

func (r *Result) String() string {
	s := fmt.Sprintf("MTR Report for %s (%s)\n", r.Host, r.IP)
	s += fmt.Sprintf("Cycles: %d\n", r.Cycles)
	s += fmt.Sprintf("%-4s %-20s %-8s %-8s %-8s %-8s %-8s %s\n",
		"Hop", "Host", "Loss%", "Last", "Avg", "Best", "Worst", "Jitter")
	for _, hop := range r.Hops {
		s += fmt.Sprintf("%-4d %-20s %-8.1f %-8v %-8v %-8v %-8v %v\n",
			hop.Number, hop.Hostname, hop.Loss, hop.Last, hop.Avg, hop.Min, hop.Max, hop.Jitter)
	}
	return s
}
