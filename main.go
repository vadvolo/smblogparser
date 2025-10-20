package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/vadvolo/smblogparser/pkg/config"
	"github.com/vadvolo/smblogparser/pkg/loki"
	"github.com/vadvolo/smblogparser/pkg/prometheus"
	"github.com/vadvolo/smblogparser/pkg/types"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func writeLokiDebugFile(filePath string, logs []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write header
	_, err = writer.WriteString(fmt.Sprintf("# Loki Debug Output - %d log lines\n", len(logs)))
	if err != nil {
		return err
	}
	_, err = writer.WriteString(fmt.Sprintf("# Timestamp: %s\n\n", time.Now().Format(time.RFC3339)))
	if err != nil {
		return err
	}

	// Write each log line with a separator
	for i, logLine := range logs {
		_, err = writer.WriteString(fmt.Sprintf("=== Log Entry #%d ===\n%s\n\n", i+1, logLine))
		if err != nil {
			return err
		}
	}

	return nil
}

// reconstructMultilineEntries combines Loki's single-line results into multi-line entries
// Loki returns each line separately, but Samba logs are multi-line:
//   Line 1: [timestamp] ../../source3/smbd/open.c:1619(open_file)
//   Line 2:   username opened file path read=Yes write=No (numopen=X)
// This function reconstructs them by combining timestamp lines with their detail lines
func reconstructMultilineEntries(logs []string) []string {
	var reconstructed []string
	var i = 0

	for i < len(logs) {
		line := logs[i]

		// Check if this is a timestamp line starting with '['
		if len(line) > 0 && line[0] == '[' {
			// Check if there's a next line and it's a detail line (starts with whitespace)
			if i+1 < len(logs) && len(logs[i+1]) > 0 && (logs[i+1][0] == ' ' || logs[i+1][0] == '\t') {
				// Combine timestamp line with detail line
				combined := line + "\n" + logs[i+1]
				reconstructed = append(reconstructed, combined)
				i += 2 // Skip both lines
				continue
			}
		}

		// If not a multi-line pair, keep as single line
		reconstructed = append(reconstructed, line)
		i++
	}

	return reconstructed
}

func readLogsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var logs []string
	scanner := bufio.NewScanner(file)

	var currentEntry string
	for scanner.Scan() {
		line := scanner.Text()

		// Check if it's an audit log (single line format)
		if strings.Contains(line, "smbd_audit:") {
			logs = append(logs, line)
			continue
		}

		// If line starts with '[', it's a new Samba log entry timestamp line
		if len(line) > 0 && line[0] == '[' {
			// If we have a previous entry, save it
			if currentEntry != "" {
				logs = append(logs, currentEntry)
			}
			currentEntry = line
		} else if currentEntry != "" {
			// Continuation line - append to current entry with newline
			currentEntry += "\n" + line
		}
	}

	// Don't forget the last entry
	if currentEntry != "" {
		logs = append(logs, currentEntry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config file")
	inputFile := flag.String("file", "", "Read logs from file instead of Loki (for testing)")
	debugFile := flag.String("debug", "", "Write raw Loki output to this file for debugging")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var logs []string

	// Read from file or Loki
	if *inputFile != "" {
		log.Printf("Reading logs from file: %s", *inputFile)
		logs, err = readLogsFromFile(*inputFile)
		if err != nil {
			log.Fatalf("Failed to read logs from file: %v", err)
		}
		log.Printf("Read %d log lines from file", len(logs))
	} else {
		// Initialize Loki client
		lokiClient := loki.NewClient(cfg.Loki.URL)

		// Query logs from Loki
		start, end := cfg.Query.GetTimeRange()
		log.Printf("Querying Loki from %v to %v with query: %s", start, end, cfg.Query.Query)

		ctx := context.Background()
		logs, err = lokiClient.QueryRange(ctx, cfg.Query.Query, start, end, cfg.Query.Limit)
		if err != nil {
			log.Fatalf("Failed to query Loki: %v", err)
		}

		log.Printf("Retrieved %d log lines from Loki", len(logs))

		// Debug mode: write raw logs to file
		if *debugFile != "" {
			if err := writeLokiDebugFile(*debugFile, logs); err != nil {
				log.Printf("Warning: failed to write debug file: %v", err)
			} else {
				log.Printf("Debug: wrote %d log lines to %s", len(logs), *debugFile)
			}
		}

		// Reconstruct multi-line entries from Loki's single-line results
		logs = reconstructMultilineEntries(logs)
		log.Printf("Reconstructed into %d multi-line log entries", len(logs))
	}

	// Parse logs
	logger := types.NewLogger(cfg.Query.Device)

	// Debug: show first few log lines
	if len(logs) > 0 {
		log.Printf("Sample log lines from Loki:")
		for i := 0; i < 3 && i < len(logs); i++ {
			log.Printf("  Line %d: %s", i+1, logs[i][:min(100, len(logs[i]))])
		}
	}

	for _, logLine := range logs {
		item := logger.ParseLogLine(logLine)
		if item != nil {
			logger.Items = append(logger.Items, item)
		}
	}

	log.Printf("Parsed %d SMB log items", len(logger.Items))

	// Aggregate metrics per user
	metrics := logger.AggregateMetrics()
	log.Printf("Aggregated metrics for %d users", len(metrics))

	// Initialize Prometheus client and push metrics
	promClient := prometheus.NewClient(cfg.Prometheus.PushgatewayURL, cfg.Prometheus.JobName)
	metricsCollector := promClient.NewMetricsCollector()

	// Set metrics
	for _, m := range metrics {
		metricsCollector.CreateOps.WithLabelValues(m.User, m.Device).Set(float64(m.Create))
		metricsCollector.OpenOps.WithLabelValues(m.User, m.Device).Set(float64(m.Open))
		metricsCollector.ModifyOps.WithLabelValues(m.User, m.Device).Set(float64(m.Modify))
		metricsCollector.DeleteOps.WithLabelValues(m.User, m.Device).Set(float64(m.Delete))

		log.Printf("User: %s, Device: %s - Create: %d, Open: %d, Modify: %d, Delete: %d",
			m.User, m.Device, m.Create, m.Open, m.Modify, m.Delete)
	}

	// Push to Prometheus
	if err := promClient.Push(); err != nil {
		log.Fatalf("Failed to push metrics to Prometheus: %v", err)
	}

	log.Println("Successfully pushed metrics to Prometheus Pushgateway")
	os.Exit(0)
}
