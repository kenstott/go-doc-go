# Insider Trading Detection Ontology

## Overview

This ontology is designed to automatically extract entities and relationships from SEC filings (Forms 3, 4, 5) and related documents to build a comprehensive knowledge graph for insider trading detection and analysis.

## Knowledge Graph Structure

### Core Entity Types

#### 1. **People & Roles**
- **Corporate Insiders**: Officers, directors, 10%+ shareholders
- **Reporting Persons**: Anyone required to file SEC ownership forms
- **Beneficial Owners**: Entities with voting/investment control

#### 2. **Companies**
- **Issuers**: Companies whose stock is being traded
- **Subsidiaries**: Related corporate entities

#### 3. **Transactions**
- **Stock Transactions**: Purchases, sales, option exercises
- **Transaction Attributes**: Dates, amounts, prices

#### 4. **Temporal Events**
- **Material Events**: Earnings, mergers, major announcements
- **Blackout Periods**: Restricted trading windows

#### 5. **Regulatory Forms**
- **SEC Forms**: Form 3, 4, 5 classifications

### Relationship Types

#### Direct Relationships
```
Person --[EMPLOYED_BY]--> Company
Person --[INSIDER_AT]--> Company
Person --[EXECUTED]--> Transaction
Transaction --[INVOLVES_SECURITIES_OF]--> Company
Transaction --[OCCURRED_ON]--> Date
Transaction --[HAS_AMOUNT]--> Amount
```

#### Temporal Relationships
```
Transaction --[OCCURRED_NEAR]--> EarningsDate
Transaction --[OCCURRED_DURING]--> BlackoutPeriod
```

#### Pattern Relationships
```
Transaction --[PATTERN_WITH]--> Transaction (same person, timeframe)
```

## Detection Capabilities

### 1. **Element Extraction from SEC Forms**

**From XML/XBRL filings like ownership.xml:**
```xml
<rptOwnerName>BELL JAMES A</rptOwnerName>
<issuerName>Apple Inc.</issuerName>
<transactionDate>2023-02-01</transactionDate>
<transactionShares>1685</transactionShares>
```

**Extracted entities:**
- Person: "BELL JAMES A" (classified as insider/reporting_person)
- Company: "Apple Inc." (classified as issuer)
- Transaction with date and amount attributes

### 2. **Relationship Building**

The system automatically creates:
```
"BELL JAMES A" --[INSIDER_AT]--> "Apple Inc."
"BELL JAMES A" --[EXECUTED]--> Transaction_123
Transaction_123 --[INVOLVES_SECURITIES_OF]--> "Apple Inc."
Transaction_123 --[OCCURRED_ON]--> "2023-02-01"
Transaction_123 --[HAS_AMOUNT]--> "1685 shares"
```

### 3. **Advanced Pattern Detection**

#### Derived Entities

**TradingPattern**
```json
{
  "entity_type": "TradingPattern",
  "person": "BELL JAMES A",
  "company": "Apple Inc.",
  "total_value": "$2.3M",
  "transaction_count": 5,
  "date_range": "2023-01-15 to 2023-03-01",
  "pattern_type": "concentrated_selling"
}
```

**InsiderProfile**
```json
{
  "entity_type": "InsiderProfile",
  "name": "BELL JAMES A",
  "companies": ["Apple Inc.", "Subsidiary Corp"],
  "roles": ["Director", "Senior VP"],
  "total_transactions": 12,
  "recent_activity": 3
}
```

**SuspiciousActivity**
```json
{
  "entity_type": "SuspiciousActivity",
  "person": "BELL JAMES A",
  "risk_level": "HIGH",
  "reasons": [
    "Large sale 2 days before earnings",
    "Transaction during blackout period",
    "Unusual volume (5x normal)"
  ],
  "risk_score": 0.85
}
```

## Use Cases

### 1. **Regulatory Compliance**
- Identify potential Section 16 violations
- Detect blackout period violations
- Monitor 10b5-1 plan adherence

### 2. **Market Surveillance**
- Flag unusual trading patterns before material events
- Identify coordinated insider activity
- Track serial violators

### 3. **Risk Assessment**
- Score insider trading risk by person/company
- Identify high-risk time periods
- Monitor concentration of insider activity

### 4. **Investigation Support**
- Map relationships between insiders and companies
- Timeline analysis of trading vs. material events
- Pattern analysis across multiple entities

## Query Examples

With this knowledge graph, you can query:

### Find all transactions near earnings
```cypher
MATCH (t:Transaction)-[:OCCURRED_NEAR]->(e:EarningsDate)
WHERE e.date WITHIN 30 DAYS OF t.date
RETURN t, e
```

### Identify repeat violators
```cypher
MATCH (p:Person)-[:EXECUTED]->(t:Transaction)-[:OCCURRED_DURING]->(b:BlackoutPeriod)
WITH p, COUNT(t) as violations
WHERE violations > 1
RETURN p.name, violations
```

### Find insider networks
```cypher
MATCH (p1:Person)-[:INSIDER_AT]->(c:Company)<-[:INSIDER_AT]-(p2:Person)
WHERE p1 <> p2
RETURN p1, p2, c
```

### Detect suspicious patterns
```cypher
MATCH (sa:SuspiciousActivity)
WHERE sa.risk_level = 'HIGH'
RETURN sa.person, sa.reasons, sa.risk_score
ORDER BY sa.risk_score DESC
```

## Configuration

### Key Thresholds
- **Temporal proximity**: 30 days for earnings, 14 days for material events
- **Volume threshold**: 3x normal trading volume
- **Pattern detection**: Minimum 3 transactions for pattern
- **Risk scoring**: Weighted by timing (40%), size (30%), frequency (20%), role (10%)

### Element Matching
The ontology uses our enhanced search system to match:
- **Element names**: `rptOwnerName`, `issuerName`, `transactionDate`
- **Paths**: `/ownershipDocument/reportingOwner/`
- **Attribute values**: Form types, transaction codes
- **Full content**: Complete text for semantic matching

This creates a comprehensive, automated system for building insider trading knowledge graphs from raw SEC filings and related documents.