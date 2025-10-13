# Configuration File Migration: YAML to TOML

## Status

The Go worker binary uses **TOML format** for configuration files. The main configuration files have been converted:

### Converted Files

1. ✅ `config.toml` - Main configuration file (converted from config.yaml)
2. ✅ `docs/examples/dev-backends.toml` - Development backend configurations (moved from config/)
3. ✅ `docs/examples/per_source_intervals.toml` - Per-source interval examples (moved from config_examples/)

### Removed Python-Specific Configs

The following Python-specific YAML configurations were removed as they are not compatible with the Go worker:

- ~~`configs/sec-html-store.yaml`~~ - Used Python-only `fastembed` provider
- ~~`configs/sec-neo4j-export.yaml`~~ - Used Python-only `fastembed` provider
- ~~`configs/sec-semantic-insider-extraction.yaml`~~ - Used Python-only `fastembed` provider

**Note**: If SEC filing processing is needed, create new TOML configs using `provider: onnx` for the Go worker.

## TOML vs YAML Key Differences

### Arrays of Tables (Content Sources)

**YAML:**
```yaml
content_sources:
  - name: "wikipedia"
    type: "web"
    base_url: "https://en.wikipedia.org"
```

**TOML:**
```toml
[[content_sources]]
name = "wikipedia"
type = "web"
base_url = "https://en.wikipedia.org"
```

### Nested Tables

**YAML:**
```yaml
processing:
  job_control:
    backend: "sqlite"
    path: "./job_queue.db"
```

**TOML:**
```toml
[processing.job_control]
backend = "sqlite"
path = "./job_queue.db"
```

### Headers/Nested Maps

**YAML:**
```yaml
headers:
  User-Agent: "Go-Doc-Go/1.0"
```

**TOML:**
```toml
[content_sources.headers]
User-Agent = "Go-Doc-Go/1.0"
```

## Usage

The worker binary expects TOML format:

```bash
bin/goworker --config config.toml --max-documents 10
```

## Converting Additional YAML Files

To convert a YAML file to TOML manually:

1. Array items (`- name: value`) become table arrays (`[[table]]` / `name = "value"`)
2. Nested maps use dot notation (`[parent.child]`)
3. Strings should be quoted
4. Booleans are lowercase (`true`, `false`)
5. Numbers don't need quotes
6. Multi-line strings use triple quotes (`"""..."""`)

## Future Work

- Consider adding YAML support to the worker if backward compatibility is needed
- Convert remaining SEC config files when they're actively used
- Add config validation to catch format errors early
