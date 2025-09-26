# Per-Source Discovery Intervals

## Overview

The Go-Doc-Go worker system now supports configurable discovery intervals for each content source, allowing you to optimize resource usage and freshness based on the update patterns of different data sources.

## Configuration

Each content source can specify its own discovery interval using:
- `discovery_interval` - The standard field name for all sources

### Priority Order

The system determines the discovery interval for each source using this priority:
1. Source-specific `discovery_interval` in config
2. Global `--discovery-interval` CLI option
3. Default value (86400 seconds = 1 day)

## How It Works

The leader worker maintains separate timers for each content source:

1. **Independent Scheduling**: Each source is checked based on its own interval
2. **Smart Sleep**: The discovery loop sleeps for the minimum interval (capped at 60s)
3. **Per-Source Tracking**: Last discovery time is tracked independently for each source

### Discovery Loop Behavior

```
For each iteration:
  - Check each source's last discovery time
  - If (current_time - last_discovery) >= source_interval:
    - Discover documents from that source
    - Update last discovery time
  - Sleep for min(60, shortest_source_interval)
```

## Recommended Intervals by Source Type

### File Systems
- **Active development directories**: 300-3600 seconds (5 min - 1 hour)
- **Documentation folders**: 3600-21600 seconds (1-6 hours)
- **Archive directories**: 86400+ seconds (daily or less)

### Web Sources
- **News/blog sites**: 3600-21600 seconds (1-6 hours)
- **Documentation sites**: 86400-604800 seconds (daily to weekly)
- **Static sites**: 604800+ seconds (weekly or less)

### API Sources (Confluence, JIRA)
- **Active projects**: 1800-7200 seconds (30 min - 2 hours)
- **Reference spaces**: 21600-86400 seconds (6 hours - daily)
- **Archived content**: 604800+ seconds (weekly or less)

### Cloud Storage (S3, Azure)
- **Upload directories**: 60-600 seconds (1-10 minutes)
- **Processing pipelines**: 300-3600 seconds (5 min - 1 hour)
- **Backup storage**: 86400+ seconds (daily or less)

### Databases
- **Transaction tables**: 300-3600 seconds (5 min - 1 hour)
- **Reference data**: 21600-86400 seconds (6 hours - daily)
- **Historical data**: 604800+ seconds (weekly or less)

## Example Configuration

```yaml
content_sources:
  # Frequent checks for active development
  - name: "src-code"
    type: "file"
    base_path: "./src"
    discovery_interval: 600  # Every 10 minutes

  # Moderate frequency for team documentation
  - name: "confluence"
    type: "confluence"
    base_url: "https://example.atlassian.net"
    discovery_interval: 7200  # Every 2 hours

  # Infrequent checks for stable content
  - name: "archive"
    type: "s3"
    bucket_name: "archive-bucket"
    discovery_interval: 604800  # Weekly
```

## Performance Considerations

### Resource Usage
- Shorter intervals = More API calls and CPU usage
- Consider rate limits for external APIs
- Balance freshness needs with system load

### Optimization Tips
1. **Group similar sources**: Sources with similar intervals will be checked together
2. **Stagger intervals**: Avoid having all sources check at the same moment
3. **Use continuous discovery**: For sources that support it, enable incremental discovery
4. **Monitor and adjust**: Start conservative and decrease intervals as needed

## Monitoring

The worker logs show when each source is discovered:
```
INFO - Discovering documents from source: project-docs (interval: 3600s)
DEBUG - Source confluence discovery interval: 7200s
INFO - Leader discovered 15 new documents from project-docs
```

## CLI Override

You can override all source intervals globally for testing:
```bash
# Force all sources to check every 30 seconds (testing only!)
python -m go_doc_go.cli.worker --discovery-interval 30
```

## Migration from Global Interval

If upgrading from a version with only global intervals:
1. The global interval becomes the default fallback
2. Add source-specific intervals gradually
3. Monitor logs to verify correct intervals are applied