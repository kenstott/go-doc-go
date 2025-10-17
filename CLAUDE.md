1# Aperio Project Guidelines

## Project Overview
Aperio is a grounded reasoning platform designed to extract structured information from various document formats (PDF, DOCX, XLSX, JSON, CSV, HTML, Markdown, etc.) and store it in a queryable format with relationship tracking.

## Python Best Practices

### Code Organization
1. **Single Responsibility Principle**: Each class/function should have one clear purpose
2. **DRY (Don't Repeat Yourself)**: Extract common functionality into reusable functions
3. **Explicit is better than implicit**: Use clear, descriptive names
4. **Composition over inheritance**: Prefer composition and mixins over deep inheritance hierarchies
5. **Default values**: You are prohibited from providing default values under most circumstances, use required values and error out if not received.
6. **Fallback code paths**: Your prohibited from creating fallbacks with human consent, particularly as a method to support legacy.
7. **Legacy**: There is no such concept as legacy. DO NOT use the word legacy. DO NOT introduce legacy code.

### Naming Conventions
- **Classes**: PascalCase (e.g., `DocumentParser`, `PdfParser`)
- **Functions/Methods**: snake_case (e.g., `parse_document`, `extract_text`)
- **Constants**: UPPER_SNAKE_CASE (e.g., `MAX_FILE_SIZE`, `DEFAULT_TIMEOUT`)
- **Private methods**: Leading underscore (e.g., `_internal_method`)

### Type Hints
Always use type hints for better code documentation and IDE support:
```python
from typing import Dict, List, Optional, Tuple, Any

def parse_document(content: Dict[str, Any], config: Optional[Dict] = None) -> Tuple[List[Dict], List[Dict]]:
    """Parse document and return elements and relationships."""
    pass
```

### Error Handling
- Use specific exceptions rather than catching broad `Exception`
- Create custom exceptions for domain-specific errors
- Always log errors with appropriate context
```python
class ParserError(Exception):
    """Base exception for parser errors."""
    pass

class InvalidDocumentError(ParserError):
    """Raised when document format is invalid."""
    pass
```

### DRY Principles Implementation

#### Common Base Classes
```python
# Base parser with common functionality
class DocumentParser(ABC):
    def __init__(self, config: Optional[Dict] = None):
        self.config = config or {}
        self.max_content_preview = self.config.get("max_content_preview", 100)
    
    def _generate_id(self, prefix: str) -> str:
        """Generate unique ID - common for all parsers."""
        return f"{prefix}{uuid.uuid4().hex[:8]}"
    
    @abstractmethod
    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        """Each parser implements specific parsing logic."""
        pass
```

#### Helper Functions
Extract repeated logic into helper functions:
```python
# utils.py
def truncate_content(text: str, max_length: int = 100) -> str:
    """Truncate content for preview - used across all parsers."""
    if len(text) <= max_length:
        return text
    return text[:max_length - 3] + "..."

def validate_element_type(element_type: str) -> bool:
    """Validate element type against ElementType enum."""
    return element_type in [e.value for e in ElementType]
```

#### Configuration Management
```python
class ParserConfig:
    """Centralized configuration management."""
    
    DEFAULTS = {
        "max_content_preview": 100,
        "extract_metadata": True,
        "extract_relationships": True,
        "max_depth": 10
    }
    
    @classmethod
    def merge_with_defaults(cls, config: Optional[Dict] = None) -> Dict:
        """Merge user config with defaults."""
        return {**cls.DEFAULTS, **(config or {})}
```

## Go Development Standards

### Binary Output Location
All compiled Go binaries MUST be output to `<project-dir>/bin/` directory.

**Requirements:**
- Use the `-o` flag to specify output location when building Go binaries
- Keep all project binaries centralized in the `bin/` directory
- This ensures binaries are properly gitignored and organized

**Examples:**
```bash
# Building the worker binary
go build -o bin/goworker ./cmd/worker

# Building the ontology CLI
go build -o bin/ontology ./cmd/ontology

# Building any command
go build -o bin/<binary-name> ./cmd/<command>/
```

**Benefits:**
- Centralized location for all binaries (easier to find and manage)
- Consistent with standard Go project layout conventions
- Properly gitignored (bin/ directory should be in .gitignore)
- Prevents binaries from being scattered across cmd/ directories
- Simplifies cleanup (rm -rf bin/)

## Testing Guidelines

### Test Organization
```
tests/
├── unit/           # Unit tests - isolated component tests
├── integration/    # Integration tests - component interactions  
├── fixtures/       # Test data and fixtures
└── conftest.py     # Pytest configuration
```

### Basic Test Structure
```python
import pytest

class TestDocumentParser:
    def test_parse_returns_valid_elements(self):
        """Test that parser returns valid element types."""
        parser = DocumentParser()
        result = parser.parse(sample_content)
        
        for element in result["elements"]:
            element_type = element["element_type"] 
            assert element_type in [e.value for e in ElementType]
```

### Test Validation Helpers
```python
def assert_valid_parse_result(result: Dict[str, Any]):
    """Validate parser output structure."""
    assert "document" in result
    assert "elements" in result
    assert "relationships" in result
    
    for element in result["elements"]:
        assert "element_id" in element
        assert "element_type" in element
        assert "content_preview" in element
```

## Common Patterns and Solutions

### Factory Pattern for Parser Creation
```python
def create_parser(doc_type: str, config: Optional[Dict] = None) -> DocumentParser:
    """Factory function to create appropriate parser."""
    parsers = {
        "pdf": PdfParser,
        "docx": DocxParser,
        "xlsx": XlsxParser,
        "csv": CsvParser,
        "json": JSONParser,
        "xml": XmlParser,
        "html": HtmlParser,
        "markdown": MarkdownParser,
        "text": TextParser
    }
    
    parser_class = parsers.get(doc_type)
    if not parser_class:
        raise ValueError(f"Unknown document type: {doc_type}")
    
    return parser_class(config)
```

### Context Managers for Resource Handling
```python
class TempFileHandler:
    """Context manager for temporary file handling."""
    
    def __init__(self, content: bytes, suffix: str = ""):
        self.content = content
        self.suffix = suffix
        self.temp_file = None
    
    def __enter__(self) -> str:
        self.temp_file = tempfile.NamedTemporaryFile(
            suffix=self.suffix, 
            delete=False
        )
        self.temp_file.write(self.content)
        self.temp_file.close()
        return self.temp_file.name
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        if self.temp_file and os.path.exists(self.temp_file.name):
            os.unlink(self.temp_file.name)
```

### Validation Decorators
```python
def validate_input(schema: Dict[str, type]):
    """Decorator to validate input parameters."""
    def decorator(func):
        @functools.wraps(func)
        def wrapper(self, content: Dict[str, Any], *args, **kwargs):
            # Validate required fields
            for field, expected_type in schema.items():
                if field not in content:
                    raise ValueError(f"Missing required field: {field}")
                if not isinstance(content[field], expected_type):
                    raise TypeError(
                        f"Field {field} must be {expected_type.__name__}, "
                        f"got {type(content[field]).__name__}"
                    )
            return func(self, content, *args, **kwargs)
        return wrapper
    return decorator

class SomeParser(DocumentParser):
    @validate_input({"id": str, "content": (str, bytes), "metadata": dict})
    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        # Input is validated before parsing
        pass
```

## Development Workflow

### Pre-Commit Verification Checklist - MANDATORY
Before any git commit, ALL of the following MUST pass:

```bash
#!/bin/bash
# pre-commit-checklist.sh - run before any git commit

echo "Pre-Commit Checklist - MANDATORY"
echo "=================================="

# 1. All modified code builds without errors
python -m py_compile src/go_doc_go/**/*.py || { echo "✗ Compilation errors found"; exit 1; }
echo "✓ Code compiles without errors"

# 2. All related tests pass (provide command + output)
pytest -v || { echo "✗ Tests failed"; exit 1; }
echo "✓ All tests pass"

# 3. No debugging artifacts left in code
if grep -r "print(" src/go_doc_go/ --include="*.py" | grep -v "__main__"; then
    echo "✗ Debug print statements found - remove before commit"
    exit 1
fi
echo "✓ No debugging artifacts"

# 4. Code quality checks
flake8 src/ tests/ || { echo "✗ Linting errors found"; exit 1; }
echo "✓ Linting passed"

mypy src/ || { echo "✗ Type checking errors found"; exit 1; }
echo "✓ Type checking passed"

black --check src/ tests/ || { echo "✗ Code formatting required"; exit 1; }
echo "✓ Code formatting verified"

# 5. Coverage requirements met
pytest --cov=src/go_doc_go --cov-report=term --cov-fail-under=70 || { echo "✗ Coverage below 70%"; exit 1; }
echo "✓ Coverage requirements met"

# 6. Performance benchmarks (if performance-critical code changed)
if git diff --name-only HEAD^ | grep -E "(queue|parser)" > /dev/null; then
    pytest -m performance || { echo "✗ Performance tests failed"; exit 1; }
    echo "✓ Performance benchmarks met"
fi

echo "All checks passed - ready to commit"
```

### Before Committing (Legacy)
1. Run unit tests: `pytest -m unit`
2. Check coverage: `pytest --cov=src/go_doc_go --cov-report=term-missing`
3. Run linter: `flake8 src/ tests/`
4. Run type checker: `mypy src/`
5. Format code: `black src/ tests/`

### Coverage Goals
- **Overall**: Minimum 70% coverage
- **Critical parsers** (PDF, DOCX, XLSX): Minimum 80% coverage
- **Utility modules**: Minimum 90% coverage
- **New code**: Must include tests before merging

### Performance Benchmarks and SLAs

#### Document Processing SLAs
- Standard document (< 10MB): Parse in < 1 second
- Large document (< 100MB): Parse in < 10 seconds  
- Memory usage: < 5x document size
- Concurrent parsing: Support 10 simultaneous parsers

#### Work Queue System SLAs
- **Document claiming latency**: < 10ms per document
- **Sustained throughput**: > 1000 docs/second with 10 concurrent workers
- **Memory usage per worker**: < 100MB base memory
- **Maximum concurrent workers**: 50 workers supported
- **Claim timeout**: 5 minutes (300 seconds)
- **Heartbeat interval**: 30 seconds

## Debugging and Troubleshooting

### Logging Best Practices
```python
import logging

logger = logging.getLogger(__name__)

class DocumentParser:
    def parse(self, content: Dict[str, Any]) -> Dict[str, Any]:
        logger.debug(f"Starting parse for document: {content.get('id')}")
        
        try:
            # Parsing logic
            result = self._do_parse(content)
            logger.info(f"Successfully parsed {len(result['elements'])} elements")
            return result
            
        except Exception as e:
            logger.error(
                f"Failed to parse document {content.get('id')}: {str(e)}", 
                exc_info=True
            )
            raise
```

### Debug Helpers
```python
def debug_element_structure(elements: List[Dict], max_depth: int = 3):
    """Print element hierarchy for debugging."""
    def print_element(elem, depth=0):
        if depth > max_depth:
            return
        indent = "  " * depth
        print(f"{indent}{elem['element_type']}: {elem['element_id'][:8]}... "
              f"[{elem['content_preview'][:30]}...]")
        
        # Print children
        children = [e for e in elements if e.get('parent_id') == elem['element_id']]
        for child in children:
            print_element(child, depth + 1)
    
    # Start with root elements
    roots = [e for e in elements if not e.get('parent_id')]
    for root in roots:
        print_element(root)
```

## Key Design Decisions

1. **Element Types**: All parsers MUST use values from the `ElementType` enum
2. **Relationship Types**: All relationships MUST use values from the `RelationshipType` enum
3. **Content Previews**: Limited to 100 characters by default for performance
4. **ID Generation**: UUIDs with meaningful prefixes for debugging
5. **Error Handling**: Fail fast with clear error messages, log all errors
6. **Memory Management**: Stream large files, use generators where possible
7. **Extensibility**: New parsers extend `DocumentParser` base class
8. **Work Queue Coordination**: Use config hash as run_id for automatic worker coordination
9. **Atomic Operations**: Use PostgreSQL row-level locking for atomic document claiming
10. **Distributed Processing**: Pull-based work queue pattern with identical workers
11. **Design Integrity Over Backward Compatibility**: Prefer breaking changes to maintain clean design and correct implementation rather than accumulating technical debt through backward compatibility hacks. When the correct design requires breaking changes, make them cleanly and document migration paths.

## Design Integrity Principle

### Breaking Changes Are Preferred When Design Is Wrong

**Philosophy**: Clean design and correct implementation take precedence over backward compatibility. Technical debt from compatibility hacks compounds over time and makes the codebase harder to maintain.

**When to Break Compatibility**:
- The current implementation violates design principles
- Field names are misleading or incorrect
- The API encourages incorrect usage
- Maintaining compatibility would require ugly hacks
- The correct fix is simpler than the compatibility layer

**Examples of Good Breaking Changes**:
```python
# BAD: Maintaining compatibility with poor design
def _validate_queries(self):
    # Supporting both old and new field names
    if "id_columns" not in query and "doc_id_columns" not in query:
        raise ValueError("Missing id_columns or doc_id_columns")
    # Normalize internally (adds complexity)
    if "doc_id_columns" in query and "id_columns" not in query:
        query["id_columns"] = query["doc_id_columns"]

# GOOD: Fix the design properly
def _validate_queries(self):
    # Breaking change but cleaner
    if "id_columns" not in query:
        raise ValueError("Missing required 'id_columns' field")
```

**Migration Strategy**:
1. Make the breaking change cleanly
2. Document the change clearly in release notes
3. Provide a migration script if needed
4. Update all tests to use the new design
5. Bump version number appropriately (major version for breaking changes)

**Anti-Pattern to Avoid**:
```python
# DON'T DO THIS: Accumulating compatibility cruft
field = config.get("new_name", config.get("old_name", config.get("legacy_name", config.get("ancient_name"))))
```






