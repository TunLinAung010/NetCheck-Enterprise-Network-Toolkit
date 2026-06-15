# NetCheck - Enterprise Network Troubleshooting & Diagnostics Toolkit

A lightweight, cross-platform network diagnostics tool that combines **ping**, **tcping**, **traceroute**, **mtr**, **nslookup**, **port scanning**, and **service checking** into a single binary. Runs on Linux, macOS, and Windows.

## Quick Install

### Linux
```bash
# Option A: Build from source
git clone https://github.com/netcheck/netcheck.git
cd netcheck
go build -o netcheck ./cmd/netcheck
sudo mv netcheck /usr/local/bin/
sudo setcap cap_net_raw+ep /usr/local/bin/netcheck  # non-root ICMP ping

# Option B: Download release
curl -L https://github.com/netcheck/netcheck/releases/latest/download/netcheck-linux-amd64 -o netcheck
chmod +x netcheck && sudo mv netcheck /usr/local/bin/
```

### macOS
```bash
curl -L https://github.com/netcheck/netcheck/releases/latest/download/netcheck-darwin-amd64 -o netcheck
chmod +x netcheck && sudo mv netcheck /usr/local/bin/
```

### Windows (PowerShell)
```powershell
Invoke-WebRequest -Uri https://github.com/netcheck/netcheck/releases/latest/download/netcheck-windows-amd64.exe -OutFile netcheck.exe
# or build:
cd D:\Pilot-Projects\OpenCode\NetCheck-Enterprise-Network-Toolkit
$env:CGO_ENABLED="0"; go build -o netcheck.exe ./cmd/netcheck
```

### Docker
```bash
docker build -t netcheck .
docker run --rm --cap-add=NET_RAW --cap-add=NET_ADMIN netcheck google.com
```

## Usage

```bash
netcheck [host] [port] [flags]
```

| Command | What it does |
|---|---|
| `netcheck google.com` | ICMP ping |
| `netcheck google.com 443` | TCP port check (OPEN/CLOSED/TIMEOUT) |
| `netcheck google.com 443 --tcping` | **Continuous TCP ping** (like tcping.exe) |
| `netcheck 8.8.8.8 53 --udp` | UDP port check (DNS/NTP/SNMP/Syslog probes) |
| `netcheck 192.168.1.10 --ports 22,80,443` | Multi-port scan |
| `netcheck 192.168.1.0/24 --discover` | Host discovery (alive/down) |
| `netcheck --dns google.com` | DNS lookup (A/AAAA/MX/TXT/NS/CNAME) |
| `netcheck --traceroute google.com` | Traceroute |
| `netcheck --mtr google.com` | MTR (continuous route analysis) |
| `netcheck https://example.com` | HTTP health check |
| `netcheck --tls example.com` | TLS certificate check (expiry alert at 30 days) |
| `netcheck google.com 443 --watch --interval 5s` | Continuous monitoring |
| `netcheck google.com --json out.json --csv out.csv --html out.html` | Export results |
| `netcheck --web` | Launch web dashboard at http://localhost:8080 |
| `netcheck --metrics` | Prometheus metrics at http://localhost:9090/metrics |

### Using `--tcping` (TCP Ping)

Continuously probes a TCP port every 1 second until Ctrl+C, showing real-time status:

```bash
netcheck google.com 443 --tcping
```

Output:
```
TCPing google.com:443 - Ctrl+C to stop
--------------------------------------------------
seq=1 port=443 OPEN     latency=12.5ms
seq=2 port=443 OPEN     latency=11.3ms
seq=3 port=443 CLOSED   latency=5.0s
^C
--------------------------------------------------
--- google.com:443 tcping statistics ---
3 probes: 2 successful, 1 failed (33.3% loss)
rtt min/avg/max = 11.3ms/11.9ms/12.5ms
```

## Features

| Feature | Description |
|---|---|
| **ICMP Ping** | RTT, packet loss %, jitter statistics |
| **TCP Port Check** | OPEN/CLOSED/TIMEOUT with latency measurements |
| **UDP Port Check** | State detection with protocol-aware probes (DNS, NTP, SNMP, Syslog) |
| **Multi-Port Scan** | Concurrent TCP/UDP scanning with worker pools |
| **Host Discovery** | Scan CIDR ranges with ICMP + TCP probes |
| **DNS Tools** | A, AAAA, MX, TXT, NS, CNAME records |
| **Traceroute** | ICMP route tracing with per-hop latency |
| **MTR** | Continuous multi-hop analysis with real-time loss/latency |
| **HTTP Health Check** | Status code, response time, DNS/TCP/TLS timing, redirects |
| **TLS Certificate Check** | Subject, issuer, expiry date, 30-day warning |
| **Continuous Monitoring** | Watch mode with configurable interval |
| **Export** | JSON, CSV, HTML reports |
| **Alerting** | Telegram, Discord, Email notifications |
| **Prometheus Metrics** | availability, latency, packet_loss, cert_expiry |
| **Web Dashboard** | Built-in Go + HTMX + Bootstrap UI |

## Performance

- 10,000+ concurrent TCP checks
- 5,000+ concurrent UDP checks
- Goroutine-based worker pools with channels
- Context-aware cancellation

## Architecture

```
├── cmd/netcheck/       CLI entry point (Cobra)
├── internal/
│   ├── ping/           ICMP ping
│   ├── tcp/            TCP port check
│   ├── udp/            UDP port check
│   ├── dns/            DNS lookup
│   ├── traceroute/     Route tracing
│   ├── mtr/            MTR analysis
│   ├── httpcheck/      HTTP health check
│   ├── tlscheck/       TLS certificate check
│   ├── portscan/       Port scanning
│   ├── discover/       Host discovery
│   ├── alerts/         Notification system
│   ├── export/         Report generation
│   ├── monitoring/     Continuous monitoring
│   ├── metrics/        Prometheus integration
│   └── web/            Web dashboard
├── pkg/logger/         Structured logging
├── pkg/utils/          Utilities
└── web/                Web assets
```

## License

MIT
