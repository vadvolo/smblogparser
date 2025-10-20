# SMB Log Parser

A tool to parse Samba logs from Loki and export metrics to Prometheus.

## Features

- Queries Samba logs from Loki
- Parses SMB operations: create, open, modify, delete
- Aggregates metrics per user and device
- Pushes metrics to Prometheus Pushgateway
- Can be run as a cronjob for periodic metric collection

## Configuration

Create a `config.yaml` file with the following structure:

```yaml
loki:
  url: "http://localhost:3100"

prometheus:
  pushgateway_url: "http://localhost:9091"
  job_name: "smblogparser"

query:
  query: '{job="samba"}'
  lookback_ms: 300000  # 5 minutes
  limit: 5000
  device: "default"
```

## Build

```bash
make build
```

This creates a binary at `bin/smblogparser`.

## Usage

Run the parser with default config file (reads from Loki):
```bash
./bin/smblogparser
```

Run with a custom config file:
```bash
./bin/smblogparser -config /path/to/config.yaml
```

### Local Testing with File Input

For testing without Loki, use the `-file` flag to read logs from a text file:

```bash
# Build the binary
make build

# Test with example file
./bin/smblogparser -file example/test-logs.txt
```

**Note:** When using `-file`, the Prometheus Pushgateway must still be running, or you can comment out the push step in the code for testing.

Example test log format (one log entry per line):
```
2024/01/15 14:30:45 john open_file /share/documents/report.pdf read
2024/01/15 14:31:12 jane open_file /share/projects/code.go create
2024/01/15 14:32:03 john pwrite /share/documents/report.pdf
```

## Running as a Cronjob

To run every 5 minutes, add to your crontab:

```bash
*/5 * * * * /path/to/smblogparser -config /path/to/config.yaml >> /var/log/smblogparser.log 2>&1
```

Example for hourly execution:
```bash
0 * * * * /path/to/smblogparser -config /path/to/config.yaml >> /var/log/smblogparser.log 2>&1
```

## Metrics

The following metrics are exported to Prometheus:

- `smb_create_operations_total{user, device}` - Total create operations per user
- `smb_open_operations_total{user, device}` - Total open operations per user
- `smb_modify_operations_total{user, device}` - Total modify operations per user
- `smb_delete_operations_total{user, device}` - Total delete operations per user

## Development

Run tests:
```bash
make test
```

Clean build artifacts:
```bash
make clean
```

Install to `/usr/local/bin`:
```bash
make install
```
