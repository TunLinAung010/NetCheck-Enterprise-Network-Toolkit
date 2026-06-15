package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/netcheck/netcheck/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry = prometheus.NewRegistry()

	availability = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "netcheck_availability",
			Help: "Target availability (1=up, 0=down)",
		},
		[]string{"target", "type"},
	)

	latency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "netcheck_latency_ms",
			Help: "Latency in milliseconds",
		},
		[]string{"target", "type"},
	)

	packetLoss = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "netcheck_packet_loss_percent",
			Help: "Packet loss percentage",
		},
		[]string{"target"},
	)

	certExpiry = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "netcheck_certificate_days_remaining",
			Help: "Days remaining until TLS certificate expiry",
		},
		[]string{"target"},
	)

	checkDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "netcheck_check_duration_seconds",
			Help:    "Duration of network checks",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	checkTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "netcheck_checks_total",
			Help: "Total number of checks performed",
		},
		[]string{"type", "status"},
	)

	mu sync.RWMutex
)

func init() {
	registry.MustRegister(availability)
	registry.MustRegister(latency)
	registry.MustRegister(packetLoss)
	registry.MustRegister(certExpiry)
	registry.MustRegister(checkDuration)
	registry.MustRegister(checkTotal)
}

func SetAvailability(target, checkType string, up bool) {
	val := 0.0
	if up {
		val = 1.0
	}
	availability.WithLabelValues(target, checkType).Set(val)
}

func SetLatency(target, checkType string, d time.Duration) {
	latency.WithLabelValues(target, checkType).Set(float64(d.Milliseconds()))
}

func SetPacketLoss(target string, loss float64) {
	packetLoss.WithLabelValues(target).Set(loss)
}

func SetCertExpiry(target string, days float64) {
	certExpiry.WithLabelValues(target).Set(days)
}

func ObserveDuration(checkType string, d time.Duration) {
	checkDuration.WithLabelValues(checkType).Observe(d.Seconds())
}

func IncCheck(checkType, status string) {
	checkTotal.WithLabelValues(checkType, status).Inc()
}

func ServeHTTP(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","version":"1.0.0"}`))
	})

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

	logger.Info("metrics server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("metrics server error: %w", err)
	}
	return nil
}
