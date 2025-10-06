# Go Worker CLI Testing

This document describes the pure Go test suite for the worker CLI, converted from the Python integration tests.

## Overview

The Go test suite (`worker_test.go`) provides comprehensive testing of the worker binary without requiring Python or the `USE_GO_MODULES` environment variable. All tests are written in pure Go using the standard `testing` package.

## Test Structure

### Test Helper

The `TestHelper` struct provides utilities for:
- Setting up isolated test directories
- Creating test-specific configurations
- Running the worker binary synchronously or asynchronously
- Collecting and analyzing output

### Test Cases

#### 1. `TestWorkerBinaryExists`
Verifies that the worker binary is compiled and available.

```bash
go test -run TestWorkerBinaryExists
```

#### 2. `TestWorkerConfigValidation`
Tests that the worker can validate and load configuration correctly.

```bash
go test -run TestWorkerConfigValidation
```

#### 3. `TestWorkerProcessDocuments`
Tests end-to-end document processing with analytics output verification.

```bash
go test -run TestWorkerProcessDocuments
```

#### 4. `TestWorkerLeaderElection`
Verifies worker leader election mechanism.

```bash
go test -run TestWorkerLeaderElection
```

#### 5. `TestMultipleWorkersCoordination`
Tests multiple workers coordinating via the job control database.

```bash
go test -run TestMultipleWorkersCoordination
```

#### 6. `TestWorkerMaxDocumentsLimit`
Verifies that workers respect the `--max-documents` limit.

```bash
go test -run TestWorkerMaxDocumentsLimit
```

#### 7. `TestJobControlDatabaseCreation`
Tests job control database creation and initialization.

```bash
go test -run TestJobControlDatabaseCreation
```

#### 8. `TestAnalyticsOutputCreation`
Verifies analytics output (Parquet files) generation.

```bash
go test -run TestAnalyticsOutputCreation
```

## Running Tests

### Build Worker Binary

First, ensure the worker binary is compiled:

```bash
cd go
go build -o ../bin/goworker ./cmd/worker
```

### Run All Tests

```bash
cd go/cmd/worker
go test -v -timeout 5m
```

### Run Specific Test

```bash
go test -v -run TestWorkerProcessDocuments -timeout 2m
```

### Run with Coverage

```bash
go test -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Configuration

Tests use isolated configurations based on `tests/config.sqlite.yaml`:

- **Test Output Directory**: `tests/test_output/worker_cli_test_go/`
- **Job Control DB**: `job_queue-go.db`
- **Analytics Output**: `analytics-output-go/`
- **Source Documents**: `tests/assets/`

Each test creates its own isolated configuration to prevent interference.

## Test Output

After tests run, output is preserved for examination:

```
tests/test_output/worker_cli_test_go/
├── job_queue-go.db              # Job control database
├── analytics-output-go/         # Parquet analytics files
│   ├── documents/
│   ├── elements/
│   ├── relationships/
│   └── embeddings/
├── logs/
│   └── worker-go.log            # Worker logs
└── worker_config.yaml           # Test configuration
```

## Key Differences from Python Tests

### Advantages of Go Tests

1. **No External Dependencies**: Pure Go, no Python runtime required
2. **Type Safety**: Compile-time checking of configuration structures
3. **Better Concurrency**: Native goroutines for parallel worker testing
4. **Faster Execution**: Native binary testing, no subprocess overhead
5. **Integrated Testing**: Same language as implementation

### Test Patterns

#### Configuration Management
```go
config := h.CreateIsolatedConfig(testDir, func(c *TestConfig) {
    c.Processing.MaxWorkers = 10
    c.Analytics.Enabled = true
})
```

#### Timeout Handling
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

stdout, stderr, err := h.RunWorker(ctx, configPath, "--max-documents", "5")
```

#### Parallel Worker Testing
```go
var wg sync.WaitGroup
for i := 0; i < 3; i++ {
    wg.Add(1)
    go func(workerNum int) {
        defer wg.Done()
        // Run worker
    }(i)
}
wg.Wait()
```

## Extending Tests

### Adding New Tests

1. Create test function with `Test` prefix:
```go
func TestMyNewFeature(t *testing.T) {
    h := NewTestHelper(t)
    testDir := h.SetupTestDir()
    configPath := h.CreateIsolatedConfig(testDir, nil)

    // Test logic here
}
```

2. Use helper methods for common operations:
- `h.RunWorker()` - Run worker synchronously
- `h.RunWorkerAsync()` - Run worker in background
- `ContainsAny()` - Check output for indicators

3. Add cleanup via `t.Cleanup()` if needed

### Custom Configuration

Modify configuration for specific tests:

```go
configPath := h.CreateIsolatedConfig(testDir, func(config *TestConfig) {
    config.Processing.JobControl.ClaimTimeout = 600
    config.Analytics.Enabled = false
    config.Logging.Level = "DEBUG"
})
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Run Go Worker Tests
  run: |
    cd go
    go build -o ../bin/goworker ./cmd/worker
    cd cmd/worker
    go test -v -timeout 10m -coverprofile=coverage.out
```

### Test Tags

Use build tags for different test categories:

```go
//go:build integration

func TestWorkerIntegration(t *testing.T) {
    // Integration test
}
```

Run with:
```bash
go test -tags=integration
```

## Troubleshooting

### Binary Not Found

```
Worker binary not found at .../bin/goworker
```

**Solution**: Build the worker first:
```bash
cd go && go build -o ../bin/goworker ./cmd/worker
```

### Test Timeout

```
panic: test timed out after 5m0s
```

**Solution**: Increase timeout:
```bash
go test -timeout 10m
```

### Permission Denied

```
permission denied: .../job_queue-go.db
```

**Solution**: Clean test output directory:
```bash
rm -rf tests/test_output/worker_cli_test_go
```

## Performance Benchmarks

Run performance benchmarks:

```bash
go test -bench=. -benchmem
```

Example benchmark:

```go
func BenchmarkWorkerProcessing(b *testing.B) {
    h := NewTestHelper(&testing.T{})
    testDir := h.SetupTestDir()
    configPath := h.CreateIsolatedConfig(testDir, nil)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        h.RunWorker(ctx, configPath, "--max-documents", "1")
        cancel()
    }
}
```

## Migration from Python Tests

When converting Python tests to Go:

1. **Replace subprocess calls** with `h.RunWorker()`
2. **Replace timeout decorators** with `context.WithTimeout()`
3. **Replace pytest fixtures** with `TestHelper` methods
4. **Replace pytest.mark** with Go build tags
5. **Replace assert statements** with `t.Fatalf()` or `t.Errorf()`

Example conversion:

**Python:**
```python
def test_worker_process(isolated_config):
    cmd = ["python", "-m", "go_doc_go.cli.worker",
           "--config", str(isolated_config),
           "--max-documents", "5"]

    result = subprocess.run(cmd, timeout=30)
    assert result.returncode == 0
```

**Go:**
```go
func TestWorkerProcess(t *testing.T) {
    h := NewTestHelper(t)
    testDir := h.SetupTestDir()
    configPath := h.CreateIsolatedConfig(testDir, nil)

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    stdout, stderr, err := h.RunWorker(ctx, configPath, "--max-documents", "5")
    if err != nil {
        t.Fatalf("Worker failed: %v", err)
    }
}
```

## Future Enhancements

- [ ] Add table-driven tests for multiple scenarios
- [ ] Add benchmarks for performance regression testing
- [ ] Add test coverage reporting
- [ ] Add integration with Go fuzzing
- [ ] Add test parallelization with `t.Parallel()`
- [ ] Add test fixtures for common document types
- [ ] Add helper for Parquet file validation
