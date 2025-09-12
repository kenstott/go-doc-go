# Document Parser Tests

This directory contains comprehensive unit tests for the document parser module.

## Test Coverage Summary

### Files Created
- **test_base_parser.py** - Tests for the base DocumentParser class (19 tests, 446 lines)
- **test_csv_parser.py** - Tests for CSV document parsing (23 tests, 505 lines)
- **test_pdf_parser.py** - Tests for PDF document parsing (16 tests, 680 lines)
- **test_json_parser.py** - Tests for JSON document parsing (23 tests, 578 lines)
- **test_error_handling.py** - Cross-parser error handling tests (23 tests, 529 lines)

### Total Statistics
- **104 test functions** created
- **2,738 lines** of test code
- **5 test modules** covering core parsers

## Test Categories

### Base Parser Tests
- ID generation and uniqueness
- Content hash generation
- Element creation
- Root element handling
- Content preview truncation
- JSON serialization
- Relationship creation

### CSV Parser Tests
- Basic CSV parsing
- Delimiter detection (comma, semicolon, tab)
- Header extraction
- Quoted fields with special characters
- Empty cells handling
- Large file handling with row limits
- Malformed CSV recovery
- Unicode content
- Date extraction

### PDF Parser Tests
- Text extraction from pages
- Metadata extraction
- Table detection and parsing
- Image extraction
- Link and annotation extraction
- Large PDF handling
- Corrupt file handling
- Password-protected PDFs
- Page ordering

### JSON Parser Tests
- Nested structure parsing
- Array handling
- Primitive value types
- Deep nesting limits
- Schema extraction
- Special characters in keys
- Unicode content
- JSON Lines format
- GeoJSON structures
- Large JSON handling

### Error Handling Tests
- Missing required fields
- Invalid content types
- Extremely large content
- Unicode and control characters
- Corrupt binary files
- Invalid syntax handling
- Memory and performance tests
- Thread safety
- Factory error handling

## Running the Tests

### Prerequisites
```bash
pip install pytest
pip install datefinder
pip install flask flask-cors
```

### Run All Tests
```bash
PYTHONPATH=src pytest tests/test_parsers/ -v
```

### Run Specific Test File
```bash
PYTHONPATH=src pytest tests/test_parsers/test_csv_parser.py -v
```

### Run Specific Test
```bash
PYTHONPATH=src pytest tests/test_parsers/test_base_parser.py::TestDocumentParser::test_generate_id -v
```

## Known Issues

1. **Import Dependencies**: The go_doc_go module has some initialization dependencies that may require additional packages to be installed.

2. **Database Connection**: Some imports trigger database connection attempts. You may need to set environment variables or mock these connections for tests to run.

3. **Module Structure**: The project uses the old name "go_doc_go" internally while the project directory is "go-doc-go".

## Test Design Principles

1. **Comprehensive Coverage**: Each parser has tests for normal operations, edge cases, and error conditions.

2. **Mocking External Dependencies**: Binary parsers (PDF, DOCX, etc.) use mocks to avoid requiring actual files.

3. **Data Validation**: Tests verify that parsed data maintains structure, relationships, and content integrity.

4. **Performance Testing**: Includes tests for large files and memory usage.

5. **Error Recovery**: Tests verify graceful handling of malformed or corrupt input.

## Next Steps

To achieve full test coverage, consider adding:
1. Tests for remaining parsers (HTML, XML, Markdown, Text, DOCX, PPTX, XLSX)
2. Integration tests for the parser factory
3. End-to-end tests with real documents
4. Performance benchmarks
5. Coverage reporting integration