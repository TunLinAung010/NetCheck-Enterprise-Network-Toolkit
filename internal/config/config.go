package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	LogLevel      string        `mapstructure:"log_level"`
	LogJSON       bool          `mapstructure:"log_json"`
	PingCount     int           `mapstructure:"ping_count"`
	PingInterval  time.Duration `mapstructure:"ping_interval"`
	PingTimeout   time.Duration `mapstructure:"ping_timeout"`
	TCPTimeout    time.Duration `mapstructure:"tcp_timeout"`
	UDPTimeout    time.Duration `mapstructure:"udp_timeout"`
	ScanWorkers   int           `mapstructure:"scan_workers"`
	WatchInterval time.Duration `mapstructure:"watch_interval"`
	Prometheus    bool          `mapstructure:"prometheus"`
	PrometheusPort int          `mapstructure:"prometheus_port"`
	WebEnabled    bool          `mapstructure:"web_enabled"`
	WebPort       int           `mapstructure:"web_port"`
	TelegramToken string        `mapstructure:"telegram_token"`
	TelegramChat  string        `mapstructure:"telegram_chat"`
	DiscordWebhook string       `mapstructure:"discord_webhook"`
	SMTPServer    string        `mapstructure:"smtp_server"`
	SMTPPort      int           `mapstructure:"smtp_port"`
	SMTPUser      string        `mapstructure:"smtp_user"`
	SMTPPass      string        `mapstructure:"smtp_pass"`
	EmailFrom     string        `mapstructure:"email_from"`
	EmailTo       string        `mapstructure:"email_to"`
	AlertOnDown   bool          `mapstructure:"alert_on_down"`
	AlertOnClose  bool          `mapstructure:"alert_on_close"`
	AlertOnLatency time.Duration `mapstructure:"alert_on_latency"`
	AlertOnCert   bool          `mapstructure:"alert_on_cert"`
}

func Default() *Config {
	return &Config{
		LogLevel:       "info",
		LogJSON:        false,
		PingCount:      4,
		PingInterval:   1 * time.Second,
		PingTimeout:    5 * time.Second,
		TCPTimeout:     5 * time.Second,
		UDPTimeout:     3 * time.Second,
		ScanWorkers:    100,
		WatchInterval:  5 * time.Second,
		PrometheusPort: 9090,
		WebPort:        8080,
		SMTPPort:       587,
	}
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("netcheck")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.netcheck")
		v.AddConfigPath("/etc/netcheck")
	}

	v.SetDefault("log_level", "info")
	v.SetDefault("log_json", false)
	v.SetDefault("ping_count", 4)
	v.SetDefault("ping_interval", "1s")
	v.SetDefault("ping_timeout", "5s")
	v.SetDefault("tcp_timeout", "5s")
	v.SetDefault("udp_timeout", "3s")
	v.SetDefault("scan_workers", 100)
	v.SetDefault("watch_interval", "5s")
	v.SetDefault("prometheus_port", 9090)
	v.SetDefault("web_port", 8080)
	v.SetDefault("smtp_port", 587)

	v.AutomaticEnv()
	v.SetEnvPrefix("NETCHECK")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.PingCount < 1 {
		return fmt.Errorf("ping_count must be >= 1")
	}
	if c.ScanWorkers < 1 {
		return fmt.Errorf("scan_workers must be >= 1")
	}
	if c.PrometheusPort < 1 || c.PrometheusPort > 65535 {
		return fmt.Errorf("invalid prometheus_port")
	}
	if c.WebPort < 1 || c.WebPort > 65535 {
		return fmt.Errorf("invalid web_port")
	}
	return nil
}

func WriteDefaultConfig(path string) error {
	cfg := Default()
	data := fmt.Sprintf(`# NetCheck Configuration
log_level: %s
log_json: %t
ping_count: %d
ping_interval: %s
ping_timeout: %s
tcp_timeout: %s
udp_timeout: %s
scan_workers: %d
watch_interval: %s
prometheus: false
prometheus_port: %d
web_enabled: false
web_port: %d

# Alerting
# telegram_token: ""
# telegram_chat: ""
# discord_webhook: ""
# smtp_server: ""
# smtp_port: 587
# smtp_user: ""
# smtp_pass: ""
# email_from: ""
# email_to: ""
alert_on_down: false
alert_on_close: false
alert_on_latency: 0s
alert_on_cert: false
`,
		cfg.LogLevel,
		cfg.LogJSON,
		cfg.PingCount,
		cfg.PingInterval,
		cfg.PingTimeout,
		cfg.TCPTimeout,
		cfg.UDPTimeout,
		cfg.ScanWorkers,
		cfg.WatchInterval,
		cfg.PrometheusPort,
		cfg.WebPort,
	)
	return os.WriteFile(path, []byte(data), 0644)
}
