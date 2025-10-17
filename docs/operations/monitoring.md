# Monitoring Guide

Guide to monitoring Go-Doc-Go worker health, performance, and processing status.

---

## Table of Contents

- [Overview](#overview)
- [Built-In Monitoring](#built-in-monitoring)
  - [Log-Based Metrics](#log-based-metrics)
  - [Worker Status Logging](#worker-status-logging)
  - [Document Processing Logs](#document-processing-logs)
- [Database Monitoring](#database-monitoring)
  - [Queue Status Queries](#queue-status-queries)
  - [Worker Activity Monitoring](#worker-activity-monitoring)
  - [Performance Metrics](#performance-metrics)
- [System Resource Monitoring](#system-resource-monitoring)
  - [CPU and Memory Usage](#cpu-and-memory-usage)
  - [Disk I/O](#disk-io)
  - [Network Activity](#network-activity)
- [Health Checks](#health-checks)
- [Alerting Strategies](#alerting-strategies)
- [Future Monitoring Features](#future-monitoring-features)

---

## Overview

Go-Doc-Go currently provides **log-based monitoring** with database queries for operational visibility. This guide covers:

- **Current Features**: Log analysis, database queries, system monitoring
- **Future Features**: Native metrics, health endpoints, Prometheus integration (planned)

### Monitoring Philosophy

1. **Simplicity First**: Start with logs and database queries
2. **Operational Focus**: Monitor what affects reliability and performance
3. **Actionable Metrics**: Focus on metrics that drive decisions
4. **Scalability**: Design monitoring to work across distributed workers

---

## Built-In Monitoring

### Log-Based Metrics

Go-Doc-Go workers emit structured logs that can be parsed for monitoring.

#### Capturing Worker Logs

```bash
## Development: View logs in real-time
./bin/goworker --config config.toml 2>&1 | tee worker.log

## Production: Run in background with log rotation
nohup ./bin/goworker --config config.toml > /var/log/goworker/worker.log 2>&1 &

## With logrotate
cat > /etc/logrotate.d/goworker << 'EOF'
/var/log/goworker/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0644 goworker goworker
}
EOF
```go

---

### Worker Status Logging

#### Key Log Messages

**Worker Startup**:
```toml
[INFO] WORKER STARTED worker_id=worker_hostname_12345 max_documents=0 workers=4
```

**Document Processing**:
```toml
[INFO] Processing document doc_id=abc123 worker_id=worker_hostname_12345
[INFO] Successfully processed document doc_id=abc123 duration=1.23s
[ERROR] Failed to process document doc_id=abc123 error=...
```

**Worker Shutdown**:
```toml
[INFO] WORKER COMPLETED total=1234 successful=1200 failed=34 duration=1h23m45s
[INFO] Graceful shutdown complete
```bash

#### Parsing Logs for Metrics

```bash
## Count documents processed
grep "Successfully processed" worker.log | wc -l

## Count failures
grep "Failed to process" worker.log | wc -l

## Average processing time
grep "Successfully processed" worker.log | \
  grep -oP 'duration=\K[0-9.]+' | \
  awk '{sum+=$1; count++} END {print sum/count}'

## Find slowest documents
grep "Successfully processed" worker.log | \
  grep -oP 'doc_id=\K[^ ]+|duration=\K[0-9.]+s' | \
  paste - - | \
  sort -k2 -nr | \
  head -10

## Error summary
grep "ERROR" worker.log | \
  cut -d' ' -f4- | \
  sort | uniq -c | sort -nr
```bash

#### Structured Log Parsing with jq (if using JSON logs)

```bash
## If logs are in JSON format (future feature)
## Count by status
cat worker.log | jq -r '.status' | sort | uniq -c

## Average duration
cat worker.log | jq -r 'select(.event=="document_processed") | .duration' | \
  awk '{sum+=$1; count++} END {print sum/count}'

## Error distribution
cat worker.log | jq -r 'select(.level=="error") | .error_type' | \
  sort | uniq -c | sort -nr
```sql

---

### Document Processing Logs

#### Monitoring Document Flow

```bash
## Documents claimed per minute
grep "claimed document" worker.log | \
  awk '{print $1" "$2}' | \
  cut -d: -f1-2 | \
  sort | uniq -c

## Processing rate (docs/minute)
total=$(grep "Successfully processed" worker.log | wc -l)
start=$(head -1 worker.log | awk '{print $1" "$2}')
end=$(tail -1 worker.log | awk '{print $1" "$2}')
echo "Processed: $total documents"
## (calculate time difference manually or with date)

## Find stuck documents (claimed but not completed in 10+ min)
grep "claimed document" worker.log | \
  awk '{print $NF}' | \
  while read doc_id; do
    if ! grep -q "Successfully processed.*$doc_id" worker.log; then
      echo "Stuck: $doc_id"
    fi
  done
```

---

## Database Monitoring

### Queue Status Queries

#### SQLite Monitoring

```bash
## Overall queue status
sqlite3 ./data/jobs.db << 'EOF'
SELECT
  status,
  COUNT(*) as count,
  ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM documents), 2) as percentage
FROM documents
GROUP BY status
ORDER BY count DESC;
EOF

## Processing rate over time
sqlite3 ./data/jobs.db << 'EOF'
SELECT
  DATE(completed_at) as date,
  COUNT(*) as completed
FROM documents
WHERE status = 'completed'
GROUP BY DATE(completed_at)
ORDER BY date DESC
LIMIT 7;
EOF

## Failed documents analysis
sqlite3 ./data/jobs.db << 'EOF'
SELECT
  error,
  COUNT(*) as count
FROM documents
WHERE status = 'failed'
GROUP BY error
ORDER BY count DESC
LIMIT 10;
EOF

## Queue depth over time
sqlite3 ./data/jobs.db << 'EOF'
SELECT
  COUNT(*) as pending_count,
  (SELECT COUNT(*) FROM documents WHERE status='processing') as processing_count,
  (SELECT COUNT(*) FROM documents WHERE status='completed') as completed_count,
  (SELECT COUNT(*) FROM documents WHERE status='failed') as failed_count;
EOF
```bash

#### PostgreSQL Monitoring

```bash
## Overall queue status
psql -d go_doc_go << 'EOF'
SELECT
  status,
  COUNT(*) as count,
  ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM documents), 2) as percentage
FROM documents
GROUP BY status
ORDER BY count DESC;
EOF

## Active workers
psql -d go_doc_go << 'EOF'
SELECT
  worker_id,
  COUNT(*) as documents_processing,
  MIN(claimed_at) as first_claim,
  MAX(claimed_at) as last_claim
FROM documents
WHERE status = 'processing'
GROUP BY worker_id
ORDER BY documents_processing DESC;
EOF

## Processing throughput (last hour)
psql -d go_doc_go << 'EOF'
SELECT
  DATE_TRUNC('hour', completed_at) as hour,
  COUNT(*) as completed,
  ROUND(AVG(EXTRACT(EPOCH FROM (completed_at - claimed_at))), 2) as avg_seconds
FROM documents
WHERE status = 'completed'
  AND completed_at > NOW() - INTERVAL '24 hours'
GROUP BY DATE_TRUNC('hour', completed_at)
ORDER BY hour DESC;
EOF

## Slow documents (> 5 minutes)
psql -d go_doc_go << 'EOF'
SELECT
  doc_id,
  worker_id,
  claimed_at,
  EXTRACT(EPOCH FROM (NOW() - claimed_at)) as seconds_processing
FROM documents
WHERE status = 'processing'
  AND claimed_at < NOW() - INTERVAL '5 minutes'
ORDER BY seconds_processing DESC
LIMIT 20;
EOF

## Database connection stats
psql -d go_doc_go << 'EOF'
SELECT
  state,
  COUNT(*) as connections,
  MAX(query_start) as last_query
FROM pg_stat_activity
WHERE datname = 'go_doc_go'
GROUP BY state;
EOF
```go

---

### Worker Activity Monitoring

#### Identifying Active Workers

```bash
## SQLite: List active workers
sqlite3 ./data/jobs.db << 'EOF'
SELECT
  worker_id,
  COUNT(*) as processing_count,
  MIN(claimed_at) as first_claim,
  MAX(claimed_at) as last_claim
FROM documents
WHERE status = 'processing'
GROUP BY worker_id;
EOF

## PostgreSQL: Worker heartbeat check
psql -d go_doc_go << 'EOF'
SELECT
  worker_id,
  COUNT(*) as processing,
  MAX(claimed_at) as last_heartbeat,
  CASE
    WHEN MAX(claimed_at) < NOW() - INTERVAL '5 minutes'
    THEN 'STALE'
    ELSE 'ACTIVE'
  END as status
FROM documents
WHERE status = 'processing'
GROUP BY worker_id
ORDER BY last_heartbeat DESC;
EOF
```bash

#### Detecting Dead Workers

```bash
## Find documents claimed by workers that haven't updated in 10+ minutes
## PostgreSQL
psql -d go_doc_go << 'EOF'
SELECT
  worker_id,
  COUNT(*) as stuck_documents,
  MAX(claimed_at) as last_seen
FROM documents
WHERE status = 'processing'
  AND claimed_at < NOW() - INTERVAL '10 minutes'
GROUP BY worker_id;
EOF

## Release documents from dead workers
psql -d go_doc_go << 'EOF'
UPDATE documents
SET
  status = 'pending',
  worker_id = NULL,
  claimed_at = NULL
WHERE status = 'processing'
  AND claimed_at < NOW() - INTERVAL '10 minutes';
EOF
```

---

### Performance Metrics

#### Processing Statistics

```bash
## SQLite: Average processing time
sqlite3 ./data/jobs.db << 'EOF'
SELECT
  AVG(CAST((julianday(completed_at) - julianday(claimed_at)) * 86400 AS INTEGER)) as avg_seconds,
  MIN(CAST((julianday(completed_at) - julianday(claimed_at)) * 86400 AS INTEGER)) as min_seconds,
  MAX(CAST((julianday(completed_at) - julianday(claimed_at)) * 86400 AS INTEGER)) as max_seconds
FROM documents
WHERE status = 'completed'
  AND completed_at IS NOT NULL
  AND claimed_at IS NOT NULL;
EOF

## PostgreSQL: Detailed processing stats
psql -d go_doc_go << 'EOF'
SELECT
  ROUND(AVG(EXTRACT(EPOCH FROM (completed_at - claimed_at))), 2) as avg_seconds,
  ROUND(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (completed_at - claimed_at))), 2) as median_seconds,
  ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (completed_at - claimed_at))), 2) as p95_seconds,
  ROUND(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (completed_at - claimed_at))), 2) as p99_seconds
FROM documents
WHERE status = 'completed'
  AND completed_at IS NOT NULL
  AND claimed_at IS NOT NULL;
EOF
```bash

#### Throughput Monitoring

```bash
## Documents per hour (PostgreSQL)
psql -d go_doc_go << 'EOF'
SELECT
  DATE_TRUNC('hour', completed_at) as hour,
  COUNT(*) as documents_completed,
  ROUND(COUNT(*) / 60.0, 2) as docs_per_minute
FROM documents
WHERE status = 'completed'
  AND completed_at > NOW() - INTERVAL '24 hours'
GROUP BY hour
ORDER BY hour DESC;
EOF
```

---

## System Resource Monitoring

### CPU and Memory Usage

#### Monitor Worker Processes

```bash
## Find worker PIDs
pgrep -a goworker

## Real-time monitoring with top
top -pid $(pgrep goworker)

## Or with htop (more detailed)
htop -p $(pgrep goworker | tr '\n' ',')

## CPU usage over time
while true; do
  ps -p $(pgrep goworker) -o pid,pcpu,pmem,etime,comm | tail -n +2
  sleep 60
done | tee cpu-monitor.log

## Memory usage trend
while true; do
  date=$(date '+%Y-%m-%d %H:%M:%S')
  mem=$(ps -p $(pgrep goworker) -o rss= | awk '{sum+=$1} END {print sum/1024 " MB"}')
  echo "$date: $mem"
  sleep 300  # Every 5 minutes
done | tee memory-monitor.log
```bash

#### System-Wide Monitoring

```bash
## Overall system stats
vmstat 5  # Every 5 seconds

## Memory breakdown
free -h

## macOS memory stats
vm_stat

## CPU load average
uptime

## Process resource summary
ps aux --sort=-rss | head -20  # Linux
ps aux -m | head -20           # macOS
```

---

### Disk I/O

```bash
## I/O statistics
iostat -x 5  # Linux
iostat -w 5  # macOS

## Disk usage
df -h

## Check database file size
ls -lh ./data/jobs.db

## PostgreSQL database size
psql -d go_doc_go -c "SELECT pg_size_pretty(pg_database_size('go_doc_go'));"

## Parquet output size
du -sh ./data/analytics.parquet
```

---

### Network Activity

```bash
## Network connections (if using remote database)
netstat -an | grep 5432  # PostgreSQL port

## Or with lsof
lsof -i :5432

## Monitor network traffic (Linux)
iftop

## Or with nethogs
sudo nethogs
```

---

## Health Checks

### Manual Health Check Script

```bash
#!/bin/bash
## health-check.sh - Manual health check for Go-Doc-Go workers

## Configuration
DB_PATH="./data/jobs.db"
LOG_FILE="./worker.log"
MAX_WORKER_IDLE_MINUTES=10

echo "=== Go-Doc-Go Health Check ==="
echo "Timestamp: $(date)"
echo

## Check 1: Is worker process running?
if pgrep -f goworker > /dev/null; then
  echo "✓ Worker process is running"
  pids=$(pgrep goworker | tr '\n' ' ')
  echo "  PIDs: $pids"
else
  echo "✗ Worker process is NOT running"
  exit 1
fi

## Check 2: CPU and memory usage
echo
echo "Resource Usage:"
ps -p $(pgrep goworker | head -1) -o pid,pcpu,pmem,rss,etime,comm

## Check 3: Recent log activity
echo
if [ -f "$LOG_FILE" ]; then
  last_log=$(tail -1 "$LOG_FILE" | awk '{print $1, $2}')
  echo "Last log entry: $last_log"
else
  echo "⚠ Log file not found: $LOG_FILE"
fi

## Check 4: Queue status
echo
echo "Queue Status:"
if [ -f "$DB_PATH" ]; then
  sqlite3 "$DB_PATH" << 'EOF'
SELECT
  status,
  COUNT(*) as count
FROM documents
GROUP BY status;
EOF
else
  echo "⚠ Database not found: $DB_PATH"
fi

## Check 5: Active workers
echo
echo "Active Workers:"
sqlite3 "$DB_PATH" << 'EOF'
SELECT
  worker_id,
  COUNT(*) as processing,
  MAX(claimed_at) as last_activity
FROM documents
WHERE status = 'processing'
GROUP BY worker_id;
EOF

## Check 6: Stuck documents
echo
stuck_count=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM documents WHERE status='processing' AND datetime(claimed_at) < datetime('now', '-${MAX_WORKER_IDLE_MINUTES} minutes');")
if [ "$stuck_count" -gt 0 ]; then
  echo "⚠ Found $stuck_count stuck documents (processing > ${MAX_WORKER_IDLE_MINUTES} min)"
else
  echo "✓ No stuck documents"
fi

## Check 7: Error rate
echo
error_count=$(grep -c "ERROR" "$LOG_FILE" 2>/dev/null || echo 0)
total_count=$(grep -c "Successfully processed" "$LOG_FILE" 2>/dev/null || echo 0)
if [ "$total_count" -gt 0 ]; then
  error_rate=$(echo "scale=2; $error_count * 100 / $total_count" | bc)
  echo "Error rate: ${error_rate}% ($error_count errors / $total_count processed)"
else
  echo "No processing activity logged"
fi

echo
echo "=== Health Check Complete ==="
```bash

Make executable:
```bash
chmod +x health-check.sh
```bash

Run periodically:
```bash
## Run every 5 minutes
*/5 * * * * /path/to/health-check.sh >> /var/log/goworker/health.log 2>&1
```bash

---

## Alerting Strategies

### Log-Based Alerts

#### Using logwatch or similar tools

```bash
## Create alert rules for critical errors
cat > /etc/logwatch/conf/services/goworker.conf << 'EOF'
Title = "Go-Doc-Go Worker"
LogFile = /var/log/goworker/worker.log

## Alert on these patterns
*ERROR*
*FATAL*
*panic*
*failed to process*
EOF
```bash

#### Simple bash alert script

```bash
#!/bin/bash
## alert-on-errors.sh - Alert when errors exceed threshold

LOG_FILE="./worker.log"
ERROR_THRESHOLD=10
ALERT_EMAIL="admin@example.com"

## Count errors in last 10 minutes
error_count=$(tail -1000 "$LOG_FILE" | grep "ERROR" | wc -l)

if [ "$error_count" -gt "$ERROR_THRESHOLD" ]; then
  echo "High error rate: $error_count errors in last 10 minutes" | \
    mail -s "Go-Doc-Go Alert: High Error Rate" "$ALERT_EMAIL"
fi
```bash

### Database-Based Alerts

```bash
#!/bin/bash
## alert-on-queue-backlog.sh - Alert when queue grows too large

DB_PATH="./data/jobs.db"
PENDING_THRESHOLD=1000
ALERT_EMAIL="admin@example.com"

pending_count=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM documents WHERE status='pending';")

if [ "$pending_count" -gt "$PENDING_THRESHOLD" ]; then
  echo "Queue backlog: $pending_count pending documents" | \
    mail -s "Go-Doc-Go Alert: Queue Backlog" "$ALERT_EMAIL"
fi
```go

---

## Future Monitoring Features

### Planned Enhancements

#### 1. Native Metrics Endpoint (Planned)

```bash
## Future: HTTP metrics endpoint
curl http://localhost:8080/metrics

## Example output:
## goworker_documents_processed_total{status="success"} 1234
## goworker_documents_processed_total{status="failed"} 56
## goworker_processing_duration_seconds_sum 4567.89
## goworker_processing_duration_seconds_count 1234
## goworker_workers_active 4
```bash

#### 2. Health Check Endpoint (Planned)

```bash
## Future: HTTP health check
curl http://localhost:8080/health

## Example output:
## {
##   "status": "healthy",
##   "uptime_seconds": 3600,
##   "workers_active": 4,
##   "documents_processing": 12,
##   "queue_depth": 234
## }
```yaml

#### 3. Prometheus Integration (Planned)

```yaml
## Future: Prometheus scrape config
scrape_configs:
  - job_name: 'goworker'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```json

#### 4. Structured Logging (Planned)

```json
// Future: JSON structured logs
{
  "timestamp": "2025-01-16T10:30:00Z",
  "level": "info",
  "event": "document_processed",
  "doc_id": "abc123",
  "worker_id": "worker_01",
  "duration_ms": 1234,
  "success": true
}
```

---

## Monitoring Checklist

### Daily Monitoring Tasks

- [ ] Check worker processes are running (`pgrep goworker`)
- [ ] Review error logs (`grep ERROR worker.log | tail -20`)
- [ ] Check queue depth (`SELECT COUNT(*) ... WHERE status='pending'`)
- [ ] Verify processing throughput (docs/hour)
- [ ] Check for stuck documents (processing > 10 min)

### Weekly Monitoring Tasks

- [ ] Review processing statistics (avg/median/p95 times)
- [ ] Analyze failed documents by error type
- [ ] Check disk space usage
- [ ] Review resource usage trends (CPU, memory)
- [ ] Verify no dead worker processes

### Monthly Monitoring Tasks

- [ ] Analyze long-term throughput trends
- [ ] Review capacity planning metrics
- [ ] Audit alert configurations
- [ ] Test recovery procedures
- [ ] Update monitoring dashboards

---

## Best Practices

### 1. Centralized Logging

```bash
## Use rsyslog or similar for centralized logs
## /etc/rsyslog.d/goworker.conf
if $programname == 'goworker' then /var/log/goworker/worker.log
& stop
```bash

### 2. Log Rotation

```bash
## Configure logrotate
/var/log/goworker/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0644 goworker goworker
    sharedscripts
    postrotate
        killall -SIGUSR1 goworker || true
    endscript
}
```go

### 3. Monitoring Documentation

Maintain a runbook with:
- Baseline metrics (normal CPU/memory/throughput)
- Alert thresholds and escalation procedures
- Common issues and resolutions
- Contact information for on-call staff

### 4. Regular Testing

```bash
## Test alerting system monthly
echo "Test alert" | mail -s "Go-Doc-Go Alert Test" admin@example.com

## Verify health check script
./health-check.sh

## Test recovery procedures
## 1. Stop worker
## 2. Verify alerts trigger
## 3. Start worker
## 4. Verify recovery
```

---

## Related Documentation

- [Troubleshooting Guide](troubleshooting.md) - Diagnose and fix common issues
- [Scaling Guide](scaling.md) - Performance optimization and distributed setup
- [Quick Reference](../../QUICK_REFERENCE.md) - Common commands and configurations
- [Configuration Reference](../configuration/README.md) - All configuration options

---

**Last Updated**: 2025-01-16
---

## Related Documentation

- **Previous**: [Scaling Guide](scaling.md)
- **Next**: [Troubleshooting](troubleshooting.md)
- **Up**: [Documentation Home](../README.md)

### Quick Links

- [Documentation Home](../README.md)
- [Quick Reference](../../QUICK_REFERENCE.md)
- [CLI Reference](../reference/cli.md)
- [Configuration Overview](../configuration/README.md)
