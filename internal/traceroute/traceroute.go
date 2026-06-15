package traceroute

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/pkg/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Method string

const (
	MethodICMP Method = "icmp"
	MethodTCP  Method = "tcp"
	MethodUDP  Method = "udp"
)

type Hop struct {
	Number   int
	IP       string
	Hostname string
	Latency  time.Duration
	Reached  bool
}

type Result struct {
	Host     string
	IP       string
	Hops     []Hop
	Method   Method
	MaxHops  int
	Complete bool
}

type Tracer struct {
	host    string
	method  Method
	maxHops int
	timeout time.Duration
	port    int
}

func New(host string, method Method, maxHops int, timeout time.Duration, port int) *Tracer {
	return &Tracer{
		host:    host,
		method:  method,
		maxHops: maxHops,
		timeout: timeout,
		port:    port,
	}
}

func (t *Tracer) Run(ctx context.Context) (*Result, error) {
	ip, err := net.ResolveIPAddr("ip", t.host)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve %s: %w", t.host, err)
	}

	result := &Result{
		Host:    t.host,
		IP:      ip.String(),
		Method:  t.method,
		MaxHops: t.maxHops,
	}

	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}
	defer conn.Close()

	for ttl := 1; ttl <= t.maxHops; ttl++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		hop := Hop{Number: ttl}

		echo := icmp.Message{
			Type: ipv4.ICMPType(8),
			Code: 0,
			Body: &icmp.Echo{
				ID:   ttl,
				Seq:  ttl,
				Data: []byte("NetCheck Trace"),
			},
		}

		msgBytes, err := echo.Marshal(nil)
		if err != nil {
			logger.Warn("failed to marshal echo: %v", err)
			continue
		}

		if err := ipv4.NewPacketConn(conn).SetTTL(ttl); err != nil {
			logger.Warn("failed to set TTL: %v", err)
			continue
		}

		start := time.Now()
		if _, err := conn.WriteTo(msgBytes, ip); err != nil {
			logger.Warn("failed to send probe: %v", err)
			continue
		}

		conn.SetReadDeadline(time.Now().Add(t.timeout))
		buf := make([]byte, 1500)
		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			hop.IP = "*"
			hop.Hostname = "*"
			hop.Latency = t.timeout
			result.Hops = append(result.Hops, hop)
			continue
		}

		hop.Latency = time.Since(start)
		hop.IP = peer.String()

		if hostnames, err := net.LookupAddr(hop.IP); err == nil && len(hostnames) > 0 {
			hop.Hostname = hostnames[0]
		} else {
			hop.Hostname = hop.IP
		}

		msg, err := icmp.ParseMessage(1, buf[:n])
		if err == nil {
			switch msg.Type {
			case ipv4.ICMPTypeTimeExceeded:
				hop.Reached = false
			case ipv4.ICMPTypeEchoReply:
				hop.Reached = true
				result.Complete = true
			}
		}

		result.Hops = append(result.Hops, hop)

		if hop.Reached {
			break
		}
	}

	return result, nil
}

func (r *Result) String() string {
	s := fmt.Sprintf("Traceroute to %s (%s) [%s]:\n", r.Host, r.IP, r.Method)
	s += fmt.Sprintf("%-6s %-20s %-40s %s\n", "Hop", "IP", "Hostname", "Latency")
	for _, hop := range r.Hops {
		s += fmt.Sprintf("%-6d %-20s %-40s %v\n", hop.Number, hop.IP, hop.Hostname, hop.Latency)
	}
	return s
}
