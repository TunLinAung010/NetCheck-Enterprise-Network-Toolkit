# NetCheck User Manual

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Command Line Interface](#command-line-interface)
4. [ICMP Ping](#icmp-ping)
5. [TCP Port Check](#tcp-port-check)
6. [UDP Port Check](#udp-port-check)
7. [Multi-Port Scan](#multi-port-scan)
8. [Host Discovery](#host-discovery)
9. [DNS Tools](#dns-tools)
10. [Traceroute](#traceroute)
11. [MTR](#mtr)
12. [HTTP Health Check](#http-health-check)
13. [TLS Certificate Check](#tls-certificate-check)
14. [Continuous Monitoring](#continuous-monitoring)
15. [Export Formats](#export-formats)
16. [Alerting](#alerting)
17. [Web Dashboard](#web-dashboard)
18. [Configuration](#configuration)

## Introduction

NetCheck is an enterprise-grade network troubleshooting toolkit that combines the functionality of multiple network diagnostic tools into a single, portable binary. It is designed for Network Engineers, DevOps Engineers, System Administrators, and NOC Teams.

## Installation

### Linux
```bash
curl -L https://github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/releases/latest/download/netcheck-linux-amd64 -o netcheck
chmod +x netcheck
sudo mv netcheck /usr/local/bin/
```

### macOS
```bash
curl -L https://github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/releases/latest/download/netcheck-darwin-amd64 -o netcheck
chmod +x netcheck
sudo mv netcheck /usr/local/bin/
```

### Windows
```powershell
Invoke-WebRequest -Uri https://github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/releases/latest/download/netcheck-windows-amd64.exe -OutFile netcheck.exe
```

### Build from Source
```bash
git clone https://github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit.git
cd netcheck
go build -o netcheck ./cmd/netcheck
```

### Docker
```bash
docker build -t netcheck .
docker run --rm --cap-add=NET_RAW --cap-add=NET_ADMIN netcheck [command]
```

## Command Line Interface

```
netcheck [host] [port] [flags]

Global Flags:
  --config, -c string   Config file path
  --version, -v         Show version
  --udp                 UDP mode
  --ports string        Ports to scan (comma-separated)
  --discover            Host discovery mode
  --dns string          DNS lookup mode
  --traceroute string   Traceroute mode
  --mtr string          MTR mode
  --tls string          TLS certificate check
  --watch               Continuous monitoring mode
  --interval string     Monitoring interval (default: 5s)
  --json string         JSON output file
  --csv string          CSV output file
  --html string         HTML output file
  --metrics             Enable Prometheus metrics
  --web                 Enable web dashboard
  --web-port int        Web dashboard port (default: 8080)
```

## ICMP Ping

Basic host reachability test using ICMP echo requests.

```bash
netcheck google.com
```
Displays: Host, IP Address, Packets Sent/Received, Packet Loss %, Min/Avg/Max RTT, Jitter

## TCP Port Check

Check if a TCP port is open on a remote host.

```bash
netcheck google.com 443
```
Displays: OPEN, CLOSED, or TIMEOUT with connection latency statistics.

## UDP Port Check

Check UDP port state with protocol-aware probes.

```bash
netcheck 8.8.8.8 53 --udp
```
States: OPEN, CLOSED, FILTERED, OPEN_OR_FILTERED

Protocol-aware probes available for:
- DNS (port 53)
- NTP (port 123)
- SNMP (port 161)
- Syslog (port 514)

## Multi-Port Scan

Scan multiple TCP or UDP ports concurrently.

```bash
netcheck 192.168.1.10 --ports 22,80,443,3306
netcheck 192.168.1.10 --ports 53,123,161 --udp
```

## Host Discovery

Discover alive hosts on a network using ICMP and TCP probes.

```bash
netcheck 192.168.1.0/24 --discover
netcheck 192.168.1.0/24 --discover --discover-icmp --discover-tcp
```

## DNS Tools

Perform DNS lookups for multiple record types.

```bash
netcheck --dns google.com
```
Returns: A, AAAA, MX, TXT, NS, CNAME records with response time.

## Traceroute

Trace the network path to a host.

```bash
netcheck --traceroute google.com
```
Supports ICMP, TCP, and UDP methods (ICMP default).

## MTR

Continuous route analysis combining traceroute and ping.

```bash
netcheck --mtr google.com
```
Displays real-time loss %, latency, jitter per hop.

## HTTP Health Check

Check HTTP/HTTPS endpoint health.

```bash
netcheck https://example.com
netcheck http://example.com
```
Returns: Status code, response time, headers, redirect count, DNS/TCP/TLS timings.

## TLS Certificate Check

Check TLS certificate validity and expiry.

```bash
netcheck --tls example.com
```
Displays: Subject, Issuer, validity period, days remaining, TLS version.
Warns if certificate expires within 30 days.

## Continuous Monitoring

Monitor host or port availability over time.

```bash
netcheck google.com 443 --watch --interval 5s
netcheck google.com --watch --interval 1s
```

## Export Formats

Export results in JSON, CSV, or HTML format.

```bash
netcheck google.com --json result.json --csv result.csv --html report.html
```

## Alerting

Configure notifications via Telegram, Discord, or Email.

Using config file:
```yaml
telegram_token: "YOUR_BOT_TOKEN"
telegram_chat: "YOUR_CHAT_ID"
discord_webhook: "YOUR_WEBHOOK_URL"
smtp_server: "smtp.gmail.com"
smtp_port: 587
smtp_user: "user@gmail.com"
smtp_pass: "password"
email_from: "user@gmail.com"
email_to: ["admin@example.com"]
alert_on_down: true
alert_on_close: true
alert_on_latency: 200ms
alert_on_cert: true
```

## Web Dashboard

Launch the built-in web UI.

```bash
netcheck --web --web-port 8080
```
Access at http://localhost:8080

Features: Live dashboard, Ping, TCP, Traceroute, DNS, TLS checks, Host discovery.

## Configuration

NetCheck can be configured via YAML file or environment variables. Configuration locations (in order of precedence):
1. `--config` flag
2. `./netcheck.yaml`
3. `$HOME/.netcheck/netcheck.yaml`
4. `/etc/netcheck/netcheck.yaml`

Environment variables use the `NETCHECK_` prefix (e.g., `NETCHECK_LOG_LEVEL=debug`).
