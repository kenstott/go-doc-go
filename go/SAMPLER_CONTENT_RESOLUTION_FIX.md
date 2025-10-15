# Sampler Content Resolution Fix

## Problem Identified

The ontology interview system was receiving placeholder text like `[table_row]` instead of actual content from container elements (table_row, json_object, json_field, etc).

## Root Cause Analysis

1. **Container elements have NULL `content` field in Parquet**
   - By design, structural elements like `table_row`, `json_object`, `json_field` don't store their own content
   - They only have a `content_preview` field (e.g., "Run Date, Account, Account Number...")
   - The `content` field is NULL/empty for these elements

2. **Sampler was only reading `content` field**
   - The sampler queried the `content` field directly from Parquet
   - When `content` was NULL, it was converted to empty string
   - The LLM received empty or minimal content for analysis

## Solution Implemented

Implemented a **3-part waterfall for content resolution** in the sampler, matching the pattern already used in the embeddings module (`internal/embeddings/contextual.go:45-62`):

### Waterfall Logic

```go
func (s *Sampler) resolveElementContent(content, contentLocation, contentPreview) string {
    // 1. Try Content field first
    if content != "" {
        return content
    }

    // 2. Try content resolver with content_location
    if contentLocation != nil && s.resolver != nil {
        resolved, err := s.resolver.ResolveContent(contentLocation, true)
        if err == nil && resolved != "" {
            return resolved
        }
    }

    // 3. Fall back to ContentPreview
    return contentPreview
}
```

### Changes Made

**File**: `internal/udml/sampler/sampler.go`

1. **Added ContentResolver support**:
   - Imported `internal/resolver` package
   - Added `resolver resolver.ContentResolver` field to `Sampler` struct
   - Added `SetResolver()` method to inject resolver (optional)

2. **Added `content_location` to query**:
   - Line 248: Added `{Field: "content_location"}` to select fields
   - This provides the resolver with source document location info

3. **Implemented 3-part waterfall**:
   - Lines 311-320: Extract content, content_location, content_preview
   - Call `resolveElementContent()` to apply waterfall logic
   - Use resolved content for the sample

4. **Added helper method**:
   - Lines 432-449: `resolveElementContent()` implements waterfall
   - Lines 466-474: `extractJSONField()` helper to extract content_location

## Results

### Before Fix
```
LLM received:
  [table_row]
  [table_row]
  [json_object]
```

The LLM suggested generic domains like "investor_relations" and "financial_reporting" based on minimal context.

### After Fix
```
LLM received:
  [table_row] Run Date, Account, Account Number, Action, Symbol...
  [table_row] 03/18/2025, SEP-IRA, 172102369, REINVESTMENT as of Mar-14-2025...
  [paragraph] The Universal Document Model provides a flexible schema...
```

The LLM now suggests **7 specific domains** with high confidence:
1. **Software Architecture & Engineering** (0.85) - "modular architecture", "Universal Document Model"
2. **Project Management** (0.90) - "Daily standups", "burn-down charts"
3. **Risk Management** (0.95) - "risk register", "mitigation strategies"
4. **Financial Management** (0.75) - budget, resource allocation
5. **Quality Assurance** (0.92) - "automated testing", "unit tests"
6. **User/Customer Management** (0.88) - person data, profiles
7. **Documentation** (0.80) - document model, hierarchical structure

## Content Resolution in Different Contexts

### Parquet-Only Storage (Current Use Case)
- **Step 1**: Content field (works for leaf elements: paragraphs, table_cells)
- **Step 2**: Content resolver (N/A - no source documents available)
- **Step 3**: Content preview (works for containers: table_row, json_object)

### With Source Documents (Future Use Case)
- **Step 1**: Content field (if already extracted)
- **Step 2**: Content resolver (resolves from original HTML/XML/DOCX using content_location)
- **Step 3**: Content preview (fallback if resolution fails)

## Testing

### Test Corpus Analysis
```sql
SELECT element_type,
       COUNT(*) as total,
       COUNT(content) as has_content,
       COUNT(content_preview) as has_preview
FROM parquet_files
GROUP BY element_type;
```

Results:
- `table_row`: 128 elements, **0 have content**, 128 have content_preview
- `json_field`: 12 elements, **0 have content**, 12 have content_preview
- `json_object`: 4 elements, **0 have content**, 4 have content_preview
- `table_header_row`: 2 elements, **0 have content**, 2 have content_preview

### Test Script

Run automated test:
```bash
./test_interview_automated.sh
```

This tests the complete workflow:
1. Samples corpus with content resolution
2. LLM analyzes actual content (not placeholders)
3. Generates domain suggestions with reasoning
4. Creates complete ontology schema

## Architecture Alignment

This fix aligns the sampler with the existing content resolution pattern used in:
- **Embeddings module** (`internal/embeddings/contextual.go:45-62`)
- **Content resolver interface** (`internal/resolver/interface.go`)
- **Default content resolver** (`internal/resolver/resolver.go`)

All modules now consistently implement the 3-part waterfall:
1. Direct content field
2. Resolved content (from source documents)
3. Content preview fallback

## Files Modified

1. `internal/udml/sampler/sampler.go` - Implemented content resolution waterfall
2. `test_interview_automated.sh` - Created automated test script

## Files Created

1. `SAMPLER_CONTENT_RESOLUTION_FIX.md` - This documentation
