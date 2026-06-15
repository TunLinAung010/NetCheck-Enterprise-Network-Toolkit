package dns

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/netcheck/netcheck/pkg/logger"
)

type RecordType string

const (
	RecordA     RecordType = "A"
	RecordAAAA  RecordType = "AAAA"
	RecordMX    RecordType = "MX"
	RecordTXT   RecordType = "TXT"
	RecordNS    RecordType = "NS"
	RecordCNAME RecordType = "CNAME"
	RecordSOA   RecordType = "SOA"
)

type Record struct {
	Type  RecordType
	Value string
	TTL   uint32
}

type Result struct {
	Domain      string
	Records     []Record
	ResponseTime time.Duration
	Nameserver  string
	Error       string
}

type Resolver struct {
	domain     string
	nameserver string
	timeout    time.Duration
}

func New(domain string, nameserver string, timeout time.Duration) *Resolver {
	return &Resolver{
		domain:     domain,
		nameserver: nameserver,
		timeout:    timeout,
	}
}

func (r *Resolver) Run(ctx context.Context) *Result {
	result := &Result{
		Domain: r.domain,
	}

	var resolver *net.Resolver
	if r.nameserver != "" {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: r.timeout}
				return d.DialContext(ctx, "udp", r.nameserver+":53")
			},
		}
	} else {
		resolver = net.DefaultResolver
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	start := time.Now()

	lookups := []struct {
		typ  RecordType
		fn   func() ([]string, error)
	}{
		{RecordA, func() ([]string, error) {
			ips, err := resolver.LookupIPAddr(ctx, r.domain)
			if err != nil {
				return nil, err
			}
			var v4s []string
			for _, ip := range ips {
				if ip.IP.To4() != nil {
					v4s = append(v4s, ip.IP.String())
				}
			}
			if len(v4s) == 0 {
				return nil, fmt.Errorf("no A records")
			}
			return v4s, nil
		}},
		{RecordAAAA, func() ([]string, error) {
			ips, err := resolver.LookupIPAddr(ctx, r.domain)
			if err != nil {
				return nil, err
			}
			var v6s []string
			for _, ip := range ips {
				if ip.IP.To4() == nil && ip.IP.To16() != nil {
					v6s = append(v6s, ip.IP.String())
				}
			}
			if len(v6s) == 0 {
				return nil, fmt.Errorf("no AAAA records")
			}
			return v6s, nil
		}},
		{RecordMX, func() ([]string, error) {
			mxs, err := resolver.LookupMX(ctx, r.domain)
			if err != nil {
				return nil, err
			}
			var strs []string
			for _, mx := range mxs {
				strs = append(strs, fmt.Sprintf("%s (priority=%d)", mx.Host, mx.Pref))
			}
			return strs, nil
		}},
		{RecordTXT, func() ([]string, error) {
			txts, err := resolver.LookupTXT(ctx, r.domain)
			if err != nil {
				return nil, err
			}
			return txts, nil
		}},
		{RecordNS, func() ([]string, error) {
			nss, err := resolver.LookupNS(ctx, r.domain)
			if err != nil {
				return nil, err
			}
			var strs []string
			for _, ns := range nss {
				strs = append(strs, ns.Host)
			}
			return strs, nil
		}},
		{RecordCNAME, func() ([]string, error) {
			cname, err := resolver.LookupCNAME(ctx, r.domain)
			if err != nil {
				return nil, err
			}
			if cname == r.domain+"." {
				return nil, fmt.Errorf("no CNAME record")
			}
			return []string{cname}, nil
		}},
	}

	for _, lookup := range lookups {
		wg.Add(1)
		go func(typ RecordType, fn func() ([]string, error)) {
			defer wg.Done()
			vals, err := fn()
			if err != nil {
				logger.Debug("DNS %s lookup for %s: %v", typ, r.domain, err)
				return
			}
			mu.Lock()
			for _, v := range vals {
				result.Records = append(result.Records, Record{Type: typ, Value: v})
			}
			mu.Unlock()
		}(lookup.typ, lookup.fn)
	}

	wg.Wait()
	result.ResponseTime = time.Since(start)

	if len(result.Records) == 0 {
		result.Error = "no DNS records found"
	}

	config, _ := dnsReadConfig()
	if config != nil && len(config.Servers) > 0 {
		result.Nameserver = config.Servers[0]
	}

	return result
}

func dnsReadConfig() (*dnsConfig, error) {
	cfg, err := net.LookupAddr("127.0.0.1")
	if err != nil {
		return nil, err
	}
	_ = cfg
	config, _ := net.ResolveIPAddr("ip", "localhost")
	_ = config
	return &dnsConfig{Servers: []string{"system"}}, nil
}

type dnsConfig struct {
	Servers []string
}

func (r *Result) String() string {
	s := fmt.Sprintf("DNS Results for %s:\n", r.Domain)
	s += fmt.Sprintf("  Response Time: %v\n", r.ResponseTime)
	if r.Nameserver != "" {
		s += fmt.Sprintf("  Nameserver: %s\n", r.Nameserver)
	}
	if r.Error != "" {
		s += fmt.Sprintf("  Error: %s\n", r.Error)
	}
	for _, rec := range r.Records {
		s += fmt.Sprintf("  %-6s %s\n", rec.Type, rec.Value)
	}
	return s
}
