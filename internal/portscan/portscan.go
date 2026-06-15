package portscan

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/netcheck/netcheck/internal/tcp"
	"github.com/netcheck/netcheck/internal/udp"
	"github.com/netcheck/netcheck/pkg/logger"
)

type ScanType string

const (
	ScanTCP ScanType = "tcp"
	ScanUDP ScanType = "udp"
)

type PortResult struct {
	Port    int
	Proto   ScanType
	State   string
	Latency time.Duration
	Service string
}

type Scanner struct {
	host    string
	ports   []int
	scanType ScanType
	timeout time.Duration
	workers int
}

func New(host string, ports []int, scanType ScanType, timeout time.Duration, workers int) *Scanner {
	return &Scanner{
		host:     host,
		ports:    ports,
		scanType: scanType,
		timeout:  timeout,
		workers:  workers,
	}
}

func commonService(port int) string {
	services := map[int]string{
		21:   "FTP",
		22:   "SSH",
		23:   "Telnet",
		25:   "SMTP",
		53:   "DNS",
		80:   "HTTP",
		110:  "POP3",
		123:  "NTP",
		143:  "IMAP",
		161:  "SNMP",
		443:  "HTTPS",
		514:  "Syslog",
		993:  "IMAPS",
		995:  "POP3S",
		1433: "MSSQL",
		3306: "MySQL",
		3389: "RDP",
		5432: "PostgreSQL",
		6379: "Redis",
		8080: "HTTP-Alt",
		8443: "HTTPS-Alt",
		9090: "Prometheus",
		27017: "MongoDB",
	}
	if svc, ok := services[port]; ok {
		return svc
	}
	return ""
}

func (s *Scanner) Run(ctx context.Context) ([]PortResult, error) {
	logger.Info("starting %s port scan on %s for %d ports", s.scanType, s.host, len(s.ports))

	portCh := make(chan int, len(s.ports))
	resultCh := make(chan PortResult, len(s.ports))
	var wg sync.WaitGroup

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range portCh {
				select {
				case <-ctx.Done():
					return
				default:
				}

				var state string
				var latency time.Duration

				if s.scanType == ScanTCP {
					checker := tcp.New(s.host, port, s.timeout, 1)
					res := checker.Run(ctx)
					state = string(res.State)
					latency = res.Latency
				} else {
					proto := udp.DetectProtocol(port)
					checker := udp.New(s.host, port, proto, s.timeout, 1)
					res := checker.Run(ctx)
					state = string(res.State)
					latency = res.Latency
				}

				resultCh <- PortResult{
					Port:    port,
					Proto:   s.scanType,
					State:   state,
					Latency: latency,
					Service: commonService(port),
				}
			}
		}()
	}

	for _, port := range s.ports {
		portCh <- port
	}
	close(portCh)

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var results []PortResult
	for res := range resultCh {
		results = append(results, res)
	}

	openCount := 0
	for _, r := range results {
		if r.State == "OPEN" {
			openCount++
		}
	}
	logger.Info("port scan complete: %d/%d ports open", openCount, len(s.ports))

	return results, nil
}

func (r PortResult) String() string {
	svc := r.Service
	if svc != "" {
		svc = " (" + svc + ")"
	}
	return fmt.Sprintf("%5d/%s %-15s %v%s", r.Port, r.Proto, r.State, r.Latency, svc)
}
