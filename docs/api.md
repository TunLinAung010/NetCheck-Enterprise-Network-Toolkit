# NetCheck API Documentation

## Command Line API

### Network Checks

#### ICMP Ping
```
netcheck <host>
netcheck <host> --json <file> --csv <file> --html <file>
```
Response: Host, IP, Sent/Received, Packet Loss %, Min/Avg/Max RTT, Jitter

#### TCP Port Check
```
netcheck <host> <port>
```
Response: OPEN | CLOSED | TIMEOUT, latency stats

#### UDP Port Check
```
netcheck <host> <port> --udp
```
Response: OPEN | CLOSED | FILTERED | OPEN_OR_FILTERED

#### Multi-Port Scan
```
netcheck <host> --ports <port1>,<port2>,...
netcheck <host> --ports <port1>,<port2>,... --udp
```
Response: Per-port state, protocol, service name, latency

#### Host Discovery
```
netcheck <cidr> --discover
```
Response: Per-host IP, state (Alive/Down/Unknown), discovery method, hostname

### DNS Tools

```
netcheck --dns <domain>
```
Response: A, AAAA, MX, TXT, NS, CNAME records with response time

### Traceroute

```
netcheck --traceroute <host>
```
Response: Per-hop number, IP, hostname, latency

### MTR

```
netcheck --mtr <host>
```
Response: Per-hop loss %, last/avg/best/worst latency, jitter

### HTTP Health Check

```
netcheck https://<url>
netcheck http://<url>
```
Response: Status code, response time, dns/tcp/tls timings, headers, redirect count

### TLS Certificate Check

```
netcheck --tls <host>
```
Response: Subject, issuer, validity period, days remaining, TLS version

### Continuous Monitoring

```
netcheck <host> [port] --watch --interval <duration>
```

### Export Formats

```
--json <file>    JSON array of results
--csv <file>     CSV with headers and data rows
--html <file>    HTML report with styled table
```

## Prometheus Metrics API

Available at `/metrics` endpoint.

### Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `netcheck_availability` | Gauge | `target`, `type` | 1 if up, 0 if down |
| `netcheck_latency_ms` | Gauge | `target`, `type` | Latency in milliseconds |
| `netcheck_packet_loss_percent` | Gauge | `target` | Packet loss percentage |
| `netcheck_certificate_days_remaining` | Gauge | `target` | Days until certificate expiry |
| `netcheck_check_duration_seconds` | Histogram | `type` | Duration of network checks |
| `netcheck_checks_total` | Counter | `type`, `status` | Total checks performed |

### Health Check

```
GET /health
```
Response: `{"status":"ok","version":"1.0.0"}`

## Web Dashboard API

### Run Check
```
POST /api/check
Content-Type: application/x-www-form-urlencoded

target=<host>&port=<port>&type=<ping|tcp|http|tls|traceroute>
```
Response: HTML fragment (HTMX) or JSON

### Check History
```
GET /api/history
```
Response: HTML table of recent check results (HTMX)

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | Network error |
| 4 | Timeout |
| 5 | Configuration error |

## Configuration File

Default locations:
- `./netcheck.yaml`
- `$HOME/.netcheck/netcheck.yaml`
- `/etc/netcheck/netcheck.yaml`

### Schema Reference

```yaml
log_level: string           # debug, info, warn, error
log_json: boolean           # Enable JSON logging
ping_count: int             # Number of ping requests (default: 4)
ping_interval: duration     # Between pings (default: 1s)
ping_timeout: duration      # Per-ping timeout (default: 5s)
tcp_timeout: duration       # TCP dial timeout (default: 5s)
udp_timeout: duration       # UDP probe timeout (default: 3s)
scan_workers: int           # Concurrent scan workers (default: 100)
watch_interval: duration    # Monitoring interval (default: 5s)
prometheus_port: int        # Metrics port (default: 9090)
web_port: int               # Dashboard port (default: 8080)
telegram_token: string      # Telegram bot token
telegram_chat: string       # Telegram chat ID
discord_webhook: string     # Discord webhook URL
smtp_server: string         # SMTP server host
smtp_port: int              # SMTP server port
smtp_user: string           # SMTP username
smtp_pass: string           # SMTP password
email_from: string          # Sender email address
email_to: [string]          # Recipient email addresses
alert_on_down: boolean      # Alert on host down
alert_on_close: boolean     # Alert on port closed
alert_on_latency: duration  # Alert threshold latency
alert_on_cert: boolean      # Alert on cert expiry
```
