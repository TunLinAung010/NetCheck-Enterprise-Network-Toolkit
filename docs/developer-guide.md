# NetCheck Developer Guide

## Architecture

NetCheck follows Clean Architecture principles with clear separation of concerns:

```
cmd/        - Application entry points
internal/   - Core business logic
pkg/        - Shared utilities and libraries
web/        - Web interface assets
docs/       - Documentation
tests/      - Test suites
```

### Module Structure

Each network protocol or feature is encapsulated in its own package under `internal/`:

| Package | Responsibility |
|---------|---------------|
| `ping` | ICMP echo request/reply handling |
| `tcp` | TCP connection state detection |
| `udp` | UDP probe with protocol-aware payloads |
| `dns` | DNS record lookup (A, AAAA, MX, TXT, NS, CNAME) |
| `traceroute` | Route path discovery via TTL manipulation |
| `mtr` | Continuous multi-hop analysis |
| `httpcheck` | HTTP/S endpoint health assessment |
| `tlscheck` | TLS certificate validation and expiry monitoring |
| `portscan` | Concurrent port scanning with worker pools |
| `discover` | Network host discovery (ICMP/TCP) |
| `alerts` | Multi-channel notification system |
| `export` | Report generation (JSON/CSV/HTML) |
| `monitoring` | Continuous data collection and aggregation |
| `metrics` | Prometheus metrics exposition |
| `web` | Embedded web dashboard |
| `config` | Configuration management |

## Development Setup

### Prerequisites

- Go 1.22 or later
- Git

### Getting Started

```bash
git clone https://github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit.git
cd netcheck
go mod download
go build -o netcheck ./cmd/netcheck
```

### Running Tests

```bash
# All tests
go test -v -race -count=1 ./...

# Specific package
go test -v -race ./internal/ping/...

# With coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`, `golint`)
- Use meaningful names with clear intent
- Package-level documentation for all public types and functions
- Error wrapping with context using `fmt.Errorf("context: %w", err)`

## Adding a New Feature

1. Create a new package under `internal/`
2. Define types and interfaces
3. Implement with context support and worker pools where applicable
4. Add export support via the `Exportable` interface
5. Register the command in `cmd/netcheck/main.go`
6. Add tests with at least 85% coverage
7. Update documentation

## Build Process

### Cross-Platform Builds

```bash
# Using build script
./scripts/build.sh

# Manual cross-compilation
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o netcheck-linux-amd64 ./cmd/netcheck
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o netcheck-darwin-amd64 ./cmd/netcheck
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o netcheck-windows-amd64.exe ./cmd/netcheck
```

### Release Build

```bash
VERSION=1.0.0 ./scripts/build.sh
```

## Docker

```bash
docker build -t netcheck .
docker run --rm --cap-add=NET_RAW --cap-add=NET_ADMIN netcheck google.com
```

## CI/CD Pipeline

The project uses GitHub Actions for continuous integration and delivery:

- `lint` - Code quality checks via golangci-lint
- `test` - Cross-platform test execution (Linux, Windows, macOS)
- `build` - Multi-architecture binary compilation
- `release` - Automated release artifact generation

## Performance Guidelines

- Use goroutine worker pools for concurrent operations
- Implement context cancellation for graceful shutdown
- Channel-based result aggregation to avoid shared state
- Configurable worker count for different deployment scenarios
- Support 10,000+ TCP and 5,000+ UDP concurrent checks

## Export System

Implement the `Exportable` interface for any check result:

```go
type Exportable interface {
    ToMap() map[string]interface{}
    Headers() []string
    Row() []string
}
```

## Alerting System

The alerting module supports multiple notification channels:

- **Telegram**: Bot API integration
- **Discord**: Webhook integration with rich embeds
- **Email**: SMTP with TLS support

Add a new channel by implementing a sender method on the `Notifier` struct.
