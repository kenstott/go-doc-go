# Recent Changes

## Document Parsing Improvements

### DOCX Parser Enhancements

#### 1. Container Elements (Parity with Python)
Added 3 container elements to match Python implementation:
- `headers` - Document headers container (child of root)
- `footers` - Document footers container (child of root)
- `comments` - Document comments container (child of root)

#### 2. Hierarchical Document Structure
Implemented proper hierarchical parent-child relationships:
- Headers can be nested (h2 as child of h1, h3 as child of h2, etc.)
- Paragraphs are children of their enclosing header
- Section stack properly maintains hierarchy levels
- Matches Python's section tracking behavior

**Example hierarchy:**
```python
root
├── headers (container)
├── footers (container)
├── body
│   └── h1 "Main Title"
│       ├── paragraph "Intro"
│       ├── h2 "Section 1"
│       │   ├── paragraph "Para in sect 1"
│       │   └── h3 "Subsection 1.1"
│       │       └── paragraph "Para in subsect 1.1"
│       └── h2 "Section 2"
│           └── paragraph "Para in sect 2"
└── comments (container)
```

#### 3. Hyperlink Extraction
- Reads `word/_rels/document.xml.rels` to extract hyperlinks
- Identifies hyperlink relationships by type
- Adds extracted links to `parseResult.Links`
- Links are automatically enqueued for crawling (respecting depth limits)

**Files changed:**
- `go/internal/parser/docx.go` - Added container elements, fixed hierarchy
- `go/internal/parser/docx_links.go` - New file for link extraction
- `src/go_doc_go/document_parser/docx.py` - Fixed hierarchy, container parent

### Link Enqueueing Architecture

#### Unified Link Handling
Refactored link processing to use a consistent pattern across all document types:

**Before:**
- HTML had custom `extractAndQueueLinks()` in worker
- Other parsers didn't extract or enqueue links
- Inconsistent approach between document types

**After:**
- All parsers return links in `parseResult.Links`
- Worker has single `queueLinksFromParseResult()` function
- Uses job control interface: `w.jobControl.EnqueueDocument()`
- Works for HTML, DOCX, PDF, and all other document types

**Features:**
- Respects `max_link_depth` from source config
- Applies `include_patterns` and `exclude_patterns` filters
- Tracks discovery depth in metadata
- Prevents infinite crawling with depth limits

**Files changed:**
- `go/internal/worker/worker.go` - Added `queueLinksFromParseResult()`, unified link handling
- `go/internal/parser/html.go` - Already had link extraction (no changes needed)

## Licensing

### Dual License Structure
Added dual GPL-3.0 + Commercial licensing:

**Files added:**
- `LICENSE` - Dual license overview and selection guide
- `LICENSE-GPL` - Full GNU General Public License v3.0 text
- `LICENSE-COMMERCIAL` - Commercial license template

**License Options:**
1. **GPL-3.0**: For open source projects (requires source disclosure)
2. **Commercial**: For proprietary software (no disclosure requirements)

## Python & Go Parity

### DOCX Parsing
✅ Both create same element structure (root, headers, footers, body, comments)
✅ Both support hierarchical headers with proper parent-child relationships
✅ Both extract hyperlinks from documents
✅ Element counts match (7 elements for simple document)

### Contextual Embeddings
✅ Both support hierarchy-aware predecessor/successor collection
✅ Predecessors stop at parent boundary (ancestors excluded)
✅ Successors skip descendants dynamically
✅ No deduplication needed when rules are correct

## Configuration

### Job Control Interface
All link enqueueing now uses the job control interface pattern:
- Decoupled from storage implementation (SQLite, PostgreSQL, etc.)
- Consistent across all document types
- Supports distributed worker coordination

## Testing

To test DOCX parsing with nested headers:
```bash
go run ./cmd/test_docx /path/to/document.docx
```bash

To test link extraction:
- Add hyperlinks to a DOCX document
- Parse with worker
- Check that links are enqueued in job control queue

## Breaking Changes

None - all changes are backward compatible additions.

## Future Enhancements

Potential improvements for consideration:
1. Parse actual DOCX page headers/footers from `word/header*.xml` and `word/footer*.xml`
2. Parse comments from `word/comments.xml` (currently just container)
3. More sophisticated hyperlink text extraction (currently uses URL as text)
4. Remove old `extractAndQueueLinks()` function (no longer called)
