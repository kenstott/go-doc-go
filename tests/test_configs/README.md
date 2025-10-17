# Test Configurations

This directory contains test configuration files for Go-Doc-Go testing and development.

## Files

### wikipedia_medical_mining.toml
Test configuration that processes 2 Wikipedia pages with crawl depth of 1:
- **Medicine**: https://en.wikipedia.org/wiki/Medicine
- **Mining**: https://en.wikipedia.org/wiki/Mining

**Features:**
- Crawl depth: 1 (follows links one level deep)
- Embedding enabled (ONNX, all-MiniLM-L6-v2)
- Ontology extraction disabled
- Cross-document semantic relationships disabled
- Outputs to `tests/test_output/`

**Usage:**
```bash
# Build the worker
go build -o bin/goworker ./cmd/worker

# Run with this config (process 10 documents max)
bin/goworker --config tests/test_configs/wikipedia_medical_mining.toml --max-documents 10
```

## Directory Structure

```
tests/
├── test_configs/      # Test configurations (THIS DIRECTORY - version controlled)
├── fixtures/          # Static test data (version controlled)
├── test_assets/       # Downloaded/generated files (gitignored, disposable)
└── test_output/       # Test results and logs (gitignored, disposable)
```

## Notes

- All test configs should output to `tests/test_output/` or `tests/test_assets/`
- Never use `/tmp` for test files - they are cleared on system restart
- Test configs in this directory are version controlled
- Test output and assets are gitignored and safe to delete