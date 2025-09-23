what i# Example: Ontology Generation for Insider Trading Domain

This example shows how to use the ontology generation prompt with actual data from SEC filings.

## Domain Context

**Domain**: Insider Trading Detection

**Domain Description**:
Insider trading involves corporate insiders (officers, directors, 10%+ shareholders) trading securities while possessing material, non-public information. SEC requires disclosure of insider transactions through Forms 3, 4, and 5. Detection involves analyzing transaction patterns, timing relative to material events, and relationships between insiders and companies.

**Key Concepts**: Corporate insiders, beneficial ownership, securities transactions, material events, reporting requirements, timing analysis, unusual patterns

**Important Relationships**:
- Person → Company (employment, board membership)
- Person → Transaction (execution)
- Transaction → Company (securities involved)
- Transaction → Date (timing)
- Transaction → Material Event (proximity analysis)

**Regulatory/Business Context**:
SEC regulations Section 16, Rule 10b-5, disclosure requirements, blackout periods, earnings announcements, material corporate events, pattern analysis for compliance and surveillance.

## Document Analysis Data

### Document Types Present
- SEC Form 4 (XML/XBRL): Ownership change statements
- SEC Form 3 (XML/XBRL): Initial ownership statements
- SEC Form 5 (XML/XBRL): Annual ownership statements
- Company financial reports (JSON/HTML): Earnings announcements
- Corporate calendars (CSV): Blackout periods, material events

### Sample Element Records

```json
[
  {
    "element_id": "xml_elem_rptOwnerName_1c2d3e4f",
    "element_type": "xml_element",
    "content_preview": "BELL JAMES A",
    "metadata": {
      "element_name": "rptOwnerName",
      "path": "/ownershipDocument/reportingOwner/reportingOwnerId/rptOwnerName",
      "attributes": {}
    }
  },
  {
    "element_id": "xml_elem_issuerName_5g6h7i8j",
    "element_type": "xml_element",
    "content_preview": "Apple Inc.",
    "metadata": {
      "element_name": "issuerName",
      "path": "/ownershipDocument/issuer/issuerName",
      "attributes": {}
    }
  },
  {
    "element_id": "xml_elem_transactionDate_9k0l1m2n",
    "element_type": "xml_element",
    "content_preview": "2023-02-01",
    "temporal_value": {
      "type": "date",
      "value": "2023-02-01"
    },
    "metadata": {
      "element_name": "transactionDate",
      "path": "/ownershipDocument/nonDerivativeTable/nonDerivativeTransaction/transactionDate",
      "attributes": {}
    }
  },
  {
    "element_id": "xml_elem_transactionShares_3o4p5q6r",
    "element_type": "xml_element",
    "content_preview": "1685",
    "metadata": {
      "element_name": "transactionShares",
      "path": "/ownershipDocument/nonDerivativeTable/nonDerivativeTransaction/transactionAmounts/transactionShares",
      "attributes": {}
    }
  },
  {
    "element_id": "xml_elem_transactionCode_7s8t9u0v",
    "element_type": "xml_element",
    "content_preview": "S",
    "metadata": {
      "element_name": "transactionCode",
      "path": "/ownershipDocument/nonDerivativeTable/nonDerivativeTransaction/transactionCoding/transactionCode",
      "attributes": {}
    }
  },
  {
    "element_id": "json_field_earnings_date_1w2x3y4z",
    "element_type": "json_field",
    "content_preview": "2023-02-02",
    "temporal_value": {
      "type": "date",
      "value": "2023-02-02"
    },
    "metadata": {
      "json_path": "earnings.announcement_date",
      "field_name": "announcement_date"
    }
  }
]
```

### Element Type Distribution
- xml_element: 85% (SEC forms)
- json_field: 10% (financial reports)
- csv_cell: 3% (calendar data)
- html_element: 2% (web reports)

### Common Metadata Patterns

**XML Elements (SEC Forms)**:
- `element_name`: Tag names like `rptOwnerName`, `issuerName`, `transactionDate`
- `path`: Hierarchical XPath showing document structure
- `attributes`: Mostly empty, some with `id` references

**JSON Fields (Financial Reports)**:
- `json_path`: Dot notation like `earnings.announcement_date`, `events.merger_date`
- `field_name`: Property names like `announcement_date`, `executive_name`

**CSV Cells (Calendar Data)**:
- `column_name`: Headers like `Executive`, `Company`, `Blackout_Start`, `Event_Type`
- `row_index`: Position in spreadsheet

### Frequent Element Names/Paths

**Most Common Element Names**:
1. `rptOwnerName` (person names) - 23%
2. `issuerName` (company names) - 18%
3. `transactionDate` (dates) - 15%
4. `transactionShares` (amounts) - 12%
5. `transactionCode` (transaction types) - 8%
6. `ownershipNature` (direct/indirect) - 6%
7. `relationshipTitle` (job titles) - 5%

**Most Common Path Patterns**:
1. `/ownershipDocument/reportingOwner/*` (person info)
2. `/ownershipDocument/issuer/*` (company info)
3. `/ownershipDocument/*/transaction*/*` (transaction details)
4. `earnings.*` (financial event data)
5. `executives.*` (leadership info)

---

## Expected Output

When this data is fed into the ontology generation prompt, it should produce rules like:

```yaml
terms:
  - id: "reporting_person"
    label: "Reporting Person"
    description: "Individual required to file SEC ownership forms"
    aliases: ["filer", "reporting owner", "insider"]

element_mappings:
  - term_id: "reporting_person"
    rules:
      - type: keywords
        search_scope: "element_name"
        keywords: ["rptOwnerName", "reportingOwnerName"]
        confidence_threshold: 0.95

      - type: keywords
        search_scope: "path"
        keywords: ["reportingOwner", "reportingOwnerId"]
        confidence_threshold: 0.85

relationship_rules:
  - id: "person_works_for_company"
    relationship_type: "EMPLOYED_BY"
    source:
      term_id: "reporting_person"
    target:
      term_id: "issuer"
    constraints:
      same_document: true
```

This example demonstrates how domain expertise + document analysis data produces precise, actionable ontology rules.
