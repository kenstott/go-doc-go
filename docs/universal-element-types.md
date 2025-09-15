k# Universal Element Types Documentation

## Overview

The Go-Doc-Go system uses a standardized set of **universal element types** to provide consistency across different document formats. While each parser may encounter format-specific element types (native types), these are mapped to universal types for unified processing and searching.

## Universal Element Types vs Native Element Types

### Universal Element Types
Universal element types are the standardized types defined in the `ElementType` enum in `src/go_doc_go/storage/element_element.py`. These provide a common vocabulary for document elements regardless of source format.

### Native Element Types
Native element types are format-specific types that parsers encounter in their respective document formats (e.g., Word's "w:p" for paragraphs, HTML's "div" elements).

## Complete List of Universal Element Types

### Document Structure Types
- **`root`** - The top-level container for the entire document
- **`body`** - Main content body of the document
- **`page`** - Individual page in paginated documents (PDF, DOCX)
- **`section`** - Logical section or chapter division

### Text Content Types
- **`header`** - Heading/title elements (H1-H6 in HTML, headings in Word)
  - Metadata includes `level` (1-6) for heading hierarchy
- **`paragraph`** - Standard text paragraph
- **`line`** - Individual line of text (used in text files)
- **`range`** - A range of characters within text
- **`substring`** - A substring within a larger text element

### List Types
- **`list`** - Container for list items (ordered or unordered)
- **`list_item`** - Individual item within a list

### Table Types
- **`table`** - Table container
- **`table_row`** - Row within a table
- **`table_header_row`** - Header row in a table
- **`table_cell`** - Individual cell within a table row
- **`table_header`** - Header cell within a table

### Media and Special Content Types
- **`image`** - Image or graphic element
- **`chart`** - Chart or graph visualization
- **`shape`** - Drawing shape or graphic object
- **`shape_group`** - Group of shapes

### Code and Quotes
- **`code_block`** - Block of source code
  - Metadata may include `language` for syntax highlighting
- **`blockquote`** - Quoted text block

### Presentation Types (PowerPoint/Slides)
- **`presentation_body`** - Main container for presentation content
- **`slide`** - Individual slide
- **`slide_notes`** - Speaker notes for a slide
- **`slide_master`** - Master slide template
- **`slide_layout`** - Slide layout template
- **`slide_masters`** - Container for slide masters
- **`slide_templates`** - Container for slide templates
- **`text_box`** - Text container in presentations

### Spreadsheet Types (Excel)
- **`workbook`** - Excel workbook container
- **`sheet`** - Individual worksheet
- **`merged_cell`** - Merged cell in spreadsheet
- **`merged_cells`** - Container for merged cells
- **`data_table`** - Logical data table detected within a sheet
- **`data_tables`** - Container for multiple data tables

### Structured Data Types (JSON/XML)
- **`json_object`** - JSON object container
- **`json_array`** - JSON array container
- **`json_field`** - Individual JSON field/property
- **`json_item`** - Item within a JSON array
- **`xml_element`** - XML element node
- **`xml_text`** - XML text content
- **`xml_list`** - XML list structure
- **`xml_object`** - XML object structure

### Document Metadata Types
- **`page_header`** - Page header content
- **`page_footer`** - Page footer content
- **`headers`** - Container for headers
- **`footers`** - Container for footers
- **`comment`** - Comment or annotation
- **`comments`** - Container for comments
- **`comments_container`** - Higher-level container for comment sections

### Special Type
- **`unknown`** - Fallback for unrecognized element types

## Container vs Leaf Elements

### Container Elements
Elements that can contain other elements:
- root, body, page, section
- div, article
- list, table
- presentation_body, slide
- workbook, sheet
- json_object, json_array
- xml_list, xml_object
- And various container types (headers, footers, comments_container, etc.)

### Leaf Elements
Elements that typically contain only content, not other elements:
- paragraph, header
- list_item
- table_cell
- image, chart
- code_block, blockquote
- comment
- json_field, json_item
- xml_text

## Element Type Filtering in Search

When using the semantic search API, you can filter results by element type:

### Excluding Element Types
To exclude certain element types from search results:
```python
# Exclude headers and list items from results
search_results = search_helper.search_with_related_elements(
    query="disappointing results",
    exclude_element_types=["header", "list_item"]
)
```

### Including Only Specific Element Types
To include only specific element types:
```python
# Only return paragraphs and blockquotes
search_results = search_helper.search_with_related_elements(
    query="revenue analysis",
    include_element_types=["paragraph", "blockquote"]
)
```

## Common Use Cases for Filtering

### Focus on Content (exclude structural elements)
```python
exclude_element_types=[
    "root", "body", "page", 
    "headers", "footers",
    "page_header", "page_footer"
]
```

### Focus on Text (exclude tables and media)
```python
include_element_types=[
    "paragraph", "header", 
    "list_item", "blockquote"
]
```

### Focus on Data (tables and structured data)
```python
include_element_types=[
    "table", "table_row", "table_cell",
    "json_field", "json_item",
    "data_table"
]
```

### Exclude Navigation and Metadata
```python
exclude_element_types=[
    "page_header", "page_footer",
    "comment", "comments",
    "slide_notes", "slide_master"
]
```

## Implementation Notes

1. **Case Insensitive**: Element type comparisons are typically case-insensitive
2. **Normalization**: Parsers should map native types to universal types during parsing
3. **Validation**: Element types should be validated against the `ElementType` enum
4. **Hierarchy**: The parent-child relationships must be maintained regardless of element type
5. **Metadata**: Additional type-specific information is stored in the element's metadata field

## Best Practices

1. **Use Universal Types**: Always use universal element types for filtering and searching
2. **Preserve Native Types**: Store native type information in metadata for debugging
3. **Consistent Mapping**: Ensure consistent mapping from native to universal types
4. **Document Mappings**: Document any new mappings when adding parser support
5. **Test Filtering**: Test element type filtering with various document formats

## Parser-Specific Mappings

### PDF Parser
- PDF text blocks → `paragraph`
- PDF annotations → `comment`
- PDF pages → `page`

### DOCX Parser
- w:p (Word paragraph) → `paragraph`
- w:tbl (Word table) → `table`
- w:tr (Word table row) → `table_row`
- Heading styles → `header` with level metadata

### HTML Parser
- `<h1>` to `<h6>` → `header` with level
- `<p>` → `paragraph`
- `<div>` → Depends on context, often container
- `<ul>`, `<ol>` → `list`
- `<li>` → `list_item`

### Markdown Parser
- `#` headers → `header` with level
- Text blocks → `paragraph`
- `- ` items → `list_item`
- Code fences → `code_block`
- `>` quotes → `blockquote`

### Excel Parser
- Workbook → `workbook`
- Sheets → `sheet`
- Data rows → `table_row`
- Cells → `table_cell` or `table_header`

### JSON Parser
- Objects → `json_object`
- Arrays → `json_array`
- Properties → `json_field`
- Array items → `json_item`

## Future Considerations

1. **Semantic Types**: Consider adding semantic types (e.g., "definition", "example", "warning")
2. **Custom Types**: Allow domain-specific custom types with fallback to universal types
3. **Type Hierarchies**: Implement type inheritance for more sophisticated filtering
4. **Type Aliases**: Support alternative names for common types
5. **Type Validation**: Stricter validation and automatic correction of invalid types
