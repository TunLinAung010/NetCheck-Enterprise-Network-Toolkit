package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/config"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/discover"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/dns"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/export"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/httpcheck"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/metrics"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/monitoring"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/mtr"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/ping"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/portscan"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/tcp"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/tlscheck"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/traceroute"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/udp"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/internal/web"
	"github.com/TunLinAung010/NetCheck-Enterprise-Network-Toolkit/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	cfgFile       string
	showVersion   bool
	udpMode       bool
	portsFlag     string
	discoverFlag  bool
	dnsFlag       string
	tracerouteFlag string
	mtrFlag       string
	tlsFlag       string
	watchFlag     bool
	intervalFlag  string
	jsonOutput    string
	csvOutput     string
	htmlOutput    string
	metricsFlag   bool
	webFlag       bool
	webPort       int
	tcpingFlag    bool
	discoverIcmp  bool
	discoverTcp   bool
	discoverArp   bool
	version       = "1.0.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "netcheck [host] [port] [flags]",
		Short: "Enterprise Network Troubleshooting & Diagnostics Toolkit",
		Long:  `NetCheck - Enterprise Network Troubleshooting & Diagnostics Toolkit

Examples:
  netcheck google.com                    ICMP Ping
  netcheck google.com 443                TCP Port Check
  netcheck example.com 443 --tcping      TCP Ping (continuous)
  netcheck 8.8.8.8 53 --udp              UDP Port Check
  netcheck 192.168.1.10 --ports 22,80    Multi-Port Scan
  netcheck 192.168.1.0/24 --discover     Host Discovery
  netcheck --dns google.com              DNS Lookup
  netcheck --traceroute google.com       Traceroute
  netcheck --mtr google.com              MTR Mode`,
		RunE:  runRoot,
	}

	flags := rootCmd.Flags()
	flags.StringVarP(&cfgFile, "config", "c", "", "config file path")
	flags.BoolVarP(&showVersion, "version", "v", false, "show version")
	flags.BoolVar(&udpMode, "udp", false, "UDP mode")
	flags.StringVar(&portsFlag, "ports", "", "ports to scan (comma-separated)")
	flags.BoolVar(&discoverFlag, "discover", false, "host discovery mode")
	flags.StringVar(&dnsFlag, "dns", "", "DNS lookup mode")
	flags.StringVar(&tracerouteFlag, "traceroute", "", "traceroute mode")
	flags.StringVar(&mtrFlag, "mtr", "", "MTR mode")
	flags.StringVar(&tlsFlag, "tls", "", "TLS certificate check")
	flags.BoolVar(&watchFlag, "watch", false, "continuous monitoring mode")
	flags.StringVar(&intervalFlag, "interval", "5s", "monitoring interval")
	flags.StringVar(&jsonOutput, "json", "", "JSON output file")
	flags.StringVar(&csvOutput, "csv", "", "CSV output file")
	flags.StringVar(&htmlOutput, "html", "", "HTML output file")
	flags.BoolVar(&metricsFlag, "metrics", false, "enable Prometheus metrics")
	flags.BoolVar(&webFlag, "web", false, "enable web dashboard")
	flags.IntVar(&webPort, "web-port", 8080, "web dashboard port")
	flags.BoolVar(&tcpingFlag, "tcping", false, "TCP ping mode (continuous port check)")
	flags.BoolVar(&discoverIcmp, "discover-icmp", true, "ICMP discovery")
	flags.BoolVar(&discoverTcp, "discover-tcp", true, "TCP discovery")
	flags.BoolVar(&discoverArp, "discover-arp", false, "ARP discovery")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	if showVersion {
		fmt.Printf("NetCheck v%s\n", version)
		return nil
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		logger.Warn("failed to load config: %v", err)
	}
	if cfg == nil {
		cfg = config.Default()
	}

	logger.SetLevel(logger.ParseLevel(cfg.LogLevel))
	logger.SetJSONMode(cfg.LogJSON)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	if metricsFlag {
		go func() {
			addr := fmt.Sprintf(":%d", cfg.PrometheusPort)
			if err := metrics.ServeHTTP(ctx, addr); err != nil {
				logger.Error("metrics server: %v", err)
			}
		}()
	}

	if webFlag {
		dashboard := web.New(webPort)
		go func() {
			if err := dashboard.Run(ctx); err != nil {
				logger.Error("web dashboard: %v", err)
			}
		}()
	}

	if dnsFlag != "" {
		return runDNS(ctx, dnsFlag)
	}

	if tracerouteFlag != "" {
		return runTraceroute(ctx, tracerouteFlag)
	}

	if mtrFlag != "" {
		return runMTR(ctx, mtrFlag)
	}

	if tlsFlag != "" {
		return runTLSCheck(ctx, tlsFlag)
	}

	if (metricsFlag || webFlag) && len(args) == 0 {
		<-ctx.Done()
		return nil
	}

	if len(args) == 0 {
		return cmd.Help()
	}

	host := args[0]

	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return runHTTPCheck(ctx, host)
	}

	if discoverFlag {
		return runDiscover(ctx, host, cfg)
	}

	if portsFlag != "" {
		return runPortScan(ctx, host, cfg)
	}

	if tcpingFlag && len(args) >= 2 {
		port, err := strconv.Atoi(args[1])
		if err == nil {
			return runTCPing(ctx, host, port)
		}
	}

	if len(args) >= 2 {
		port, err := strconv.Atoi(args[1])
		if err == nil {
			if udpMode {
				return runUDPCheck(ctx, host, port, cfg)
			}
			if watchFlag {
				return runWatch(ctx, host, port)
			}
			return runTCPCheck(ctx, host, port, cfg)
		}
	}

	if watchFlag {
		return runWatch(ctx, host, 0)
	}

	return runPing(ctx, host, cfg)
}

func runPing(ctx context.Context, host string, cfg *config.Config) error {
	p := ping.New(host, cfg.PingCount, cfg.PingInterval, cfg.PingTimeout)
	res, err := p.Run(ctx)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	fmt.Println(res.String())
	metrics.SetAvailability(host, "ping", res.Received > 0)
	metrics.SetPacketLoss(host, res.PacketLoss)
	writeExport(res)
	return nil
}

func runTCPCheck(ctx context.Context, host string, port int, cfg *config.Config) error {
	c := tcp.New(host, port, cfg.TCPTimeout, 3)
	res := c.Run(ctx)
	fmt.Printf("TCP Port Check: %s:%d\n", host, port)
	fmt.Printf("  State: %s\n", res.State)
	fmt.Printf("  Latency: %v (Min: %v, Max: %v, Avg: %v)\n", res.Latency, res.Min, res.Max, res.Avg)
	metrics.SetAvailability(fmt.Sprintf("%s:%d", host, port), "tcp", res.State == tcp.OPEN)
	metrics.SetLatency(fmt.Sprintf("%s:%d", host, port), "tcp", res.Latency)
	writeExport(res)
	return nil
}

func runTCPing(ctx context.Context, host string, port int) error {
	fmt.Printf("TCPing %s:%d - Ctrl+C to stop\n", host, port)
	fmt.Println(strings.Repeat("-", 50))

	var total, success, fail int
	var minLat, maxLat, totalLat time.Duration

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(done)
	}()

	seq := 0
	for {
		select {
		case <-done:
			if total > 0 {
				avgLat := totalLat / time.Duration(success)
				fmt.Println(strings.Repeat("-", 50))
				fmt.Printf("--- %s:%d tcping statistics ---\n", host, port)
				fmt.Printf("%d probes: %d successful, %d failed (%.1f%% loss)\n",
					total, success, fail, float64(fail)/float64(total)*100)
				if success > 0 {
					fmt.Printf("rtt min/avg/max = %v/%v/%v\n", minLat, avgLat, maxLat)
				}
			}
			return nil
		case <-ticker.C:
		}

		seq++
		start := time.Now()
		addr := fmt.Sprintf("%s:%d", host, port)
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		latency := time.Since(start)

		total++
		if err != nil {
			fail++
			fmt.Printf("seq=%d port=%d %-8s latency=%v\n", seq, port, "CLOSED", latency)
		} else {
			conn.Close()
			success++
			if success == 1 {
				minLat = latency
				maxLat = latency
			} else {
				if latency < minLat {
					minLat = latency
				}
				if latency > maxLat {
					maxLat = latency
				}
			}
			totalLat += latency
			fmt.Printf("seq=%d port=%d %-8s latency=%v\n", seq, port, "OPEN", latency)
		}
	}
}

func runUDPCheck(ctx context.Context, host string, port int, cfg *config.Config) error {
	proto := udp.DetectProtocol(port)
	c := udp.New(host, port, proto, cfg.UDPTimeout, 3)
	res := c.Run(ctx)
	fmt.Printf("UDP Port Check: %s:%d [%s]\n", host, port, proto)
	fmt.Printf("  State: %s\n", res.State)
	fmt.Printf("  Latency: %v\n", res.Latency)
	metrics.SetAvailability(fmt.Sprintf("%s:%d", host, port), "udp", res.State == udp.OPEN)
	writeExport(res)
	return nil
}

func runPortScan(ctx context.Context, host string, cfg *config.Config) error {
	portStrs := strings.Split(portsFlag, ",")
	var ports []int
	for _, ps := range portStrs {
		p, err := strconv.Atoi(strings.TrimSpace(ps))
		if err != nil {
			return fmt.Errorf("invalid port: %s", ps)
		}
		ports = append(ports, p)
	}

	scanType := portscan.ScanTCP
	if udpMode {
		scanType = portscan.ScanUDP
	}

	s := portscan.New(host, ports, scanType, cfg.TCPTimeout, cfg.ScanWorkers)
	results, err := s.Run(ctx)
	if err != nil {
		return fmt.Errorf("port scan failed: %w", err)
	}

	fmt.Printf("Port Scan Results for %s:\n", host)
	fmt.Printf("%-5s %-5s %-15s %s\n", "Port", "Proto", "State", "Service")
	fmt.Println(strings.Repeat("-", 50))
	for _, r := range results {
		fmt.Println(r.String())
	}
	return nil
}

func runDiscover(ctx context.Context, cidr string, cfg *config.Config) error {
	d := discover.New(cidr, discoverIcmp, discoverTcp, discoverArp, []int{22, 80, 443, 3389}, cfg.TCPTimeout, cfg.ScanWorkers)
	results, err := d.Run(ctx)
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	fmt.Printf("Host Discovery Results for %s:\n", cidr)
	fmt.Printf("%-16s %-8s %-12s %s\n", "IP", "State", "Discovery", "Hostname")
	fmt.Println(strings.Repeat("-", 60))
	for _, r := range results {
		fmt.Println(r.String())
	}
	aliveCount := 0
	for _, r := range results {
		if r.State == discover.Alive {
			aliveCount++
		}
	}
	fmt.Printf("\n%d/%d hosts alive\n", aliveCount, len(results))
	return nil
}

func runDNS(ctx context.Context, domain string) error {
	r := dns.New(domain, "", time.Second*5)
	res := r.Run(ctx)
	fmt.Println(res.String())
	return nil
}

func runTraceroute(ctx context.Context, host string) error {
	t := traceroute.New(host, traceroute.MethodICMP, 30, time.Second*3, 0)
	res, err := t.Run(ctx)
	if err != nil {
		return fmt.Errorf("traceroute failed: %w", err)
	}
	fmt.Println(res.String())
	return nil
}

func runMTR(ctx context.Context, host string) error {
	m := mtr.New(host, 10, time.Second, time.Second*3, 30)
	res, err := m.Run(ctx)
	if err != nil {
		return fmt.Errorf("mtr failed: %w", err)
	}
	fmt.Println(res.String())
	return nil
}

func runHTTPCheck(ctx context.Context, url string) error {
	c := httpcheck.New(url, time.Second*10, true)
	res := c.Run(ctx)
	fmt.Println(res.String())
	return nil
}

func runTLSCheck(ctx context.Context, host string) error {
	c := tlscheck.New(host, 443, time.Second*10)
	res := c.Run(ctx)
	fmt.Println(res.String())
	metrics.SetCertExpiry(host, float64(res.DaysRemaining))
	if res.ExpiringSoon && !res.Expired {
		logger.Warn("TLS certificate for %s expires in %d days", host, res.DaysRemaining)
	}
	return nil
}

func runWatch(ctx context.Context, host string, port int) error {
	interval, err := time.ParseDuration(intervalFlag)
	if err != nil {
		interval = 5 * time.Second
	}

	monType := monitoring.MonitorPing
	if port > 0 {
		monType = monitoring.MonitorTCP
	}

	m := monitoring.New(host, port, monType, interval, 0)
	ch := m.Start(ctx)

	fmt.Printf("Monitoring %s", host)
	if port > 0 {
		fmt.Printf(":%d", port)
	}
	fmt.Printf(" every %v\n", interval)
	fmt.Println(strings.Repeat("-", 60))

	for dp := range ch {
		status := "DOWN"
		if dp.Available {
			status = "UP"
		}
		fmt.Printf("[%s] %s - Status: %s", dp.Timestamp.Format("15:04:05"), host, status)
		if dp.Latency > 0 {
			fmt.Printf(", Latency: %v", dp.Latency)
		}
		if dp.ErrorMessage != "" {
			fmt.Printf(", Error: %s", dp.ErrorMessage)
		}
		fmt.Println()
	}
	return nil
}

func writeExport(data export.Exportable) {
	for _, output := range []struct {
		path   string
		format export.ExportFormat
	}{
		{jsonOutput, export.FormatJSON},
		{csvOutput, export.FormatCSV},
		{htmlOutput, export.FormatHTML},
	} {
		if output.path == "" {
			continue
		}
		e := export.NewExporter(output.format, output.path)
		e.Add(data)
		if err := e.Export(); err != nil {
			logger.Error("export failed: %v", err)
		}
	}
}
