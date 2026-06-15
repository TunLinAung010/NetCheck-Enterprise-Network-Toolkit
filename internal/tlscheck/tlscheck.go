package tlscheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

type Result struct {
	Host          string
	Subject       string
	Issuer        string
	NotBefore     time.Time
	NotAfter      time.Time
	DaysRemaining int
	Expired       bool
	ExpiringSoon  bool
	DNSNames      []string
	TLSVersion    string
	CipherSuite   string
	Error         string
}

type Checker struct {
	host    string
	port    int
	timeout time.Duration
}

func New(host string, port int, timeout time.Duration) *Checker {
	return &Checker{
		host:    host,
		port:    port,
		timeout: timeout,
	}
}

func (c *Checker) Run(ctx context.Context) *Result {
	result := &Result{Host: c.host}

	addr := net.JoinHostPort(c.host, fmt.Sprintf("%d", c.port))

	dialer := &net.Dialer{Timeout: c.timeout}
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		result.Error = fmt.Sprintf("TCP connection failed: %v", err)
		return result
	}

	tlsConn := tls.Client(rawConn, &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         c.host,
	})

	if err := tlsConn.Handshake(); err != nil {
		rawConn.Close()
		result.Error = fmt.Sprintf("TLS handshake failed: %v", err)
		return result
	}
	defer tlsConn.Close()

	state := tlsConn.ConnectionState()

	if len(state.PeerCertificates) == 0 {
		result.Error = "no peer certificates"
		return result
	}

	cert := state.PeerCertificates[0]
	result.Subject = cert.Subject.CommonName
	if len(cert.Subject.Organization) > 0 {
		result.Subject = cert.Subject.Organization[0] + " - " + cert.Subject.CommonName
	}
	result.Issuer = cert.Issuer.CommonName
	if len(cert.Issuer.Organization) > 0 {
		result.Issuer = cert.Issuer.Organization[0]
	}
	result.NotBefore = cert.NotBefore
	result.NotAfter = cert.NotAfter
	result.DNSNames = cert.DNSNames

	daysUntilExpiry := time.Until(cert.NotAfter).Hours() / 24
	result.DaysRemaining = int(daysUntilExpiry)

	if result.DaysRemaining < 0 {
		result.Expired = true
	}
	if result.DaysRemaining >= 0 && result.DaysRemaining <= 30 {
		result.ExpiringSoon = true
	}

	result.TLSVersion = tlsVersionString(state.Version)
	if state.CipherSuite > 0 {
		result.CipherSuite = tls.CipherSuiteName(state.CipherSuite)
	}

	return result
}

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("0x%04X", version)
	}
}

func (r *Result) String() string {
	s := fmt.Sprintf("TLS Certificate for %s:\n", r.Host)
	if r.Error != "" {
		s += fmt.Sprintf("  Error: %s\n", r.Error)
		return s
	}
	s += fmt.Sprintf("  Subject: %s\n", r.Subject)
	s += fmt.Sprintf("  Issuer: %s\n", r.Issuer)
	s += fmt.Sprintf("  Valid From: %s\n", r.NotBefore.Format(time.RFC3339))
	s += fmt.Sprintf("  Valid Until: %s\n", r.NotAfter.Format(time.RFC3339))
	s += fmt.Sprintf("  Days Remaining: %d\n", r.DaysRemaining)
	if r.Expired {
		s += fmt.Sprintf("  EXPIRED\n")
	} else if r.ExpiringSoon {
		s += fmt.Sprintf("  Expiring within 30 days\n")
	}
	s += fmt.Sprintf("  TLS Version: %s\n", r.TLSVersion)
	if r.CipherSuite != "" {
		s += fmt.Sprintf("  Cipher Suite: %s\n", r.CipherSuite)
	}
	if len(r.DNSNames) > 0 {
		s += fmt.Sprintf("  DNS Names: %v\n", r.DNSNames)
	}
	return s
}
