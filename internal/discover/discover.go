package discover

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/pkg/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type HostState string

const (
	Alive   HostState = "Alive"
	Down    HostState = "Down"
	Unknown HostState = "Unknown"
)

type HostResult struct {
	IP        string
	Hostname  string
	State     HostState
	Discovery string
	Latency   time.Duration
}

type Discoverer struct {
	cidr       string
	icmpEnabled bool
	tcpEnabled  bool
	arpEnabled  bool
	tcpPorts   []int
	timeout    time.Duration
	workers    int
}

func New(cidr string, icmp, tcp, arp bool, tcpPorts []int, timeout time.Duration, workers int) *Discoverer {
	return &Discoverer{
		cidr:       cidr,
		icmpEnabled: icmp,
		tcpEnabled:  tcp,
		arpEnabled:  arp,
		tcpPorts:   tcpPorts,
		timeout:    timeout,
		workers:    workers,
	}
}

func (d *Discoverer) Run(ctx context.Context) ([]HostResult, error) {
	ips, err := parseCIDR(d.cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	logger.Info("discovering hosts in %s (%d addresses)", d.cidr, len(ips))

	ipCh := make(chan string, len(ips))
	resultCh := make(chan HostResult, len(ips))
	var wg sync.WaitGroup

	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ipStr := range ipCh {
				select {
				case <-ctx.Done():
					return
				default:
				}

				result := HostResult{IP: ipStr}
				if names, err := net.LookupAddr(ipStr); err == nil && len(names) > 0 {
					result.Hostname = names[0]
				}

				if d.icmpEnabled {
					if pingHost(ipStr, d.timeout) {
						result.State = Alive
						result.Discovery = "ICMP"
						resultCh <- result
						continue
					}
				}

				if d.tcpEnabled && len(d.tcpPorts) > 0 {
					for _, port := range d.tcpPorts {
						addr := net.JoinHostPort(ipStr, fmt.Sprintf("%d", port))
						conn, err := net.DialTimeout("tcp", addr, d.timeout)
						if err == nil {
							conn.Close()
							result.State = Alive
							result.Discovery = fmt.Sprintf("TCP/%d", port)
							resultCh <- result
							break
						}
					}
					if result.State == Alive {
						continue
					}
				}

				result.State = Unknown
				resultCh <- result
			}
		}()
	}

	for _, ip := range ips {
		ipCh <- ip.String()
	}
	close(ipCh)

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var results []HostResult
	for res := range resultCh {
		results = append(results, res)
	}

	aliveCount := 0
	for _, r := range results {
		if r.State == Alive {
			aliveCount++
		}
	}
	logger.Info("discovery complete: %d/%d hosts alive", aliveCount, len(results))

	return results, nil
}

func parseCIDR(cidr string) ([]net.IP, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	var ips []net.IP
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		n := make(net.IP, len(ip))
		copy(n, ip)
		ips = append(ips, n)
	}
	if len(ips) > 2 {
		return ips[1 : len(ips)-1], nil
	}
	return ips, nil
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func pingHost(ipStr string, timeout time.Duration) bool {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false
	}
	defer conn.Close()

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	echo := icmp.Message{
		Type: ipv4.ICMPType(8),
		Code: 0,
		Body: &icmp.Echo{
			ID:   time.Now().Nanosecond() & 0xFFFF,
			Seq:  1,
			Data: []byte("NetCheck Discovery"),
		},
	}

	msgBytes, err := echo.Marshal(nil)
	if err != nil {
		return false
	}

	conn.SetWriteDeadline(time.Now().Add(timeout))
	if _, err := conn.WriteTo(msgBytes, &net.IPAddr{IP: ip}); err != nil {
		return false
	}

	conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 1500)
	n, _, err := conn.ReadFrom(buf)
	if err != nil {
		return false
	}

	msg, err := icmp.ParseMessage(1, buf[:n])
	if err != nil {
		return false
	}

	return msg.Type == ipv4.ICMPTypeEchoReply
}

func (r HostResult) String() string {
	host := r.Hostname
	if host == "" {
		host = "-"
	}
	return fmt.Sprintf("%-16s %-8s %-12s %s", r.IP, r.State, r.Discovery, host)
}
