# Discovery Configuration Migration Guide

## Overview

The discovery/crawling configuration has been restructured to support both hyperlink crawling and code dependency crawling in a unified way.

## Breaking Changes

### Old Configuration Format (Deprecated)

```toml
[[content_sources]]
name = "my_source"
type = "web"
base_url = "https://example.com"
url_list = ["https://example.com/start"]

# OLD: Fields at root level
max_link_depth = 2
include_patterns = ["https://example.com/**"]
exclude_patterns = ["/archive/"]
```

### New Configuration Format

```toml
[[content_sources]]
name = "my_source"
type = "web"
base_url = "https://example.com"
url_list = ["https://example.com/start"]

# NEW: Nested under [content_sources.discovery]
[content_sources.discovery]
enabled = true
max_depth = 2
include_patterns = ["https://example.com/**"]
exclude_patterns = ["/archive/"]

[content_sources.discovery.hyperlinks]
enabled = true

[content_sources.discovery.code_dependencies]
enabled = false  # New feature
```

## Migration Steps

### Step 1: Update Content Sources

For each `[[content_sources]]` block in your configuration:

**Before**:
```toml
[[content_sources]]
name = "docs"
type = "web"
max_link_depth = 3
include_patterns = ["^https://docs.example.com/"]
exclude_patterns = ["/old/", "/archive/"]
```

**After**:
```toml
[[content_sources]]
name = "docs"
type = "web"

[content_sources.discovery]
enabled = true
max_depth = 3
include_patterns = ["^https://docs.example.com/"]
exclude_patterns = ["/old/", "/archive/"]

[content_sources.discovery.hyperlinks]
enabled = true

[content_sources.discovery.code_dependencies]
enabled = false
```

### Step 2: Field Mapping

| Old Field | New Field | Notes |
|-----------|-----------|-------|
| `max_link_depth` | `discovery.max_depth` | Now applies to all discovery types |
| `include_patterns` | `discovery.include_patterns` | Now applies to all discovery types |
| `exclude_patterns` | `discovery.exclude_patterns` | Now applies to all discovery types |
| N/A | `discovery.enabled` | New master switch |
| N/A | `discovery.hyperlinks.enabled` | New subsection |
| N/A | `discovery.code_dependencies.*` | New feature |

### Step 3: Enable/Disable Discovery Types

**For web sources** (HTML/Markdown):
```toml
[content_sources.discovery.hyperlinks]
enabled = true

[content_sources.discovery.code_dependencies]
enabled = false  # Not applicable
```

**For code sources** (Go/JS/Python/Java):
```toml
[content_sources.discovery.hyperlinks]
enabled = false  # Not applicable

[content_sources.discovery.code_dependencies]
enabled = true
follow_stdlib = false
follow_local = true
follow_external = false
```

## New Features

### Code Dependency Crawling

The new configuration adds support for automatically discovering and crawling code dependencies (imports):

```toml
[content_sources.discovery.code_dependencies]
enabled = true              # Enable code dependency crawling
max_depth = 3               # Override global depth for code only (optional)
follow_stdlib = false       # Don't follow standard library (fmt, os, etc.)
follow_local = true         # Follow same-project imports
follow_external = false     # Don't follow third-party packages
```

**Supported languages**:
- Go (`.go`)
- JavaScript/TypeScript (`.js`, `.ts`, `.jsx`, `.tsx`)
- Python (`.py`)
- Java (`.java`)

### Hierarchical Configuration

You can now override `max_depth` at different levels:

```toml
[content_sources.discovery]
max_depth = 2  # Global default

[content_sources.discovery.hyperlinks]
max_depth = 1  # Override for hyperlinks only

[content_sources.discovery.code_dependencies]
max_depth = 3  # Override for code dependencies only
```

## Example Migrations

### Example 1: Wikipedia Crawler

**Before**:
```toml
[[content_sources]]
name = "wikipedia"
type = "web"
base_url = "https://en.wikipedia.org"
url_list = ["https://en.wikipedia.org/wiki/Medicine"]
include_patterns = ["^https://en.wikipedia.org/wiki/"]
exclude_patterns = ["/Special:", "/Talk:"]
max_link_depth = 1
```

**After**:
```toml
[[content_sources]]
name = "wikipedia"
type = "web"
base_url = "https://en.wikipedia.org"
url_list = ["https://en.wikipedia.org/wiki/Medicine"]

[content_sources.discovery]
enabled = true
max_depth = 1
include_patterns = ["^https://en.wikipedia.org/wiki/"]
exclude_patterns = ["/Special:", "/Talk:"]

[content_sources.discovery.hyperlinks]
enabled = true

[content_sources.discovery.code_dependencies]
enabled = false
```

### Example 2: File Documentation

**Before**:
```toml
[[content_sources]]
name = "docs"
type = "file"
base_path = "./docs"
file_pattern = "**/*.md"
max_link_depth = 2
```

**After**:
```toml
[[content_sources]]
name = "docs"
type = "file"
base_path = "./docs"
file_pattern = "**/*.md"

[content_sources.discovery]
enabled = true
max_depth = 2
include_patterns = []
exclude_patterns = []

[content_sources.discovery.hyperlinks]
enabled = true

[content_sources.discovery.code_dependencies]
enabled = false
```

### Example 3: Go Codebase (New Feature)

```toml
[[content_sources]]
name = "go_codebase"
type = "file"
base_path = "./go/internal"
file_pattern = "**/*.go"

[content_sources.discovery]
enabled = true
max_depth = 3
include_patterns = ["github.com/myorg/**"]  # Only our modules
exclude_patterns = ["**/*_test.go", "**/vendor/**"]

[content_sources.discovery.hyperlinks]
enabled = false  # Not applicable

[content_sources.discovery.code_dependencies]
enabled = true
follow_stdlib = false   # Skip Go stdlib (fmt, os, etc.)
follow_local = true     # Follow our internal packages
follow_external = false # Skip external dependencies
```

## Backward Compatibility

**IMPORTANT**: The old configuration format (`max_link_depth`, `include_patterns`, `exclude_patterns` at root level) is **NOT** supported.

You **MUST** update your configuration files to use the new `[content_sources.discovery]` structure.

## Validation

After migration, verify your configuration:

```bash
# Validate configuration loads without errors
bin/goworker --config config.toml --max-documents 0

# Check for deprecation warnings in logs
grep -i "deprecated\|warning" logs/worker.log
```

## Testing

Test your migrated configuration with a small document set:

```toml
# Create a test config with max_documents limit
[[content_sources]]
name = "test_source"
# ... your configuration ...

[processing]
max_workers = 1  # Single worker for easier debugging
```

Run worker:
```bash
bin/goworker --config test_config.toml --max-documents 10
```

## Troubleshooting

### Issue: Configuration not loading

**Error**: `failed to parse TOML` or `invalid configuration`

**Solution**: Ensure all `[[content_sources]]` blocks have the new nested structure. Check for missing brackets `[content_sources.discovery]`.

### Issue: Discovery not working

**Symptom**: No discovered documents are queued

**Check**:
1. `discovery.enabled = true`
2. `discovery.hyperlinks.enabled = true` OR `discovery.code_dependencies.enabled = true`
3. `max_depth > 0`
4. `include_patterns` not too restrictive

**Debug**:
```bash
# Enable debug logging
[logging]
level = "DEBUG"

# Check logs for discovery messages
grep "Queued.*discovered" logs/worker.log
```

### Issue: Too many documents discovered

**Symptom**: System crawls too much (entire stdlib, all third-party packages)

**Solution**: Adjust filters:
```toml
[content_sources.discovery]
max_depth = 1  # Reduce depth

[content_sources.discovery.code_dependencies]
follow_stdlib = false   # Disable stdlib crawling
follow_external = false # Disable external package crawling
include_patterns = ["github.com/myorg/**"]  # Whitelist only your code
```

## Getting Help

- Check CLAUDE.md for full configuration documentation
- Review example configs in `tests/test_configs/`
- File issues at https://github.com/anthropics/claude-code/issues
