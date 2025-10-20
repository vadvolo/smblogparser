package prometheus

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

type Client struct {
	pushgatewayURL string
	jobName        string
	registry       *prometheus.Registry
}

// MetricsCollector holds all the Prometheus metrics
type MetricsCollector struct {
	CreateOps *prometheus.GaugeVec
	OpenOps   *prometheus.GaugeVec
	ModifyOps *prometheus.GaugeVec
	DeleteOps *prometheus.GaugeVec
}

func NewClient(pushgatewayURL, jobName string) *Client {
	return &Client{
		pushgatewayURL: pushgatewayURL,
		jobName:        jobName,
		registry:       prometheus.NewRegistry(),
	}
}

// NewMetricsCollector creates a new metrics collector
func (c *Client) NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		CreateOps: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smb_create_operations_total",
				Help: "Total number of SMB create operations per user",
			},
			[]string{"user", "device"},
		),
		OpenOps: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smb_open_operations_total",
				Help: "Total number of SMB open operations per user",
			},
			[]string{"user", "device"},
		),
		ModifyOps: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smb_modify_operations_total",
				Help: "Total number of SMB modify operations per user",
			},
			[]string{"user", "device"},
		),
		DeleteOps: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "smb_delete_operations_total",
				Help: "Total number of SMB delete operations per user",
			},
			[]string{"user", "device"},
		),
	}

	c.registry.MustRegister(mc.CreateOps, mc.OpenOps, mc.ModifyOps, mc.DeleteOps)
	return mc
}

// Push sends metrics to Pushgateway
func (c *Client) Push() error {
	if err := push.New(c.pushgatewayURL, c.jobName).
		Gatherer(c.registry).
		Push(); err != nil {
		return fmt.Errorf("failed to push metrics: %w", err)
	}
	return nil
}