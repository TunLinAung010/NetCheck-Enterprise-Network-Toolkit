package udp

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/netcheck/netcheck/pkg/logger"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type State string

const (
	OPEN             State = "OPEN"
	CLOSED           State = "CLOSED"
	FILTERED         State = "FILTERED"
	OPEN_OR_FILTERED State = "OPEN_OR_FILTERED"
)

type Protocol string

const (
	ProtocolDNS   Protocol = "dns"
	ProtocolNTP   Protocol = "ntp"
	ProtocolSNMP  Protocol = "snmp"
	ProtocolSyslog Protocol = "syslog"
	ProtocolGeneric Protocol = "generic"
)

type Result struct {
	Host     string
	Port     int
	State    State
	Protocol Protocol
	Latency  time.Duration
	Response []byte
}

type Checker struct {
	host     string
	port     int
	protocol Protocol
	timeout  time.Duration
	count    int
}

func New(host string, port int, protocol Protocol, timeout time.Duration, count int) *Checker {
	return &Checker{
		host:     host,
		port:     port,
		protocol: protocol,
		timeout:  timeout,
		count:    count,
	}
}

func DetectProtocol(port int) Protocol {
	switch port {
	case 53:
		return ProtocolDNS
	case 123:
		return ProtocolNTP
	case 161:
		return ProtocolSNMP
	case 514:
		return ProtocolSyslog
	default:
		return ProtocolGeneric
	}
}

func (c *Checker) Run(ctx context.Context) *Result {
	result := &Result{
		Host:     c.host,
		Port:     c.port,
		Protocol: c.protocol,
	}

	payload := c.buildProbe()
	addr := &net.UDPAddr{
		IP:   net.ParseIP(c.host),
		Port: c.port,
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		result.State = FILTERED
		return result
	}
	defer conn.Close()

	icmpConn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err == nil {
		defer icmpConn.Close()
	}

	icmpCh := make(chan bool, 1)
	if icmpConn != nil {
		go func() {
			buf := make([]byte, 1500)
			for {
				select {
				case <-ctx.Done():
					return
				default:
					icmpConn.SetReadDeadline(time.Now().Add(c.timeout))
					n, peer, err := icmpConn.ReadFrom(buf)
					if err != nil {
						return
					}
					msg, err := icmp.ParseMessage(1, buf[:n])
					if err != nil {
						continue
					}
					if msg.Type == ipv4.ICMPTypeDestinationUnreachable {
						if body, ok := msg.Body.(*icmp.DstUnreach); ok {
							if body.Data != nil && len(body.Data) >= 8 {
								originalDstPort := int(binary.BigEndian.Uint16(body.Data[2:4]))
								if originalDstPort == c.port {
									icmpCh <- true
									return
								}
							}
						}
					}
					_ = peer
				}
			}
		}()
	}

	start := time.Now()
	conn.SetWriteDeadline(time.Now().Add(c.timeout))
	if _, err := conn.Write(payload); err != nil {
		result.State = FILTERED
		return result
	}

	conn.SetReadDeadline(time.Now().Add(c.timeout))
	response := make([]byte, 1500)

	readCh := make(chan struct {
		n   int
		err error
	}, 1)

	go func() {
		n, err := conn.Read(response)
		readCh <- struct {
			n   int
			err error
		}{n, err}
	}()

	select {
	case <-readCh:
		result.Latency = time.Since(start)
		result.State = OPEN
		result.Response = response
	case <-icmpCh:
		result.State = CLOSED
	case <-time.After(c.timeout):
		result.State = OPEN_OR_FILTERED
	case <-ctx.Done():
		result.State = FILTERED
	}

	return result
}

func (c *Checker) buildProbe() []byte {
	switch c.protocol {
	case ProtocolDNS:
		return buildDNSProbe()
	case ProtocolNTP:
		return buildNTPProbe()
	case ProtocolSNMP:
		return buildSNMPProbe()
	case ProtocolSyslog:
		return buildSyslogProbe()
	default:
		return []byte("NetCheck UDP Probe")
	}
}

func buildDNSProbe() []byte {
	buf := new(bytes.Buffer)
	id := uint16(time.Now().UnixNano() & 0xFFFF)
	binary.Write(buf, binary.BigEndian, id)
	flags := uint16(0x0100)
	binary.Write(buf, binary.BigEndian, flags)
	qdcount := uint16(1)
	binary.Write(buf, binary.BigEndian, qdcount)
	binary.Write(buf, binary.BigEndian, uint16(0))
	binary.Write(buf, binary.BigEndian, uint16(0))
	binary.Write(buf, binary.BigEndian, uint16(0))

	for _, b := range []byte("google.com") {
		if b == '.' {
			buf.WriteByte(6)
		} else {
			buf.WriteByte(b)
		}
	}
	buf.WriteByte(0)

	qtype := uint16(1)
	qclass := uint16(1)
	binary.Write(buf, binary.BigEndian, qtype)
	binary.Write(buf, binary.BigEndian, qclass)

	return buf.Bytes()
}

func buildNTPProbe() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(0xE3)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	for i := 0; i < 44; i++ {
		buf.WriteByte(0x00)
	}
	return buf.Bytes()
}

func buildSNMPProbe() []byte {
	return []byte{
		0x30, 0x26, 0x02, 0x01, 0x01, 0x04, 0x06, 0x70,
		0x75, 0x62, 0x6c, 0x69, 0x63, 0xa0, 0x19, 0x02,
		0x04, 0x4f, 0xae, 0x2f, 0x0a, 0x02, 0x01, 0x00,
		0x02, 0x01, 0x00, 0x30, 0x0b, 0x30, 0x09, 0x06,
		0x05, 0x2b, 0x06, 0x01, 0x02, 0x01, 0x05, 0x00,
	}
}

func buildSyslogProbe() []byte {
	ts := time.Now().Format(time.RFC3339)
	msg := fmt.Sprintf("<14>1 %s localhost NetCheck - - probe", ts)
	return []byte(msg)
}

func BatchCheck(ctx context.Context, host string, ports []int, timeout time.Duration, workers int) []*Result {
	logger.Info("scanning %d UDP ports on %s with %d workers", len(ports), host, workers)
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
				proto := DetectProtocol(port)
				checker := New(host, port, proto, timeout, 1)
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

	logger.Info("UDP scan complete for %s", host)
	return results
}

func (r *Result) String() string {
	return fmt.Sprintf("UDP %s:%d [%s] (proto=%s, latency=%v)", r.Host, r.Port, r.State, r.Protocol, r.Latency)
}
