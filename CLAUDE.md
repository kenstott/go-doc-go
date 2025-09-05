# Doculyzer Project Guidelines

## Project Overview
Doculyzer is a comprehensive document parsing and analysis system designed to extract structured information from various document formats (PDF, DOCX, XLSX, JSON, CSV, HTML, Markdown, etc.) and store it in a queryable format with relationship tracking.

## Python Best Practices

### Code Organization
1. **Single Responsibility Principle**: Each class/function should have one clear purpose
2. **DRY (Don't Repeat Yourself)**: Extract common functionality into reusable functions
3. **Explicit is better than implicit**: Use clear, descriptive names
4. **Composition over inheritance**: Prefer composition and mixins over deep inheritance hierarchies

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

## Testing Best Practices

### Test Organization
```
tests/
├── unit/           # Unit tests - test individual components in isolation
├── integration/    # Integration tests - test component interactions
├── performance/    # Performance tests - test speed and resource usage
├── fixtures/       # Test data and fixtures
└── conftest.py     # Pytest configuration and shared fixtures
```

### Test Categorization with Markers

#### pytest.ini Configuration
```ini
[tool.pytest.ini_options]
markers = [
    "unit: Unit tests - fast, isolated component tests",
    "integration: Integration tests - test component interactions",
    "performance: Performance tests - measure speed and resources",
    "slow: Tests that take > 1 second",
    "requires_pdf: Tests requiring PyMuPDF",
    "requires_docx: Tests requiring python-docx"
]
```

#### Test Marking Examples
```python
import pytest
from unittest.mock import Mock, patch

@pytest.mark.unit
class TestDocumentParser:
    """Unit tests for DocumentParser base class."""
    
    def test_generate_id(self):
        """Test ID generation - no external dependencies."""
        parser = DocumentParser()
        id1 = parser._generate_id("test_")
        id2 = parser._generate_id("test_")
        
        assert id1.startswith("test_")
        assert id1 != id2  # IDs should be unique
        assert len(id1) == 13  # prefix + 8 chars

@pytest.mark.integration
class TestPdfParserIntegration:
    """Integration tests for PDF parser with real files."""
    
    @pytest.mark.requires_pdf
    def test_parse_real_pdf(self, sample_pdf_path):
        """Test parsing actual PDF file."""
        parser = PdfParser()
        with open(sample_pdf_path, 'rb') as f:
            result = parser.parse({
                "id": sample_pdf_path,
                "content": f.read(),
                "metadata": {}
            })
        
        assert_valid_parse_result(result)

@pytest.mark.performance
class TestParserPerformance:
    """Performance tests for parsers."""
    
    @pytest.mark.slow
    def test_large_document_performance(self, large_pdf_path):
        """Test parsing speed for large documents."""
        import time
        
        parser = PdfParser({"max_pages": 1000})
        
        start_time = time.time()
        with open(large_pdf_path, 'rb') as f:
            result = parser.parse({
                "id": large_pdf_path,
                "content": f.read(),
                "metadata": {}
            })
        elapsed = time.time() - start_time
        
        # Performance assertions
        assert elapsed < 10.0  # Should parse in under 10 seconds
        assert len(result["elements"]) > 0
```

### Running Tests by Category
```bash
# Run only unit tests (fast)
pytest -m unit

# Run integration tests
pytest -m integration

# Run performance tests
pytest -m performance

# Run all non-slow tests
pytest -m "not slow"

# Run with coverage for unit tests only
pytest -m unit --cov=src/go_doc_go --cov-report=html
```

### Test Design Principles

#### 1. Test Design Objectives, Not Implementation
```python
# BAD: Testing current behavior
def test_pdf_parser_creates_content_element():
    """This tests what the code does, not what it should do."""
    result = parser.parse(pdf_content)
    # This might pass even if "content" is wrong
    assert result["elements"][0]["element_type"] == "content"

# GOOD: Testing design requirements
def test_pdf_parser_creates_valid_element_types():
    """Test that all created elements use valid ElementType values."""
    result = parser.parse(pdf_content)
    
    for element in result["elements"]:
        element_type = element["element_type"]
        # Validate against design specification
        assert element_type in [e.value for e in ElementType], \
            f"Invalid element type '{element_type}' not in ElementType enum"
```

#### 2. Use Fixtures for Test Data
```python
# conftest.py
import pytest
from pathlib import Path

@pytest.fixture
def sample_documents_dir():
    """Directory containing sample documents for testing."""
    return Path(__file__).parent / "fixtures" / "documents"

@pytest.fixture
def simple_pdf_content():
    """Create a simple PDF for testing."""
    import fitz
    doc = fitz.open()
    page = doc.new_page()
    page.insert_text((72, 72), "Test Document", fontsize=16)
    page.insert_text((72, 100), "Test paragraph content.", fontsize=11)
    return doc.tobytes()

@pytest.fixture
def expected_pdf_structure():
    """Expected structure for simple PDF parsing."""
    return {
        "min_elements": 3,  # root, body, page minimum
        "required_types": [ElementType.ROOT, ElementType.BODY, ElementType.PAGE],
        "relationships": [RelationshipType.CONTAINS, RelationshipType.CONTAINED_BY]
    }
```

#### 3. Test Helpers and Assertions
```python
# test_helpers.py
def assert_valid_parse_result(result: Dict[str, Any]):
    """Common assertions for parse results."""
    assert "document" in result
    assert "elements" in result
    assert "relationships" in result
    
    # Validate document structure
    doc = result["document"]
    assert "doc_id" in doc
    assert "doc_type" in doc
    assert "metadata" in doc
    
    # Validate all elements
    for element in result["elements"]:
        assert_valid_element(element)
    
    # Validate relationships
    for rel in result["relationships"]:
        assert_valid_relationship(rel)

def assert_valid_element(element: Dict[str, Any]):
    """Validate element structure and types."""
    required_fields = ["element_id", "element_type", "content_preview"]
    for field in required_fields:
        assert field in element, f"Missing required field: {field}"
    
    # Validate element type
    element_type = element["element_type"]
    valid_types = [e.value for e in ElementType]
    assert element_type in valid_types, \
        f"Invalid element_type: {element_type}"

def assert_valid_relationship(rel: Dict[str, Any]):
    """Validate relationship structure."""
    required_fields = ["source_id", "target_id", "relationship_type"]
    for field in required_fields:
        assert field in rel, f"Missing required field: {field}"
    
    # Validate relationship type
    rel_type = rel["relationship_type"]
    valid_types = [r.value for r in RelationshipType]
    assert rel_type in valid_types, \
        f"Invalid relationship_type: {rel_type}"
```

### Performance Testing Guidelines

```python
@pytest.mark.performance
class TestPerformanceRequirements:
    """Test performance requirements and SLAs."""
    
    @pytest.fixture(autouse=True)
    def setup_performance_monitoring(self):
        """Setup performance monitoring for tests."""
        import psutil
        import time
        
        self.process = psutil.Process()
        self.start_memory = self.process.memory_info().rss / 1024 / 1024  # MB
        self.start_time = time.time()
        
        yield
        
        self.end_memory = self.process.memory_info().rss / 1024 / 1024  # MB
        self.elapsed_time = time.time() - self.start_time
        self.memory_used = self.end_memory - self.start_memory
    
    def test_memory_usage_under_limit(self, large_document):
        """Test that memory usage stays within limits."""
        parser = create_parser("pdf")
        result = parser.parse(large_document)
        
        # Memory should not exceed 500MB for a 100MB document
        assert self.memory_used < 500, \
            f"Memory usage {self.memory_used}MB exceeds limit"
    
    def test_parsing_speed_requirements(self, standard_document):
        """Test parsing speed meets requirements."""
        parser = create_parser("pdf")
        result = parser.parse(standard_document)
        
        # Should parse standard document in under 1 second
        assert self.elapsed_time < 1.0, \
            f"Parsing took {self.elapsed_time}s, exceeds 1s limit"
    
    @pytest.mark.parametrize("doc_size,time_limit", [
        (1, 0.1),    # 1MB document should parse in 100ms
        (10, 1.0),   # 10MB document should parse in 1s
        (100, 10.0), # 100MB document should parse in 10s
    ])
    def test_scaling_performance(self, create_document, doc_size, time_limit):
        """Test that parsing scales linearly with document size."""
        document = create_document(size_mb=doc_size)
        parser = create_parser("pdf")
        
        start = time.time()
        result = parser.parse(document)
        elapsed = time.time() - start
        
        assert elapsed < time_limit, \
            f"{doc_size}MB document took {elapsed}s, exceeds {time_limit}s limit"
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

### Before Committing
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

### Performance Benchmarks
- Standard document (< 10MB): Parse in < 1 second
- Large document (< 100MB): Parse in < 10 seconds
- Memory usage: < 5x document size
- Concurrent parsing: Support 10 simultaneous parsers

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

## Common Issues and Solutions

### Issue: Tests passing but functionality broken
**Solution**: Tests should validate against design specs, not current behavior
```python
# Always test against specifications
assert element_type in VALID_ELEMENT_TYPES  # Not what the code returns
```

### Issue: Slow tests
**Solution**: Use test markers and run categories separately
```python
@pytest.mark.slow
@pytest.mark.integration
def test_large_file_parsing():
    pass
```

### Issue: Flaky tests
**Solution**: Use fixtures and deterministic test data
```python
@pytest.fixture
def deterministic_uuid(monkeypatch):
    """Make UUID generation deterministic for tests."""
    counter = 0
    def mock_uuid4():
        nonlocal counter
        counter += 1
        return f"test-uuid-{counter:08d}"
    monkeypatch.setattr(uuid, 'uuid4', mock_uuid4)
```

## Code Review Checklist

- [ ] All new code has tests
- [ ] Tests validate design objectives, not implementation
- [ ] No duplicated code (DRY principle followed)
- [ ] Type hints used for all functions
- [ ] Error handling is specific and logged
- [ ] Performance tests for resource-intensive operations
- [ ] Documentation updated for API changes
- [ ] Element types and relationships use proper enums
- [ ] Memory usage is bounded for large inputs
- [ ] Coverage meets minimum requirements