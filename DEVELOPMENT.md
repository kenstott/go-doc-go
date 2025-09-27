# Go-Doc-Go Development Guide

## Development Setup

### Prerequisites

- Python 3.9+
- Git
- Virtual environment tool (venv, conda, etc.)

### Local Development Environment

```bash
# 1. Clone the repository
git clone https://github.com/your-org/go-doc-go.git
cd go-doc-go

# 2. Create virtual environment
python -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate

# 3. Install in development mode
pip install -e ".[dev]"

# 4. Install pre-commit hooks (optional)
pre-commit install
```

## Project Structure

```
go-doc-go/
├── src/
│   └── go_doc_go/
│       ├── __init__.py
│       ├── main.py              # CLI entry point
│       ├── cli/                 # CLI commands
│       │   ├── worker.py        # Document processing
│       │   ├── search.py        # Search interface
│       │   ├── analytics.py     # Analytics interface
│       │   └── status.py        # Status monitoring
│       ├── document_parser/     # Document parsers
│       ├── storage/             # Storage backends
│       ├── embeddings/          # Embedding providers
│       ├── content_source/      # Content sources
│       └── config.py            # Configuration handling
├── tests/                       # Test suite
├── docs/                        # Documentation
├── examples/                    # Example configurations
└── pyproject.toml              # Project metadata
```

## Development Workflow

### 1. Making Changes

```bash
# Create a feature branch
git checkout -b feature/your-feature-name

# Make your changes
# ... edit files ...

# Test your changes locally
python -m go_doc_go worker --config test_config.yaml --max-documents 5
```

### 2. Running Tests

```bash
# Run all tests
pytest

# Run specific test file
pytest tests/test_worker.py

# Run with coverage
pytest --cov=go_doc_go --cov-report=term-missing

# Run specific test
pytest tests/test_worker.py::TestWorker::test_process_document
```

### 3. Code Quality Checks

```bash
# Format code with black
black src/ tests/

# Sort imports
isort src/ tests/

# Type checking
mypy src/

# Linting
flake8 src/ tests/

# All checks at once
make lint  # If Makefile is available
```

### 4. Testing CLI Commands

```bash
# Test worker command
PYTHONPATH=src python -m go_doc_go worker --help
PYTHONPATH=src python -m go_doc_go worker --config test_config.yaml --max-documents 1

# Test search command
PYTHONPATH=src python -m go_doc_go search "test query"

# Test analytics
PYTHONPATH=src python -m go_doc_go analytics --detailed

# Test status monitoring
PYTHONPATH=src python -m go_doc_go status --follow
```

## Adding New Features

### Adding a New CLI Command

1. Create a new file in `src/go_doc_go/cli/`:

```python
# src/go_doc_go/cli/mycommand.py
import click

@click.command()
@click.option('--config', '-c', default='config.yaml', help='Configuration file')
def main(config):
    """My new command description."""
    click.echo(f"Running my command with config: {config}")
    # Your command logic here

if __name__ == '__main__':
    main()
```

2. Register it in `src/go_doc_go/main.py`:

```python
# Add import
from .cli.mycommand import main as mycommand_cmd

# Register command
cli.add_command(mycommand_cmd, name='mycommand')
```

3. Test the new command:

```bash
python -m go_doc_go mycommand --help
```

### Adding a New Document Parser

1. Create parser in `src/go_doc_go/document_parser/`:

```python
# src/go_doc_go/document_parser/myformat.py
from .base import BaseParser

class MyFormatParser(BaseParser):
    def parse(self, content, metadata=None):
        """Parse my format documents."""
        elements = []
        # Your parsing logic here
        return {
            'document': {...},
            'elements': elements,
            'relationships': []
        }
```

2. Register in parser factory:

```python
# src/go_doc_go/document_parser/factory.py
from .myformat import MyFormatParser

PARSERS = {
    # ... existing parsers ...
    'myformat': MyFormatParser,
}
```

### Adding a New Storage Backend

1. Create storage backend in `src/go_doc_go/storage/`:

```python
# src/go_doc_go/storage/mybackend.py
from .base import BaseStorage

class MyBackendStorage(BaseStorage):
    def __init__(self, config):
        self.config = config
        # Initialize connection

    def store_document(self, doc):
        # Store document logic
        pass

    def search(self, query, **kwargs):
        # Search logic
        pass
```

2. Register in storage factory:

```python
# src/go_doc_go/storage/__init__.py
from .mybackend import MyBackendStorage

STORAGE_BACKENDS = {
    # ... existing backends ...
    'mybackend': MyBackendStorage,
}
```

## Testing Guidelines

### Unit Tests

```python
# tests/test_myfeature.py
import pytest
from go_doc_go.myfeature import MyFeature

class TestMyFeature:
    def test_basic_functionality(self):
        feature = MyFeature()
        result = feature.process("input")
        assert result == "expected"

    def test_edge_case(self):
        feature = MyFeature()
        with pytest.raises(ValueError):
            feature.process(None)
```

### Integration Tests

```python
# tests/integration/test_pipeline.py
import tempfile
from go_doc_go.cli.worker import process_documents

def test_end_to_end_processing():
    with tempfile.TemporaryDirectory() as tmpdir:
        config = {
            'storage': {'backend': 'sqlite', 'path': f'{tmpdir}/test.db'},
            'content_sources': [{'type': 'file', 'base_path': './test_data'}]
        }
        result = process_documents(config, max_documents=1)
        assert result['processed'] == 1
```

### CLI Tests

```python
# tests/test_cli.py
from click.testing import CliRunner
from go_doc_go.main import cli

def test_worker_command():
    runner = CliRunner()
    result = runner.invoke(cli, ['worker', '--help'])
    assert result.exit_code == 0
    assert 'worker' in result.output
```

## Debugging

### Enable Debug Logging

```python
# In your code
import logging
logging.basicConfig(level=logging.DEBUG)

# Or via CLI
python -m go_doc_go worker --log-level debug
```

### Use Python Debugger

```python
# Add breakpoint in code
import pdb; pdb.set_trace()

# Or use breakpoint() in Python 3.7+
breakpoint()
```

### Profile Performance

```bash
# Profile a command
python -m cProfile -o profile.stats -m go_doc_go worker --max-documents 100

# Analyze profile
python -m pstats profile.stats
```

## Configuration for Development

### Test Configuration

Create a `test_config.yaml` for development:

```yaml
# test_config.yaml
storage:
  backend: sqlite
  path: ./test_data/test.db

content_sources:
  - name: test_docs
    type: file
    base_path: ./test_data/documents

processing:
  batch_size: 10
  max_workers: 2

logging:
  level: DEBUG
```

### Environment Variables

Create a `.env` file for development:

```bash
# .env
GO_DOC_GO_CONFIG_PATH=./test_config.yaml
GO_DOC_GO_LOG_LEVEL=DEBUG
PYTHONPATH=src
```

## Common Development Tasks

### Update Dependencies

```bash
# Update requirements
pip-compile pyproject.toml -o requirements.txt

# Install updated dependencies
pip install -r requirements.txt
```

### Build Documentation

```bash
# Build docs with Sphinx (if configured)
cd docs
make html

# Or with mkdocs
mkdocs build
```

### Create Distribution

```bash
# Build package
python -m build

# Check distribution
twine check dist/*

# Upload to PyPI (requires credentials)
twine upload dist/*
```

## Git Workflow

### Commit Messages

Follow conventional commits:

```bash
feat: add new parser for XML documents
fix: resolve memory leak in embedding generation
docs: update configuration examples
test: add tests for worker command
refactor: simplify storage backend interface
```

### Pull Request Process

1. Create feature branch
2. Make changes with tests
3. Run tests and linting
4. Push branch
5. Create pull request
6. Address review comments
7. Merge after approval

## Troubleshooting Development Issues

### Import Errors

```bash
# Ensure PYTHONPATH is set
export PYTHONPATH=src

# Or install in development mode
pip install -e .
```

### Database Lock Issues (SQLite)

```python
# Use WAL mode for concurrent access
storage:
  backend: sqlite
  journal_mode: WAL
```

### Memory Issues During Testing

```bash
# Limit test scope
pytest -m "not memory_intensive"

# Or increase memory
ulimit -v unlimited
```

## Performance Optimization

### Profiling

```python
import cProfile
import pstats

profiler = cProfile.Profile()
profiler.enable()

# Your code here
process_documents(config)

profiler.disable()
stats = pstats.Stats(profiler)
stats.sort_stats('cumulative')
stats.print_stats(10)
```

### Memory Profiling

```bash
# Install memory profiler
pip install memory_profiler

# Profile memory usage
python -m memory_profiler your_script.py
```

## Release Process

1. Update version in `pyproject.toml`
2. Update CHANGELOG.md
3. Run full test suite
4. Create git tag: `git tag v1.2.3`
5. Push tag: `git push origin v1.2.3`
6. Build and upload to PyPI

## Contributing Guidelines

1. Fork the repository
2. Create feature branch
3. Write tests for new features
4. Ensure all tests pass
5. Update documentation
6. Submit pull request

## Resources

- [Click Documentation](https://click.palletsprojects.com/) - CLI framework
- [SQLAlchemy Docs](https://www.sqlalchemy.org/) - Database toolkit
- [pytest Documentation](https://docs.pytest.org/) - Testing framework
- [Black](https://black.readthedocs.io/) - Code formatter
- [mypy](https://mypy.readthedocs.io/) - Type checker