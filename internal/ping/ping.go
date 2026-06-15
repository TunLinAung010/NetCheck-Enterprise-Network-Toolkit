package ping

import (
	"context"
	"fmt"
	"math"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/pkg/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type Result struct {
	Host       string
	IP         string
	RTT        []time.Duration
	Min        time.Duration
	Max        time.Duration
	Avg        time.Duration
	Jitter     time.Duration
	PacketLoss float64
	Sent       int
	Received   int
}

type Pinger struct {
	host     string
	count    int
	interval time.Duration
	timeout  time.Duration
}

func New(host string, count int, interval, timeout time.Duration) *Pinger {
	return &Pinger{
		host:     host,
		count:    count,
		interval: interval,
		timeout:  timeout,
	}
}

func (p *Pinger) Run(ctx context.Context) (*Result, error) {
	ip, err := net.ResolveIPAddr("ip", p.host)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve %s: %w", p.host, err)
	}

	isIPv6 := ip.IP.To4() == nil
	result := &Result{
		Host: p.host,
		IP:   ip.String(),
	}

	var conn *icmp.PacketConn
	if isIPv6 {
		conn, err = icmp.ListenPacket("ip6:ipv6-icmp", "::")
	} else {
		conn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}
	defer conn.Close()

	var rtts []time.Duration
	var mu sync.Mutex

	receiveDone := make(chan struct{})
	go func() {
		defer close(receiveDone)
		recvBuf := make([]byte, 1500)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, _, err := conn.ReadFrom(recvBuf)
			if err != nil {
				continue
			}
			proto := 1
			if isIPv6 {
				proto = 58
			}
			msg, err := icmp.ParseMessage(proto, recvBuf[:n])
			if err != nil {
				continue
			}
			var replyType icmp.Type
			if isIPv6 {
				replyType = ipv6.ICMPTypeEchoReply
			} else {
				replyType = ipv4.ICMPTypeEchoReply
			}
			if msg.Type != replyType {
				continue
			}
			echo, ok := msg.Body.(*icmp.Echo)
			if !ok {
				continue
			}
			mu.Lock()
			rtt := time.Since(p.timeBase(echo.Seq))
			if rtt < p.timeout {
				rtts = append(rtts, rtt)
			}
			mu.Unlock()
		}
	}()

	for i := 0; i < p.count; i++ {
		select {
		case <-ctx.Done():
			return p.processResult(result, rtts), ctx.Err()
		default:
		}

		var reqType icmp.Type
		if isIPv6 {
			reqType = ipv6.ICMPTypeEchoRequest
		} else {
			reqType = ipv4.ICMPType(8)
		}

		echo := icmp.Message{
			Type: reqType,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  i,
				Data: []byte("NetCheck"),
			},
		}

		msgBytes, err := echo.Marshal(nil)
		if err != nil {
			logger.Warn("failed to marshal echo: %v", err)
			continue
		}

		if _, err := conn.WriteTo(msgBytes, ip); err != nil {
			logger.Warn("failed to send ping: %v", err)
			continue
		}

		result.Sent++

		time.Sleep(p.interval)
	}

	time.Sleep(200 * time.Millisecond)
	close(receiveDone)

	return p.processResult(result, rtts), nil
}

func (p *Pinger) timeBase(seq int) time.Time {
	return time.Now().Add(-time.Duration(seq) * time.Millisecond)
}

func (p *Pinger) processResult(result *Result, rtts []time.Duration) *Result {
	result.Received = len(rtts)
	result.RTT = rtts

	if len(rtts) == 0 {
		result.PacketLoss = 100
		return result
	}

	sort.Slice(rtts, func(i, j int) bool {
		return rtts[i] < rtts[j]
	})

	result.Min = rtts[0]
	result.Max = rtts[len(rtts)-1]

	var total time.Duration
	for _, r := range rtts {
		total += r
	}
	result.Avg = total / time.Duration(len(rtts))

	var jitterTotal float64
	mean := float64(result.Avg)
	for _, r := range rtts {
		jitterTotal += math.Abs(float64(r) - mean)
	}
	result.Jitter = time.Duration(jitterTotal / float64(len(rtts)))

	if result.Sent > 0 {
		result.PacketLoss = float64(result.Sent-result.Received) / float64(result.Sent) * 100
	}

	return result
}

func (r *Result) String() string {
	return fmt.Sprintf("PING %s (%s)\n", r.Host, r.IP) +
		fmt.Sprintf("  Packets: Sent=%d, Received=%d, Lost=%.0f%%\n", r.Sent, r.Received, r.PacketLoss) +
		fmt.Sprintf("  RTT: Min=%v, Avg=%v, Max=%v, Jitter=%v\n", r.Min, r.Avg, r.Max, r.Jitter)
}
