package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/netcheck/netcheck/internal/httpcheck"
	"github.com/netcheck/netcheck/internal/ping"
	"github.com/netcheck/netcheck/internal/tcp"
	"github.com/netcheck/netcheck/internal/tlscheck"
	"github.com/netcheck/netcheck/internal/traceroute"
	"github.com/netcheck/netcheck/pkg/logger"
)

type Dashboard struct {
	Port    int
	mu      sync.RWMutex
	results []CheckResult
}

type CheckResult struct {
	Target     string
	Type       string
	Status     string
	Latency    string
	Timestamp  time.Time
	Details    string
}

type JobRequest struct {
	Target string `json:"target"`
	Port   int    `json:"port,omitempty"`
	Type   string `json:"type"`
}

func New(port int) *Dashboard {
	return &Dashboard{
		Port: port,
	}
}

func (d *Dashboard) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", d.handleIndex)
	mux.HandleFunc("/api/check", d.handleCheck)
	mux.HandleFunc("/api/history", d.handleHistory)
	mux.HandleFunc("/ping", d.handlePingPage)
	mux.HandleFunc("/tcp", d.handleTCPPage)
	mux.HandleFunc("/traceroute", d.handleTraceroutePage)
	mux.HandleFunc("/dns", d.handleDNSPage)
	mux.HandleFunc("/tls", d.handleTLSPage)
	mux.HandleFunc("/discover", d.handleDiscoverPage)

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	addr := fmt.Sprintf(":%d", d.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	logger.Info("web dashboard listening on http://localhost%s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("web server error: %w", err)
	}
	return nil
}

func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>NetCheck Dashboard</title>
<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
<script src="https://unpkg.com/htmx.org@1.9.10"></script>
<style>
body { background: #f8f9fa; }
.card { margin-bottom: 1rem; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
.status-open { color: #198754; font-weight: bold; }
.status-closed { color: #dc3545; }
.status-filtered { color: #ffc107; }
.navbar-brand { font-weight: bold; letter-spacing: 1px; }
.check-result { font-family: 'Courier New', monospace; font-size: 0.9rem; }
.loading { opacity: 0.7; }
</style>
</head>
<body>
<nav class="navbar navbar-expand-lg navbar-dark bg-dark">
  <div class="container">
    <a class="navbar-brand" href="/">NetCheck</a>
    <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav">
      <span class="navbar-toggler-icon"></span>
    </button>
    <div class="collapse navbar-collapse" id="navbarNav">
      <ul class="navbar-nav">
        <li class="nav-item"><a class="nav-link" href="/">Dashboard</a></li>
        <li class="nav-item"><a class="nav-link" href="/ping">Ping</a></li>
        <li class="nav-item"><a class="nav-link" href="/tcp">TCP Check</a></li>
        <li class="nav-item"><a class="nav-link" href="/traceroute">Traceroute</a></li>
        <li class="nav-item"><a class="nav-link" href="/dns">DNS</a></li>
        <li class="nav-item"><a class="nav-link" href="/tls">TLS</a></li>
        <li class="nav-item"><a class="nav-link" href="/discover">Discover</a></li>
      </ul>
    </div>
  </div>
</nav>

<div class="container mt-4">
  <div class="row">
    <div class="col-md-8">
      <div class="card">
        <div class="card-header"><h5>Quick Check</h5></div>
        <div class="card-body">
          <form hx-post="/api/check" hx-target="#result" hx-indicator="#spinner">
            <div class="row g-3">
              <div class="col-md-4">
                <input type="text" class="form-control" name="target" placeholder="Hostname or IP" required>
              </div>
              <div class="col-md-2">
                <input type="number" class="form-control" name="port" placeholder="Port">
              </div>
              <div class="col-md-3">
                <select class="form-select" name="type">
                  <option value="ping">Ping</option>
                  <option value="tcp">TCP</option>
                  <option value="http">HTTP</option>
                  <option value="tls">TLS</option>
                  <option value="traceroute">Traceroute</option>
                </select>
              </div>
              <div class="col-md-3">
                <button type="submit" class="btn btn-primary w-100">Run Check</button>
              </div>
            </div>
          </form>
          <div id="spinner" class="htmx-indicator mt-2">
            <div class="spinner-border text-primary" role="status">
              <span class="visually-hidden">Running...</span>
            </div>
          </div>
        </div>
      </div>
      <div id="result"></div>
    </div>
    <div class="col-md-4">
      <div class="card">
        <div class="card-header"><h5>Recent Checks</h5></div>
        <div class="card-body" id="history" hx-get="/api/history" hx-trigger="every:10s">
          <p class="text-muted">No checks yet</p>
        </div>
      </div>
    </div>
  </div>
</div>

<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (d *Dashboard) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	target := r.FormValue("target")
	portStr := r.FormValue("port")
	checkType := r.FormValue("type")

	port := 0
	if portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var result CheckResult
	result.Target = target
	result.Type = checkType
	result.Timestamp = time.Now()

	switch checkType {
	case "ping":
		p := ping.New(target, 4, time.Second, time.Second*5)
		res, err := p.Run(ctx)
		if err != nil {
			result.Status = "ERROR"
			result.Details = err.Error()
		} else {
			result.Status = "OK"
			result.Latency = res.Avg.String()
			result.Details = fmt.Sprintf("Sent=%d, Received=%d, Loss=%.0f%%, Min=%v, Max=%v, Jitter=%v",
				res.Sent, res.Received, res.PacketLoss, res.Min, res.Max, res.Jitter)
			if res.PacketLoss > 0 {
				result.Status = "PARTIAL"
			}
		}

	case "tcp":
		if port == 0 {
			port = 443
		}
		c := tcp.New(target, port, time.Second*5, 1)
		res := c.Run(ctx)
		result.Status = string(res.State)
		result.Latency = res.Latency.String()
		result.Details = fmt.Sprintf("Port %d: %s", port, res.State)

	case "http":
		url := target
		if !strings.HasPrefix(url, "http") {
			url = "https://" + url
		}
		c := httpcheck.New(url, time.Second*10, true)
		res := c.Run(ctx)
		result.Status = fmt.Sprintf("%d", res.StatusCode)
		result.Latency = res.ResponseTime.String()
		result.Details = fmt.Sprintf("Status: %d %s, DNS: %v, TCP: %v, Redirects: %d",
			res.StatusCode, res.StatusText, res.DNSLookup, res.TCPConnect, res.RedirectCount)

	case "tls":
		c := tlscheck.New(target, 443, time.Second*10)
		res := c.Run(ctx)
		if res.Error != "" {
			result.Status = "ERROR"
			result.Details = res.Error
		} else {
			result.Status = "OK"
			result.Latency = ""
			result.Details = fmt.Sprintf("Subject: %s, Issuer: %s, Expires: %s (%d days)",
				res.Subject, res.Issuer, res.NotAfter.Format(time.RFC3339), res.DaysRemaining)
			if res.ExpiringSoon {
				result.Status = "WARNING"
			}
		}

	case "traceroute":
		t := traceroute.New(target, traceroute.MethodICMP, 30, time.Second*3, 0)
		res, err := t.Run(ctx)
		if err != nil {
			result.Status = "ERROR"
			result.Details = err.Error()
		} else {
			result.Status = "OK"
			hops := make([]string, len(res.Hops))
			for i, hop := range res.Hops {
				hops[i] = fmt.Sprintf("%d: %s (%v)", hop.Number, hop.IP, hop.Latency)
			}
			result.Details = strings.Join(hops, "\n")
		}

	default:
		result.Status = "ERROR"
		result.Details = fmt.Sprintf("unknown check type: %s", checkType)
	}

	d.mu.Lock()
	d.results = append(d.results, result)
	if len(d.results) > 100 {
		d.results = d.results[len(d.results)-100:]
	}
	d.mu.Unlock()

	if r.Header.Get("HX-Request") == "true" {
		html := fmt.Sprintf(`<div class="card mt-3">
<div class="card-header d-flex justify-content-between">
<span>%s %s</span>
<span class="status-%s">%s</span>
</div>
<div class="card-body check-result"><pre>%s</pre></div>
<div class="card-footer text-muted small">%s | Latency: %s</div>
</div>`,
			result.Type, result.Target,
			strings.ToLower(result.Status), result.Status,
			result.Details,
			result.Timestamp.Format(time.RFC3339), result.Latency)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func (d *Dashboard) handleHistory(w http.ResponseWriter, r *http.Request) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.results) == 0 {
		w.Write([]byte(`<p class="text-muted">No checks yet</p>`))
		return
	}

	html := `<table class="table table-sm"><thead><tr><th>Time</th><th>Target</th><th>Type</th><th>Status</th><th>Latency</th></tr></thead><tbody>`
	for i := len(d.results) - 1; i >= 0 && i >= len(d.results)-10; i-- {
		r := d.results[i]
		html += fmt.Sprintf(`<tr>
<td class="small">%s</td>
<td>%s</td>
<td>%s</td>
<td class="status-%s">%s</td>
<td>%s</td>
</tr>`,
			r.Timestamp.Format("15:04:05"), r.Target, r.Type,
			strings.ToLower(r.Status), r.Status, r.Latency)
	}
	html += `</tbody></table>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (d *Dashboard) handlePingPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(pageWrapper("Ping", `<div class="card">
<div class="card-header"><h5>ICMP Ping</h5></div>
<div class="card-body">
<form hx-post="/api/check" hx-target="#result" hx-indicator="#spinner">
<input type="hidden" name="type" value="ping">
<div class="row g-3">
<div class="col-md-6"><input type="text" class="form-control" name="target" placeholder="Hostname or IP" required></div>
<div class="col-md-3"><button type="submit" class="btn btn-primary">Ping</button></div>
</div>
</form>
<div id="spinner" class="htmx-indicator mt-2"><div class="spinner-border text-primary" role="status"><span class="visually-hidden">Running...</span></div></div>
</div></div>
<div id="result"></div>`)))
}

func (d *Dashboard) handleTCPPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(pageWrapper("TCP Check", `<div class="card">
<div class="card-header"><h5>TCP Port Check</h5></div>
<div class="card-body">
<form hx-post="/api/check" hx-target="#result" hx-indicator="#spinner">
<input type="hidden" name="type" value="tcp">
<div class="row g-3">
<div class="col-md-4"><input type="text" class="form-control" name="target" placeholder="Hostname or IP" required></div>
<div class="col-md-2"><input type="number" class="form-control" name="port" placeholder="Port" value="443"></div>
<div class="col-md-3"><button type="submit" class="btn btn-primary">Check Port</button></div>
</div>
</form>
<div id="spinner" class="htmx-indicator mt-2"><div class="spinner-border text-primary" role="status"><span class="visually-hidden">Running...</span></div></div>
</div></div>
<div id="result"></div>`)))
}

func (d *Dashboard) handleTraceroutePage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(pageWrapper("Traceroute", `<div class="card">
<div class="card-header"><h5>Traceroute</h5></div>
<div class="card-body">
<form hx-post="/api/check" hx-target="#result" hx-indicator="#spinner">
<input type="hidden" name="type" value="traceroute">
<div class="row g-3">
<div class="col-md-6"><input type="text" class="form-control" name="target" placeholder="Hostname or IP" required></div>
<div class="col-md-3"><button type="submit" class="btn btn-primary">Trace</button></div>
</div>
</form>
<div id="spinner" class="htmx-indicator mt-2"><div class="spinner-border text-primary" role="status"><span class="visually-hidden">Running...</span></div></div>
</div></div>
<div id="result"></div>`)))
}

func (d *Dashboard) handleDNSPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(pageWrapper("DNS Lookup", `<div class="card">
<div class="card-header"><h5>DNS Lookup</h5></div>
<div class="card-body">
<form hx-post="/api/check" hx-target="#result" hx-indicator="#spinner">
<input type="hidden" name="type" value="dns">
<div class="row g-3">
<div class="col-md-6"><input type="text" class="form-control" name="target" placeholder="Domain name" required></div>
<div class="col-md-3"><button type="submit" class="btn btn-primary">Lookup</button></div>
</div>
</form>
<div id="spinner" class="htmx-indicator mt-2"><div class="spinner-border text-primary" role="status"><span class="visually-hidden">Running...</span></div></div>
</div></div>
<div id="result"></div>`)))
}

func (d *Dashboard) handleTLSPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(pageWrapper("TLS Certificate", `<div class="card">
<div class="card-header"><h5>TLS Certificate Check</h5></div>
<div class="card-body">
<form hx-post="/api/check" hx-target="#result" hx-indicator="#spinner">
<input type="hidden" name="type" value="tls">
<div class="row g-3">
<div class="col-md-6"><input type="text" class="form-control" name="target" placeholder="Hostname" required></div>
<div class="col-md-3"><button type="submit" class="btn btn-primary">Check TLS</button></div>
</div>
</form>
<div id="spinner" class="htmx-indicator mt-2"><div class="spinner-border text-primary" role="status"><span class="visually-hidden">Running...</span></div></div>
</div></div>
<div id="result"></div>`)))
}

func (d *Dashboard) handleDiscoverPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(pageWrapper("Host Discovery", `<div class="card">
<div class="card-header"><h5>Network Discovery</h5></div>
<div class="card-body">
<form hx-post="/api/discover" hx-target="#result" hx-indicator="#spinner">
<div class="row g-3">
<div class="col-md-6"><input type="text" class="form-control" name="target" placeholder="CIDR (e.g. 192.168.1.0/24)" required></div>
<div class="col-md-3"><button type="submit" class="btn btn-primary">Discover</button></div>
</div>
</form>
<div id="spinner" class="htmx-indicator mt-2"><div class="spinner-border text-primary" role="status"><span class="visually-hidden">Running...</span></div></div>
</div></div>
<div id="result"></div>`)))
}

func pageWrapper(title, content string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s - NetCheck</title>
<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
<script src="https://unpkg.com/htmx.org@1.9.10"></script>
<style>
body { background: #f8f9fa; }
.card { margin-bottom: 1rem; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
.status-ok { color: #198754; font-weight: bold; }
.status-open { color: #198754; font-weight: bold; }
.status-closed { color: #dc3545; }
.status-filtered { color: #ffc107; }
.status-error { color: #dc3545; }
.status-warning { color: #ffc107; }
.status-partial { color: #ffc107; }
.navbar-brand { font-weight: bold; letter-spacing: 1px; }
.check-result { font-family: 'Courier New', monospace; font-size: 0.9rem; }
pre { background: #f4f4f4; padding: 1rem; border-radius: 4px; overflow-x: auto; }
.htmx-indicator { display: none; }
.htmx-request .htmx-indicator { display: inline-block; }
</style>
</head>
<body>
<nav class="navbar navbar-expand-lg navbar-dark bg-dark">
<div class="container">
<a class="navbar-brand" href="/">NetCheck</a>
<button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav">
<span class="navbar-toggler-icon"></span>
</button>
<div class="collapse navbar-collapse" id="navbarNav">
<ul class="navbar-nav">
<li class="nav-item"><a class="nav-link" href="/">Dashboard</a></li>
<li class="nav-item"><a class="nav-link" href="/ping">Ping</a></li>
<li class="nav-item"><a class="nav-link" href="/tcp">TCP Check</a></li>
<li class="nav-item"><a class="nav-link" href="/traceroute">Traceroute</a></li>
<li class="nav-item"><a class="nav-link" href="/dns">DNS</a></li>
<li class="nav-item"><a class="nav-link" href="/tls">TLS</a></li>
<li class="nav-item"><a class="nav-link" href="/discover">Discover</a></li>
</ul>
</div>
</div>
</nav>
<div class="container mt-4">%s</div>
<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>`, title, content)
}

func findAvailablePort(start int) int {
	for port := start; port < start+100; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return start
}
