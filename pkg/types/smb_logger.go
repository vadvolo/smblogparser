package types

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type SmbLogItem struct {
	Login     string
	Path      string
	File      string
	Action    string
	Device    string
	Timestamp time.Time
	Created   time.Time
}

func NewSmbLogItem() *SmbLogItem {
	return &SmbLogItem{
		Created: time.Now(),
	}
}

func (item *SmbLogItem) Print() {
	date, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		fmt.Sprintf("%e", err)
	}

	fmt.Println(string(date))
}

type Logger struct {
	Device string
	Items  []*SmbLogItem
}

// UserMetrics holds operation counts per user
type UserMetrics struct {
	User   string
	Device string
	Create int
	Open   int
	Modify int
	Delete int
}

func NewLogger(device string) *Logger {
	return &Logger{
		Device: device,
	}
}

// ParseLogLine parses a single log line from Loki
// Supports two formats:
//  1. Standard Samba logs (multi-line):
//     Line 1: [timestamp] ../../source3/smbd/open.c:1619(open_file)
//     Line 2:   username opened/closed file filepath read=Yes write=No (numopen=X)
//  2. Full audit logs:
//     smbd_audit: user|IP|machine|share|operation|status|filepath
func (l *Logger) ParseLogLine(logLine string) *SmbLogItem {
	// Check for full_audit format first
	if strings.Contains(logLine, "smbd_audit:") {
		return l.parseAuditLine(logLine)
	}

	// Check if this line contains operation keywords
	if !strings.Contains(logLine, "open_file") &&
		!strings.Contains(logLine, "close_normal_file") &&
		!strings.Contains(logLine, "pwrite") &&
		!strings.Contains(logLine, "unlink") &&
		!strings.Contains(logLine, "rmdir") {
		return nil
	}

	logItem := NewSmbLogItem()
	logItem.Device = l.Device

	// Extract timestamp from first line: [2025/09/09 13:01:08.165460, 2]
	timestampRe := regexp.MustCompile(`\[(\d{4}/\d{2}/\d{2}\s+\d{2}:\d{2}:\d{2})`)
	if matches := timestampRe.FindStringSubmatch(logLine); len(matches) > 1 {
		t, err := time.Parse("2006/01/02 15:04:05", matches[1])
		if err == nil {
			logItem.Timestamp = t
		}
	}

	// Extract username and file from second line
	// Format: "  skk opened file Конструкторский отдел/file.dwg read=Yes write=No (numopen=58)"
	// or: "  skk closed file Конструкторский отдел/file.dwg (numopen=63) NT_STATUS_OK"

	// Extract username (word after leading whitespace, before "opened" or "closed")
	// Use (?m) for multiline mode to make ^ match start of line
	userRe := regexp.MustCompile(`(?m)^\s+(\S+)\s+(?:opened|closed)`)
	if matches := userRe.FindStringSubmatch(logLine); len(matches) > 1 {
		logItem.Login = matches[1]
	}

	// Extract filename (between "file " and " read=" or " (numopen")
	fileRe := regexp.MustCompile(`file\s+(.+?)\s+(?:read=|\(numopen)`)
	if matches := fileRe.FindStringSubmatch(logLine); len(matches) > 1 {
		logItem.File = strings.TrimSpace(matches[1])
	}

	// Determine action based on keywords and write permissions
	switch {
	case strings.Contains(logLine, "opened") && strings.Contains(logLine, "read=Yes") && strings.Contains(logLine, "write=Yes"):
		logItem.Action = "modify"
	case strings.Contains(logLine, "opened") && strings.Contains(logLine, "write=Yes"):
		logItem.Action = "create"
	case strings.Contains(logLine, "opened"):
		logItem.Action = "open"
	case strings.Contains(logLine, "pwrite") || strings.Contains(logLine, "write"):
		logItem.Action = "modify"
	case strings.Contains(logLine, "unlink") || strings.Contains(logLine, "rmdir"):
		logItem.Action = "delete"
	case strings.Contains(logLine, "closed"):
		logItem.Action = "close"
	}

	return logItem
}

// parseAuditLine parses full_audit format logs
// Format: smbd_audit: user|IP|machine|share|operation|status|filepath
func (l *Logger) parseAuditLine(logLine string) *SmbLogItem {
	logItem := NewSmbLogItem()
	logItem.Device = l.Device

	// Extract timestamp if present in syslog format
	// Example: Sep 23 12:23:56 hostname smbd_audit: ...
	timestampRe := regexp.MustCompile(`^(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})`)
	if matches := timestampRe.FindStringSubmatch(logLine); len(matches) > 1 {
		// Parse syslog timestamp (without year)
		t, err := time.Parse("Jan 2 15:04:05", matches[1])
		if err == nil {
			// Add current year
			now := time.Now()
			logItem.Timestamp = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
		}
	}

	// Extract audit data after "smbd_audit:"
	auditRe := regexp.MustCompile(`smbd_audit:\s*(.+)`)
	if matches := auditRe.FindStringSubmatch(logLine); len(matches) > 1 {
		parts := strings.Split(matches[1], "|")
		if len(parts) >= 5 {
			logItem.Login = parts[0]
			operation := parts[4]

			// Get filepath if present
			if len(parts) >= 7 {
				logItem.File = parts[6]
			}

			// Map operation to action
			switch operation {
			case "open":
				logItem.Action = "open"
			case "pwrite":
				logItem.Action = "modify"
			case "unlink":
				logItem.Action = "delete"
			case "rmdir":
				logItem.Action = "delete"
			case "mkdir":
				logItem.Action = "create"
			case "rename":
				logItem.Action = "modify"
			case "close":
				logItem.Action = "close"
			}
		}
	}

	return logItem
}

// AggregateMetrics aggregates metrics per user
func (l *Logger) AggregateMetrics() map[string]*UserMetrics {
	metrics := make(map[string]*UserMetrics)

	for _, item := range l.Items {
		key := item.Login + "|" + item.Device
		if _, exists := metrics[key]; !exists {
			metrics[key] = &UserMetrics{
				User:   item.Login,
				Device: item.Device,
			}
		}

		switch item.Action {
		case "create":
			metrics[key].Create++
		case "open":
			metrics[key].Open++
		case "modify":
			metrics[key].Modify++
		case "delete":
			metrics[key].Delete++
		}
	}

	return metrics
}

func (l *Logger) ExportCVS() {
	fmt.Println(l.Items)
	for _, item := range l.Items {
		fmt.Printf("%s,%s,%s,%s,%s\n", item.Login, item.Device, item.Timestamp, item.File, item.Action)
	}
}
