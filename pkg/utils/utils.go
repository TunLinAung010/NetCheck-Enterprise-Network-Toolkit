package utils

import (
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"
)

func ResolveHost(host string) (string, error) {
	ips, err := net.LookupHost(host)
	if err != nil {
		return "", fmt.Errorf("unable to resolve %s: %w", host, err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no addresses found for %s", host)
	}
	return ips[0], nil
}

func ResolveIP(host string) (net.IP, error) {
	addr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve %s: %w", host, err)
	}
	return addr.IP, nil
}

func LookupRDNS(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ip
	}
	return names[0]
}

func ParseCIDR(cidr string) ([]net.IP, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}
	var ips []net.IP
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		n := make(net.IP, len(ip))
		copy(n, ip)
		ips = append(ips, n)
	}
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
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

func IsPrivileged() bool {
	if os.Geteuid() == 0 {
		return true
	}
	return false
}

func Stats(durations []time.Duration) (min, max, avg time.Duration, jitter time.Duration, loss float64, sent int) {
	if len(durations) == 0 {
		return 0, 0, 0, 0, 100, 0
	}
	sent = len(durations)
	var total time.Duration
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	min = durations[0]
	max = durations[len(durations)-1]
	for _, d := range durations {
		total += d
	}
	avg = total / time.Duration(len(durations))

	var jitterTotal float64
	mean := float64(avg)
	for _, d := range durations {
		jitterTotal += math.Abs(float64(d) - mean)
	}
	if len(durations) > 0 {
		jitter = time.Duration(jitterTotal / float64(len(durations)))
	}
	return
}

func WaitForSignal() os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	return <-sigCh
}

func IsIPv6(host string) bool {
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.To4() == nil
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return false
	}
	for _, a := range addrs {
		ip := net.ParseIP(a)
		if ip != nil && ip.To4() == nil {
			return true
		}
	}
	return false
}
