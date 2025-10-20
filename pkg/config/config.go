package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Loki       LokiConfig       `yaml:"loki"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Query      QueryConfig      `yaml:"query"`
}

type LokiConfig struct {
	URL string `yaml:"url"`
}

type PrometheusConfig struct {
	PushgatewayURL string `yaml:"pushgateway_url"`
	JobName        string `yaml:"job_name"`
}

type QueryConfig struct {
	Query      string `yaml:"query"`
	LookbackMs int64  `yaml:"lookback_ms"` // How far back to query (in milliseconds)
	Limit      int    `yaml:"limit"`
	Device     string `yaml:"device"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if cfg.Query.LookbackMs == 0 {
		cfg.Query.LookbackMs = 5 * 60 * 1000 // 5 minutes default
	}
	if cfg.Query.Limit == 0 {
		cfg.Query.Limit = 5000
	}
	if cfg.Prometheus.JobName == "" {
		cfg.Prometheus.JobName = "smblogparser"
	}

	return &cfg, nil
}

func (c *QueryConfig) GetTimeRange() (time.Time, time.Time) {
	end := time.Now()
	start := end.Add(-time.Duration(c.LookbackMs) * time.Millisecond)
	return start, end
}