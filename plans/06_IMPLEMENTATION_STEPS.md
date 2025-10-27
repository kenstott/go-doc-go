# Implementation Steps

## Overview

This document provides a step-by-step implementation checklist for the refactoring.

**Total Steps**: 10 (Step 0-9)
**Estimated Total Changes**: ~1,083 lines

---

## Step 0: Create Global Domain Catalogs

### Files to Create

**New files** (5 total):
- `go/internal/udml/ontology/catalogs/global_who.go`
- `go/internal/udml/ontology/catalogs/global_what.go`
- `go/internal/udml/ontology/catalogs/global_where.go`
- `go/internal/udml/ontology/catalogs/global_when.go`
- `go/internal/udml/ontology/catalogs/global_why.go`

**File to delete**:
- `go/internal/udml/ontology/catalogs/common.go` (existing 730 lines)

### Actions

1. **Define 10 top-level entity types** with baseline extraction rules:
   - person, organization (WHO)
   - document, identifier, role (WHAT)
   - location (WHERE)
   - date, event (WHEN)
   - assertion, hypothesis (WHY)

2. **Define 23 subtypes** with `parent_type` references:
   - See [02_GLOBAL_DOMAIN.md](02_GLOBAL_DOMAIN.md) for complete list

3. **Convert all rules to unified structure**:
   - Remove `Type` field
   - `RuleTypeRegex` → use `Pattern` only
   - `RuleTypeKeyword` → use `PhraseList`
   - Rename filters: `ProximityFilter` → `Proximity`, etc.

4. **Add required fields**:
   - `w_category` for all entity templates
   - `parent_type` for all subtypes
   - Leave `children` arrays empty (auto-computed)

5. **Use new filter names**:
   - `proximity` (not `proximity_filter`)
   - `semantic` (not `semantic_filter`)
   - `dictionary` (not `dictionary_filter`)
   - `llm_validation` (not `llm_false_positive_test`)

6. **Add global relationship rules**:
   - Define common relationship patterns (e.g., `person_works_at_organization`)
   - Use low confidence (0.60-0.70) since they're general patterns
   - Domain catalogs can inherit and refine these rules

### Example: global_who.go Structure

```go
package catalogs

import "github.com/kennethstott/doculyzer-go-conversion/internal/udml/ontology"

// GlobalWhoEntityTemplates defines WHO entity types (person, organization)
var GlobalWhoEntityTemplates = map[string]EntityTemplate{
    "person": {
        EntityType:   "person",
        ParentType:   "",  // Top-level
        WCategory:    "who",
        Domain:       "global",
        Description:  "Individual human being",
        Aliases:      []string{"individual", "human"},
        ElementTypes: []string{"paragraph", "div", "list_item", "table_cell"},
        SampleRules: []ontology.ExtractionRule{
            {
                InstanceName: `(?P<name>[A-Z][a-z]+ [A-Z][a-z]+)`,
                Dictionary: &ontology.DictionaryFilter{
                    RequireUnknownWords: true,
                    MaxKnownWordsRatio:  0.5,
                },
                Semantic: &ontology.SemanticFilter{
                    ReferenceConcepts: []string{
                        "individual person with biography",
                        "personal pronouns referencing name",
                    },
                    SimilarityThreshold: 0.65,
                },
            },
        },
    },
    
    "public_figure": {
        EntityType:   "public_figure",
        ParentType:   "global.person",  // Child of person
        WCategory:    "who",
        Domain:       "global",
        Description:  "Notable public figure or celebrity",
        Aliases:      []string{"celebrity", "notable"},
        ElementTypes: []string{"paragraph", "heading"},
        SampleRules: []ontology.ExtractionRule{
            {
                InstanceName: `(?P<name>[A-Z][a-z]+ [A-Z][a-z]+)`,
                Proximity: &ontology.ProximityFilter{
                    Keywords:     []string{"famous", "notable", "celebrity"},
                    MaxDistance:  50,
                    DistanceUnit: "word",
                },
            },
        },
    },
    
    // ... executive, employee
    
    "organization": {
        EntityType:   "organization",
        ParentType:   "",  // Top-level
        WCategory:    "who",
        Domain:       "global",
        Description:  "Company, institution, agency, or group",
        // ... rules
    },
    
    // ... business, nonprofit, government, educational, healthcare, religious, media
}

// GlobalWhoRelationshipTemplates defines WHO relationship rules
var GlobalWhoRelationshipTemplates = []ontology.EntityRelationshipRule{
    {
        Name:             "person_works_at_organization",
        SourceEntityType: "person",
        TargetEntityType: "organization",
        RelationshipType: ontology.RelationshipPartOf,
        Description:      "Person employed by or affiliated with organization",
        Confidence:       0.70,  // Lower confidence - general pattern
        ExtractionPatterns: []ontology.RelationshipExtractionPattern{
            {
                Type:        ontology.RelPatternProximity,
                SignalWords: []string{"works at", "employed by", "employee of", "works for"},
                MaxDistance: 50,
                Direction:   "forward",
            },
        },
    },
    {
        Name:             "person_located_in_location",
        SourceEntityType: "person",
        TargetEntityType: "location",
        RelationshipType: ontology.RelationshipLocatedIn,
        Description:      "Person resides in or is from location",
        Confidence:       0.65,
        ExtractionPatterns: []ontology.RelationshipExtractionPattern{
            {
                Type:        ontology.RelPatternProximity,
                SignalWords: []string{"lives in", "from", "resides in", "based in"},
                MaxDistance: 50,
                Direction:   "forward",
            },
        },
    },
    {
        Name:             "organization_located_in_location",
        SourceEntityType: "organization",
        TargetEntityType: "location",
        RelationshipType: ontology.RelationshipLocatedIn,
        Description:      "Organization headquartered in or operates in location",
        Confidence:       0.70,
        ExtractionPatterns: []ontology.RelationshipExtractionPattern{
            {
                Type:        ontology.RelPatternProximity,
                SignalWords: []string{"headquartered in", "based in", "located in", "operates in"},
                MaxDistance: 50,
                Direction:   "forward",
            },
        },
    },
}
```

**Notes on Global Relationship Rules**:
- Use **low confidence** (0.60-0.70) since patterns are general
- Domain catalogs can **inherit and refine** using `parent_relationship` field
- Only apply when global entities are **extracted** (not synthesized)
- Example inheritance:
  ```yaml
  # Medical domain inherits from global rule
  entity_relationship_rules:
    - name: physician_works_at_hospital
      parent_relationship: global.person_works_at_organization
      source_entity_type: physician  # More specific
      target_entity_type: hospital   # More specific
      confidence: 0.85  # Higher confidence
      extraction_patterns:
        - type: proximity
          signal_words: [practicing at, on staff at]  # Domain-specific terms
          max_distance: 50
  ```

### Estimated Changes

- Delete: 730 lines (common.go)
- Add: 800 lines (5 new files)
- **Net: +70 lines**

---

## Step 1: Update ElementEntityMapping Schema

### File

`go/internal/udml/ontology/types.go` (lines 90-99)

### Changes

Add 3 new fields to `ElementEntityMapping` struct:

```go
type ElementEntityMapping struct {
    EntityType      string           `json:"entity_type" yaml:"entity_type"`
    ParentType      string           `json:"parent_type,omitempty" yaml:"parent_type,omitempty"`     // NEW
    Children        []string         `json:"children,omitempty" yaml:"children,omitempty"`           // NEW
    Domain          string           `json:"domain" yaml:"domain"`
    WCategory       string           `json:"w_category" yaml:"w_category"`                           // NEW
    Description     string           `json:"description" yaml:"description"`
    ElementTypes    []string         `json:"element_types,omitempty" yaml:"element_types,omitempty"`
    ElementFilter   string           `json:"element_filter,omitempty" yaml:"element_filter,omitempty"`
    Confidence      float64          `json:"confidence" yaml:"confidence"`
    ExtractionRules []ExtractionRule `json:"extraction_rules" yaml:"extraction_rules"`
}
```

### Estimated Changes

**+3 lines**

---

## Step 2: Remove Type Field from ExtractionRule

### File

`go/internal/udml/ontology/types.go` (lines 101-166)

### Changes

**Delete** (lines 101-108):
```go
// DELETE THESE LINES
type ExtractionRuleType string

const (
    RuleTypeContent  ExtractionRuleType = "content_extraction"
    RuleTypeMetadata ExtractionRuleType = "metadata_field"
    RuleTypeJSONPath ExtractionRuleType = "jsonpath_query"
)
```

**Update ExtractionRule struct** (lines 144-166):

**Before**:
```go
type ExtractionRule struct {
    Type                 ExtractionRuleType   `json:"type" yaml:"type"`
    InstanceName         string               `json:"instance_name,omitempty" yaml:"instance_name,omitempty"`
    Pattern              string               `json:"pattern,omitempty" yaml:"pattern,omitempty"`
    ProximityFilter      *ProximityFilter     `json:"proximity_filter,omitempty" yaml:"proximity_filter,omitempty"`
    SemanticFilter       *SemanticFilter      `json:"semantic_filter,omitempty" yaml:"semantic_filter,omitempty"`
    FieldPath            string               `json:"field_path,omitempty" yaml:"field_path,omitempty"`
    JSONPathExpr         string               `json:"jsonpath_expr,omitempty" yaml:"jsonpath_expr,omitempty"`
    DictionaryFilter     *DictionaryFilter    `json:"dictionary_filter,omitempty" yaml:"dictionary_filter,omitempty"`
    LLMFalsePositiveTest *LLMValidationPrompt `json:"llm_false_positive_test,omitempty" yaml:"llm_false_positive_test,omitempty"`
}
```

**After**:
```go
type ExtractionRule struct {
    // Universal addressing (OPTIONAL)
    JSONPath         string                   `json:"jsonpath,omitempty" yaml:"jsonpath,omitempty"`

    // Extraction methods (choose ONE)
    PhraseList       []string                 `json:"phrase_list,omitempty" yaml:"phrase_list,omitempty"`
    InstanceName     string                   `json:"instance_name,omitempty" yaml:"instance_name,omitempty"`

    // Pre-filters (all OPTIONAL, AND logic, fail-fast)
    Pattern          string                   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
    Proximity        *ProximityFilter         `json:"proximity,omitempty" yaml:"proximity,omitempty"`
    Dictionary       *DictionaryFilter        `json:"dictionary,omitempty" yaml:"dictionary,omitempty"`
    Semantic         *SemanticFilter          `json:"semantic,omitempty" yaml:"semantic,omitempty"`
    LLMValidation    *LLMValidationPrompt     `json:"llm_validation,omitempty" yaml:"llm_validation,omitempty"`
}
```

**Update validation** (lines 521-580):

**Remove Type-based switch**:
```go
// DELETE THIS SECTION
switch rule.Type {
case RuleTypeContent:
    // ...
case RuleTypeMetadata:
    // ...
case RuleTypeJSONPath:
    // ...
}
```

**Add new validation**:
```go
// At least one extraction method required
hasExtractionMethod := len(rule.PhraseList) > 0 || rule.InstanceName != ""
if !hasExtractionMethod {
    return NewValidationError(fmt.Sprintf("entity mapping %d (%s), rule %d: must have phrase_list or instance_name", i, mapping.EntityType, j))
}

// If instance_name used, must have (?P<name>...) capture group
if rule.InstanceName != "" {
    if !strings.Contains(rule.InstanceName, "(?P<name>") {
        return NewValidationError(fmt.Sprintf("entity mapping %d (%s), rule %d: instance_name must contain (?P<name>...) capture group", i, mapping.EntityType, j))
    }
}

// Validate filters if present (existing validation logic for filters)
// ...
```

### Estimated Changes

- Delete: 50 lines
- Add: 10 lines
- **Net: -40 lines**

---

## Step 3: Add Hierarchy Computation & Validation

### File

`go/internal/udml/ontology/types.go`

### Changes

**Update `Validate()` method** (add after line 583):

```go
func (s *OntologySchema) Validate() error {
    // ... existing validation ...
    
    // NEW: Compute bidirectional hierarchies
    if err := s.ComputeHierarchies(); err != nil {
        return fmt.Errorf("hierarchy computation failed: %w", err)
    }
    
    // NEW: Validate hierarchies
    if err := s.ValidateHierarchies(); err != nil {
        return fmt.Errorf("hierarchy validation failed: %w", err)
    }
    
    return nil
}
```

**Add new methods** (at end of file):

```go
// ComputeHierarchies fills missing parent_type and children relationships
func (s *OntologySchema) ComputeHierarchies() error {
    // Build entity map (first occurrence per qualified name)
    entityMap := make(map[string]*ElementEntityMapping)
    for i := range s.ElementEntityMappings {
        mapping := &s.ElementEntityMappings[i]
        key := mapping.Domain + "." + mapping.EntityType
        
        if _, exists := entityMap[key]; !exists {
            entityMap[key] = mapping
        }
    }
    
    // Phase 1: Fill parent's children from child's parent_type
    for i := range s.ElementEntityMappings {
        mapping := &s.ElementEntityMappings[i]
        
        if mapping.ParentType != "" {
            parent := entityMap[mapping.ParentType]
            if parent != nil {
                childRef := mapping.Domain + "." + mapping.EntityType
                if !contains(parent.Children, childRef) {
                    parent.Children = append(parent.Children, childRef)
                }
            }
        }
    }
    
    // Phase 2: Fill child's parent_type from parent's children
    for i := range s.ElementEntityMappings {
        mapping := &s.ElementEntityMappings[i]
        
        for _, childRef := range mapping.Children {
            child := entityMap[childRef]
            if child != nil && child.ParentType == "" {
                child.ParentType = mapping.Domain + "." + mapping.EntityType
            }
        }
    }
    
    return nil
}

// ValidateHierarchies checks for broken references and circular hierarchies
func (s *OntologySchema) ValidateHierarchies() error {
    // Build entity existence map
    entityMap := make(map[string]bool)
    for _, mapping := range s.ElementEntityMappings {
        key := mapping.Domain + "." + mapping.EntityType
        entityMap[key] = true
    }
    
    // Check for broken references
    for _, mapping := range s.ElementEntityMappings {
        qualifiedName := mapping.Domain + "." + mapping.EntityType
        
        // Check parent exists
        if mapping.ParentType != "" && !entityMap[mapping.ParentType] {
            return fmt.Errorf("entity %s references non-existent parent: %s", 
                qualifiedName, mapping.ParentType)
        }
        
        // Check children exist
        for _, childRef := range mapping.Children {
            if !entityMap[childRef] {
                return fmt.Errorf("entity %s references non-existent child: %s",
                    qualifiedName, childRef)
            }
        }
    }
    
    // Check for circular hierarchies
    for _, mapping := range s.ElementEntityMappings {
        qualifiedName := mapping.Domain + "." + mapping.EntityType
        if s.hasCycle(qualifiedName, make(map[string]bool)) {
            return fmt.Errorf("circular hierarchy detected involving: %s", qualifiedName)
        }
    }
    
    return nil
}

// hasCycle detects circular parent references
func (s *OntologySchema) hasCycle(entityName string, visited map[string]bool) bool {
    if visited[entityName] {
        return true // Cycle detected
    }
    
    // Find entity mapping
    var parentType string
    for _, mapping := range s.ElementEntityMappings {
        if mapping.Domain+"."+mapping.EntityType == entityName {
            parentType = mapping.ParentType
            break
        }
    }
    
    if parentType == "" {
        return false // Reached top
    }
    
    visited[entityName] = true
    result := s.hasCycle(parentType, visited)
    delete(visited, entityName) // Backtrack
    
    return result
}

// Helper: contains checks if string slice contains value
func contains(slice []string, value string) bool {
    for _, item := range slice {
        if item == value {
            return true
        }
    }
    return false
}
```

### Estimated Changes

**+150 lines**

---

## Step 4: Unify Extraction Logic

### File

`go/internal/udml/ontology/extractor.go`

### Changes

**Replace `tryExtractWithRule()`** (lines 206-222):

**Before** (Type-based dispatch):
```go
func (e *RuleBasedExtractor) tryExtractWithRule(ctx context.Context, mapping ElementEntityMapping, rule ExtractionRule, elem Element) *Entity {
    switch rule.Type {
    case RuleTypeContent:
        // Not implemented
        return nil
    case RuleTypeMetadata:
        return e.tryExtractWithMetadata(mapping, rule, elem)
    case RuleTypeJSONPath:
        return e.tryExtractWithJSONPath(mapping, rule, elem)
    default:
        return nil
    }
}
```

**After** (Unified pipeline):
```go
func (e *RuleBasedExtractor) tryExtractWithRule(ctx context.Context, mapping ElementEntityMapping, rule ExtractionRule, elem Element) *Entity {
    // STEP 1: JSONPath validation (OPTIONAL - instant if omitted)
    var sourceData string
    if rule.JSONPath != "" {
        elemJSON := e.elementToJSON(&elem)
        results := e.evaluateJSONPath(elemJSON, rule.JSONPath)
        if len(results) == 0 {
            return nil // FAIL FAST
        }
        sourceData = fmt.Sprint(results[0])
    } else {
        // Fast path: extract from content directly
        sourceData = e.resolveContent(&elem)
    }
    
    if sourceData == "" {
        return nil // FAIL FAST
    }
    
    // STEP 2: Phrase list matching (OPTIONAL - 10-100x faster than regex)
    var entityName string
    if len(rule.PhraseList) > 0 {
        entityName = e.findPhraseMatch(sourceData, rule.PhraseList)
        if entityName == "" {
            return nil // FAIL FAST
        }
    } else if rule.InstanceName != "" {
        // STEP 3: Instance name extraction (regex fallback)
        entityName = e.extractInstanceName(sourceData, rule, sourceData)
        if entityName == "" {
            return nil // FAIL FAST
        }
    } else {
        return nil // No extraction method
    }
    
    // STEP 4: Pattern filter (OPTIONAL - cheap regex)
    if rule.Pattern != "" {
        matched, _ := regexp.MatchString(rule.Pattern, sourceData)
        if !matched {
            return nil // FAIL FAST
        }
    }
    
    // STEP 5: Proximity filter (OPTIONAL - moderate cost)
    if rule.Proximity != nil {
        if !e.checkProximityFilter(sourceData, entityName, rule.Proximity) {
            return nil // FAIL FAST
        }
    }
    
    // STEP 6: Dictionary filter (OPTIONAL - moderate cost)
    if rule.Dictionary != nil {
        if !e.checkDictionaryFilter(entityName, rule.Dictionary) {
            return nil // FAIL FAST
        }
    }
    
    // STEP 7: Semantic filter (OPTIONAL - expensive)
    if rule.Semantic != nil {
        if !e.checkSemanticFilter(&elem, rule.Semantic) {
            return nil // FAIL FAST
        }
    }
    
    // All filters passed - create entity
    return &Entity{
        ID:         e.generateID("ent"),
        Name:       entityName,
        Type:       EntityType(mapping.EntityType),
        Domain:     mapping.Domain,
        Confidence: mapping.Confidence,
        Attributes: map[string]interface{}{
            "w_category": mapping.WCategory,
            "jsonpath":   rule.JSONPath,
        },
        ElementID: elem.ElementID,
        Mentions: []Mention{
            {ElementID: elem.ElementID, Text: entityName, StartPos: 0, EndPos: len(entityName)},
        },
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}
```

**Delete old methods**:
- `tryExtractWithMetadata()` (lines 226-260)
- `tryExtractWithJSONPath()` (lines 262-336)

**Add new helper methods**:

```go
// elementToJSON converts Element to JSON structure for JSONPath queries
func (e *RuleBasedExtractor) elementToJSON(elem *Element) map[string]interface{} {
    return map[string]interface{}{
        "element_id":       elem.ElementID,
        "element_type":     elem.ElementType,
        "content":          e.resolveContent(elem),
        "content_preview":  elem.ContentPreview,
        "content_location": elem.ContentLocation,
        "parent_id":        elem.ParentID,
        "element_order":    elem.ElementOrder,
        "metadata":         elem.Metadata,
    }
}

// findPhraseMatch performs exact string matching (10-100x faster than regex)
func (e *RuleBasedExtractor) findPhraseMatch(content string, phrases []string) string {
    contentLower := strings.ToLower(content)
    
    var longestMatch string
    for _, phrase := range phrases {
        phraseLower := strings.ToLower(phrase)
        if strings.Contains(contentLower, phraseLower) {
            if len(phrase) > len(longestMatch) {
                longestMatch = phrase
            }
        }
    }
    
    return longestMatch
}

// checkProximityFilter validates entity appears near keywords
func (e *RuleBasedExtractor) checkProximityFilter(content string, entityName string, filter *ProximityFilter) bool {
    // Implementation: check if any keyword appears within max_distance of entity
    // For now, simple contains check
    for _, keyword := range filter.Keywords {
        if strings.Contains(strings.ToLower(content), strings.ToLower(keyword)) {
            return true
        }
    }
    return false
}
```

### Estimated Changes

- Delete: 220 lines (old methods)
- Add: 180 lines (unified + helpers)
- **Net: -40 lines**

---

## Step 5: Add Canonicalization and Hierarchy Materialization

### File

`go/internal/udml/ontology/extractor.go`

### Changes

**Add canonicalization method** (at end of file):

```go
// canonicalizeEntities deduplicates extracted entities
// Key: name|type|element_id
// Keeps highest confidence, merges mentions
func (e *RuleBasedExtractor) canonicalizeEntities(extractedEntities []Entity) ([]Entity, error) {
    // Canonicalization key: name|type|element_id
    canonicalIndex := make(map[string]*Entity)

    for _, entity := range extractedEntities {
        key := entity.Name + "|" + string(entity.Type) + "|" + entity.ElementID

        if canonical, exists := canonicalIndex[key]; exists {
            // Merge: Keep highest confidence, combine mentions
            if entity.Confidence > canonical.Confidence {
                canonical.Confidence = entity.Confidence
            }
            // Merge mentions from both entities
            canonical.Mentions = append(canonical.Mentions, entity.Mentions...)
        } else {
            // First occurrence - make canonical
            entityCopy := entity
            canonicalIndex[key] = &entityCopy
        }
    }

    // Convert map to slice
    var canonicalEntities []Entity
    for _, entity := range canonicalIndex {
        canonicalEntities = append(canonicalEntities, *entity)
    }

    return canonicalEntities, nil
}
```

**Add materialization method** (at end of file):

```go
// MaterializeMissingAncestors creates composite parent entities and IS-A relationships
// Called AFTER canonicalization
// Only works with canonical entities (no duplicates)
func (e *RuleBasedExtractor) MaterializeMissingAncestors(canonicalLeaves []Entity, schema *OntologySchema) ([]Entity, []Relationship, error) {
    var compositeEntities []Entity
    var isaRelationships []Relationship
    
    // Build entity type map for lookup
    entityTypeMap := make(map[string]*ElementEntityMapping)
    for i := range schema.ElementEntityMappings {
        mapping := &schema.ElementEntityMappings[i]
        key := mapping.Domain + "." + mapping.EntityType
        if _, exists := entityTypeMap[key]; !exists {
            entityTypeMap[key] = mapping
        }
    }

    // Canonical index for composites (prevent duplicates during materialization)
    // Key: name|type|element_id
    canonicalIndex := make(map[string]*Entity)

    // For each canonical leaf entity, walk up parent chain
    for _, leafEntity := range canonicalLeaves {
        currentEntity := leafEntity
        currentKey := leafEntity.Domain + "." + string(leafEntity.Type)

        // Walk up hierarchy
        for {
            mapping := entityTypeMap[currentKey]
            if mapping == nil || mapping.ParentType == "" {
                break // Reached top or orphan
            }

            parentMapping := entityTypeMap[mapping.ParentType]
            if parentMapping == nil {
                break // Invalid parent
            }

            // Check if parent already exists in canonical index
            canonicalKey := currentEntity.Name + "|" + parentMapping.EntityType + "|" + currentEntity.ElementID

            var parentEntity *Entity
            if cached, exists := canonicalIndex[canonicalKey]; exists {
                // Reuse existing canonical composite
                parentEntity = cached
            } else {
                // Materialize new canonical composite (synthesized entity)
                parentEntity = &Entity{
                    ID:         e.generateID("ent"),
                    Name:       currentEntity.Name,  // Inherit name from child
                    Type:       EntityType(parentMapping.EntityType),
                    Domain:     parentMapping.Domain,
                    Attributes: map[string]interface{}{
                        "synthesized": true,  // Created during materialization (not extracted)
                        "w_category":  parentMapping.WCategory,
                    },
                    ElementID:  currentEntity.ElementID,
                    Mentions:   currentEntity.Mentions,  // Inherit mentions
                    CreatedAt:  time.Now(),
                    UpdatedAt:  time.Now(),
                }
                compositeEntities = append(compositeEntities, *parentEntity)
                canonicalIndex[canonicalKey] = parentEntity
            }

            // Create IS-A relationship: child IS-A parent (only between canonical entities)
            isaRelationships = append(isaRelationships, Relationship{
                ID:         e.generateID("rel"),
                Type:       RelationshipIsA,
                Domain:     currentEntity.Domain,
                SourceID:   currentEntity.ID,
                TargetID:   parentEntity.ID,
                Confidence: 1.0,
                Attributes: map[string]interface{}{
                    "hierarchy_level": "direct_parent",
                },
                ElementID: currentEntity.ElementID,
                Evidence:  fmt.Sprintf("%s is a %s", currentEntity.Type, parentEntity.Type),
                CreatedAt: time.Now(),
            })

            // Move up hierarchy
            currentEntity = *parentEntity
            currentKey = mapping.ParentType
        }
    }

    return compositeEntities, isaRelationships, nil
}
```

**Update `ExtractFromElements()`** (lines 88-135):

**Before**:
```go
func (e *RuleBasedExtractor) ExtractFromElements(ctx context.Context, docID string, elements []Element) (*Ontology, error) {
    // ...

    // Extract entities using schema rules
    entities, err := e.extractEntities(ctx, elements)
    if err != nil {
        return nil, fmt.Errorf("entity extraction failed: %w", err)
    }
    ontology.Entities = entities

    // Extract relationships using schema rules
    relationships, err := e.extractRelationships(ctx, elements, entities)
    if err != nil {
        return nil, fmt.Errorf("relationship extraction failed: %w", err)
    }
    ontology.Relationships = relationships

    // ...
}
```

**After** (canonical-only flow):
```go
func (e *RuleBasedExtractor) ExtractFromElements(ctx context.Context, docID string, elements []Element) (*Ontology, error) {
    // ...

    // STEP 1: Extract entities (with duplicates from multiple definitions)
    extractedEntities, err := e.extractEntities(ctx, elements)
    if err != nil {
        return nil, fmt.Errorf("entity extraction failed: %w", err)
    }

    // STEP 2: Canonicalize (deduplicate, merge mentions, keep highest confidence)
    canonicalLeaves, err := e.canonicalizeEntities(extractedEntities)
    if err != nil {
        return nil, fmt.Errorf("canonicalization failed: %w", err)
    }

    // STEP 3: Materialize missing ancestors (walk up, create composites as needed)
    compositeAncestors, isaRelationships, err := e.MaterializeMissingAncestors(canonicalLeaves, e.schema)
    if err != nil {
        return nil, fmt.Errorf("hierarchy materialization failed: %w", err)
    }

    // STEP 4: Combine canonical leaves + canonical composites
    allCanonicalEntities := append(canonicalLeaves, compositeAncestors...)
    ontology.Entities = allCanonicalEntities

    // STEP 5: Extract domain relationships (between canonical entities only)
    domainRelationships, err := e.extractRelationships(ctx, elements, allCanonicalEntities)
    if err != nil {
        return nil, fmt.Errorf("relationship extraction failed: %w", err)
    }

    // STEP 6: Combine IS-A + domain relationships
    ontology.Relationships = append(isaRelationships, domainRelationships...)

    // ...
}
```

### Key Changes

**Canonical-only graph architecture**:
1. Extract → produces duplicates (multiple definitions)
2. Canonicalize → deduplicate, merge mentions, keep highest confidence
3. Materialize → only for canonical entities (no duplicates)
4. Final graph → ALL entities are canonical (extracted or materialized)

### Estimated Changes

**+180 lines** (canonicalization + updated materialization)

---

## Step 5.5: Update Relationship Extraction to Filter Synthesized Entities

### File

`go/internal/udml/ontology/extractor.go`

### Changes

**Update `extractRelationships()` method** to only create relationships between extracted entities:

```go
func (e *RuleBasedExtractor) extractRelationships(ctx context.Context, elements []Element, entities []Entity) ([]Relationship, error) {
    // Build index of extracted entities (synthesized=false)
    extractedIndex := make(map[string]*Entity)
    for i := range entities {
        entity := &entities[i]
        // Only include entities that were extracted (not synthesized)
        if synthesized, ok := entity.Attributes["synthesized"].(bool); !ok || !synthesized {
            extractedIndex[entity.ID] = entity
        }
    }

    var relationships []Relationship

    // For each relationship rule in schema...
    for _, relRule := range e.schema.RelationshipRules {
        for _, elem := range elements {
            // Try to find source and target entities
            sourceEntity := findEntityByRule(elem, relRule.SourcePattern, extractedIndex)
            targetEntity := findEntityByRule(elem, relRule.TargetPattern, extractedIndex)

            // Only create relationship if BOTH are extracted (not synthesized)
            if sourceEntity != nil && targetEntity != nil {
                relationship := Relationship{
                    ID:         e.generateID("rel"),
                    Type:       RelationshipType(relRule.RelationshipType),
                    Domain:     relRule.Domain,
                    SourceID:   sourceEntity.ID,
                    TargetID:   targetEntity.ID,
                    Confidence: relRule.Confidence,
                    ElementID:  elem.ElementID,
                    Evidence:   elem.ContentPreview,
                    CreatedAt:  time.Now(),
                }
                relationships = append(relationships, relationship)
            }
        }
    }

    return relationships, nil
}
```

### Design Principle

**Domain relationships ONLY between extracted entities** (synthesized=false)

**Rationale**:
- Relationships should reflect what was actually extracted from text
- Synthesized entities exist only to complete the hierarchy
- Creating relationships involving synthesized entities creates noise
- Hierarchical queries can traverse IS-A relationships when needed

### Example

**Before filtering** (relationships created with all entities):
```
✅ surgeon["Dr. Smith"] WORKS_AT hospital["City Hospital"]  // both extracted
❌ person["Dr. Smith"] WORKS_AT hospital["City Hospital"]   // person is synthesized
❌ surgeon["Dr. Smith"] WORKS_AT organization["City Hospital"]  // organization is synthesized
❌ person["Dr. Smith"] WORKS_AT organization["City Hospital"]  // both synthesized
```

**After filtering** (relationships only with extracted entities):
```
✅ surgeon["Dr. Smith"] WORKS_AT hospital["City Hospital"]  // both extracted
```

### Estimated Changes

**+30 lines** (filtering logic in extractRelationships)

---

## Step 6: Update Domain Catalogs

### Files

All YAML files in `examples/ontologies/` (36 files)

### Changes

For each catalog file:

1. **Remove `type` field** from all extraction rules
2. **Convert metadata extraction**:
   - `type: metadata_field, field_path: author.name` → `jsonpath: $.metadata.author.name`
3. **Convert JSONPath extraction**:
   - `type: jsonpath_query, jsonpath_expr: $.content.path` → `jsonpath: $.content.path`
4. **Rename filter fields**:
   - `proximity_filter` → `proximity`
   - `cooccurrence_terms` → `keywords`
   - `semantic_filter` → `semantic`
   - `dictionary_filter` → `dictionary`
   - `llm_false_positive_test` → `llm_validation`
5. **Add `parent_type`** to entities extending global types:
   - Example: `physician` → `parent_type: global.person`
6. **Add `w_category`** to all entities

### Example

**Before**:
```yaml
entity_types:
  - entity_type: physician
    domain: medical
    description: Medical doctor
    confidence: 0.90
    extraction_rules:
      - type: content_extraction
        instance_name: (?P<name>Dr\. .+)
        proximity_filter:
          cooccurrence_terms: [patient, diagnosis]
          max_distance: 100
        semantic_filter:
          reference_concepts: [medical practice]
          similarity_threshold: 0.70
```

**After**:
```yaml
entity_types:
  - entity_type: physician
    domain: medical
    parent_type: global.person    # NEW
    w_category: who               # NEW
    description: Medical doctor
    confidence: 0.90
    extraction_rules:
      - instance_name: (?P<name>Dr\. .+)
        proximity:                # RENAMED
          keywords: [patient, diagnosis]  # RENAMED
          max_distance: 100
        semantic:                 # RENAMED
          reference_concepts: [medical practice]
          similarity_threshold: 0.70
```

### Estimated Changes

**~200 lines across 36 files** (~5-6 lines per file average)

---

## Step 6.2: Update Domain Catalog Relationship Rules

### Files

All YAML files in `examples/ontologies/` with relationship definitions (~20 files)

### Changes

For each catalog file with `relationship_types`:

1. **Rename field**: `relationship_types` → `entity_relationship_rules`
2. **Add `name` field**: Unique identifier for each rule
3. **Rename entity fields**:
   - `source_entity` → `source_entity_type`
   - `target_entity` → `target_entity_type`
4. **Move relationship_type**: From field name to dedicated field
5. **Add `confidence` field**: Pattern reliability score (0.0-1.0)
6. **Update extraction patterns**:
   - `sample_rules` → `extraction_patterns`
   - `type: keyword_match` → `type: proximity`
   - `keywords` → `signal_words`
   - Add `max_distance` and `direction` fields

### Example: Pharmaceutical Domain

**Before**:
```yaml
relationship_types:
  - relationship_type: treats_indication
    description: Drug treats indication
    source_entity: drug_product
    target_entity: indication
    sample_rules:
      - type: keyword_match
        keywords: [treats, indicated for, approved for]

  - relationship_type: causes_adverse_event
    description: Drug causes adverse event
    source_entity: drug_product
    target_entity: adverse_event
    sample_rules:
      - type: keyword_match
        keywords: [causes, associated with, side effect]
```

**After**:
```yaml
entity_relationship_rules:
  - name: drug_treats_indication
    source_entity_type: drug_product
    target_entity_type: indication
    relationship_type: treats
    description: Drug treats indication
    confidence: 0.85
    extraction_patterns:
      - type: proximity
        signal_words: [treats, indicated for, approved for, therapy for]
        max_distance: 50
        direction: forward

  - name: drug_causes_adverse_event
    source_entity_type: drug_product
    target_entity_type: adverse_event
    relationship_type: causes
    description: Drug causes adverse event
    confidence: 0.75
    extraction_patterns:
      - type: proximity
        signal_words: [causes, associated with, side effect, adverse event]
        max_distance: 100
        direction: forward
```

### Automated Migration Script

```bash
#!/bin/bash
# Apply automated renames to domain catalogs

for file in examples/ontologies/**/*.yaml; do
    echo "Processing $file..."

    # Rename top-level field
    sed -i '' 's/^relationship_types:/entity_relationship_rules:/g' "$file"

    # Rename entity fields
    sed -i '' 's/source_entity:/source_entity_type:/g' "$file"
    sed -i '' 's/target_entity:/target_entity_type:/g' "$file"

    # Rename sample_rules
    sed -i '' 's/sample_rules:/extraction_patterns:/g' "$file"

    # Update pattern types
    sed -i '' 's/type: keyword_match/type: proximity/g' "$file"

    # Rename keywords
    sed -i '' 's/keywords:/signal_words:/g' "$file"
done

echo "Automated migration complete. Manual updates still required:"
echo "  - Add 'name' field to each rule"
echo "  - Add 'confidence' field (0.60-0.95)"
echo "  - Add 'max_distance' to proximity patterns"
echo "  - Add 'direction' to proximity patterns"
```

### Manual Updates Required

After automated script, manually add:

1. **`name` field**: Create unique identifier from relationship type and entities
   - Example: `drug_treats_indication`, `physician_works_at_hospital`
2. **`confidence` field**: Estimate based on pattern specificity:
   - High confidence (0.85-0.95): Specific signal words, structured patterns
   - Medium confidence (0.70-0.84): Moderate signal words, some ambiguity
   - Low confidence (0.60-0.69): Broad co-occurrence, statistical patterns
3. **`max_distance`**: Token distance for proximity patterns:
   - Close proximity (20-50): Strong relationships (e.g., "treats", "authored by")
   - Medium proximity (50-100): Moderate relationships (e.g., "related to")
   - Far proximity (100-200): Loose relationships (e.g., "associated with")
4. **`direction`**: Relationship directionality:
   - `forward`: Source entity appears before target
   - `backward`: Target entity appears before source
   - `bidirectional`: Either order acceptable

### Estimated Changes

**~150 lines across 20 files** (~7-8 lines per file average)

**See also**: [09_RELATIONSHIP_RULES.md](09_RELATIONSHIP_RULES.md) for comprehensive relationship rule documentation

---

## Step 7: Update Catalog Loader

### File

`go/internal/udml/ontology/catalogs/loader.go`

### Changes

**Update `ExtractionRuleConfig` struct**:

**Before**:
```go
type ExtractionRuleConfig struct {
    Type             string                     `yaml:"type"`
    InstanceName     string                     `yaml:"instance_name,omitempty"`
    Pattern          string                     `yaml:"pattern,omitempty"`
    FieldPath        string                     `yaml:"field_path,omitempty"`
    JSONPathExpr     string                     `yaml:"jsonpath_expr,omitempty"`
    ProximityFilter  *ProximityFilterConfig     `yaml:"proximity_filter,omitempty"`
    SemanticFilter   *SemanticFilterConfig      `yaml:"semantic_filter,omitempty"`
    DictionaryFilter *DictionaryFilterConfig    `yaml:"dictionary_filter,omitempty"`
    LLMFalsePositiveTest *LLMValidationPromptConfig `yaml:"llm_false_positive_test,omitempty"`
}
```

**After**:
```go
type ExtractionRuleConfig struct {
    JSONPath      string                     `yaml:"jsonpath,omitempty"`
    PhraseList    []string                   `yaml:"phrase_list,omitempty"`
    InstanceName  string                     `yaml:"instance_name,omitempty"`
    Pattern       string                     `yaml:"pattern,omitempty"`
    Proximity     *ProximityFilterConfig     `yaml:"proximity,omitempty"`
    Semantic      *SemanticFilterConfig      `yaml:"semantic,omitempty"`
    Dictionary    *DictionaryFilterConfig    `yaml:"dictionary,omitempty"`
    LLMValidation *LLMValidationPromptConfig `yaml:"llm_validation,omitempty"`
}
```

**Update conversion functions**:

```go
func convertToOntologyRule(config ExtractionRuleConfig) ontology.ExtractionRule {
    return ontology.ExtractionRule{
        JSONPath:      config.JSONPath,
        PhraseList:    config.PhraseList,
        InstanceName:  config.InstanceName,
        Pattern:       config.Pattern,
        Proximity:     convertProximityFilter(config.Proximity),
        Semantic:      convertSemanticFilter(config.Semantic),
        Dictionary:    convertDictionaryFilter(config.Dictionary),
        LLMValidation: convertLLMValidation(config.LLMValidation),
    }
}

func convertFromOntologyRule(rule ontology.ExtractionRule) ExtractionRuleConfig {
    return ExtractionRuleConfig{
        JSONPath:      rule.JSONPath,
        PhraseList:    rule.PhraseList,
        InstanceName:  rule.InstanceName,
        Pattern:       rule.Pattern,
        Proximity:     convertProximityFilterToConfig(rule.Proximity),
        Semantic:      convertSemanticFilterToConfig(rule.Semantic),
        Dictionary:    convertDictionaryFilterToConfig(rule.Dictionary),
        LLMValidation: convertLLMValidationToConfig(rule.LLMValidation),
    }
}
```

### Estimated Changes

**~40 lines modified**

---

## Step 8: Add Unit Tests

### New File

`go/internal/udml/ontology/types_test.go`

### Tests

```go
package ontology

import (
    "testing"
)

func TestComputeHierarchies_BidirectionalFill(t *testing.T) {
    schema := &OntologySchema{
        ElementEntityMappings: []ElementEntityMapping{
            {
                EntityType: "person",
                Domain:     "global",
                ParentType: "",
                Children:   []string{}, // Empty, should be filled
            },
            {
                EntityType: "physician",
                Domain:     "medical",
                ParentType: "global.person", // Declares parent
                Children:   []string{},
            },
        },
    }

    err := schema.ComputeHierarchies()
    if err != nil {
        t.Fatalf("ComputeHierarchies failed: %v", err)
    }

    // Check parent's children was auto-filled
    person := findMapping(schema, "global", "person")
    if !contains(person.Children, "medical.physician") {
        t.Errorf("Expected person.Children to contain medical.physician, got %v", person.Children)
    }
}

func TestValidateHierarchies_BrokenParentReference(t *testing.T) {
    schema := &OntologySchema{
        ElementEntityMappings: []ElementEntityMapping{
            {
                EntityType: "physician",
                Domain:     "medical",
                ParentType: "global.foobar", // Broken reference
            },
        },
    }

    err := schema.ValidateHierarchies()
    if err == nil {
        t.Error("Expected validation error for broken parent reference")
    }
}

func TestValidateHierarchies_CircularDetection(t *testing.T) {
    schema := &OntologySchema{
        ElementEntityMappings: []ElementEntityMapping{
            {
                EntityType: "physician",
                Domain:     "medical",
                ParentType: "medical.surgeon",
            },
            {
                EntityType: "surgeon",
                Domain:     "medical",
                ParentType: "medical.physician", // Circular!
            },
        },
    }

    err := schema.ValidateHierarchies()
    if err == nil {
        t.Error("Expected validation error for circular hierarchy")
    }
}

// ... more tests
```

### Update File

`go/internal/udml/ontology/extractor_test.go`

### Tests

```go
func TestMaterializeHierarchy_SimpleChain(t *testing.T) {
    schema := &OntologySchema{
        ElementEntityMappings: []ElementEntityMapping{
            {EntityType: "person", Domain: "global", ParentType: "", Confidence: 0.80},
            {EntityType: "physician", Domain: "medical", ParentType: "global.person", Confidence: 0.85},
        },
    }

    extractor := NewRuleBasedExtractor(schema, nil)
    
    leafEntities := []Entity{
        {
            ID:   "ent_1",
            Name: "Dr. Smith",
            Type: "physician",
            Domain: "medical",
        },
    }

    composites, rels, err := extractor.MaterializeHierarchy(leafEntities, schema)
    if err != nil {
        t.Fatalf("MaterializeHierarchy failed: %v", err)
    }

    // Should create 1 composite (person)
    if len(composites) != 1 {
        t.Errorf("Expected 1 composite entity, got %d", len(composites))
    }

    // Should create 1 IS-A relationship
    if len(rels) != 1 {
        t.Errorf("Expected 1 IS-A relationship, got %d", len(rels))
    }

    // Check composite entity
    if composites[0].Type != "person" {
        t.Errorf("Expected composite type 'person', got %s", composites[0].Type)
    }
    if composites[0].Name != "Dr. Smith" {
        t.Errorf("Expected composite name 'Dr. Smith', got %s", composites[0].Name)
    }
}

// ... more tests
```

### Estimated Changes

**+400 lines** (new test file + additions to existing test file)

---

## Step 9: Update Builder LLM Prompts

### File

`go/internal/udml/ontology/builder.go`

### Changes

**Load global domain at initialization**:

```go
func NewOntologyBuilder(config BuilderConfig) (*OntologyBuilder, error) {
    // ... existing code ...
    
    // Load global domain catalogs
    globalCatalogs := []string{
        "global_who.go",
        "global_what.go",
        "global_where.go",
        "global_when.go",
        "global_why.go",
    }
    
    // ... register global entity types ...
    
    return builder, nil
}
```

**Update LLM prompts** to include global types:

```go
func (b *OntologyBuilder) buildEntityMappingPrompt(domain string, samples []Sample) string {
    prompt := fmt.Sprintf(`You are designing entity extraction rules for the %s domain.

GLOBAL ENTITY TYPES (available for extension):
- person (who) - Individual human being
  - Subtypes: public_figure, executive, employee
- organization (who) - Company, institution, agency
  - Subtypes: business, nonprofit, government, educational, healthcare, religious, media
- document (what) - Referenced documents, reports, papers
- identifier (what) - Generic identifier or reference code
  - Subtypes: email, phone, url, code, id_number
- role (what) - Job title, position, functional role
- location (where) - Geographic place
  - Subtypes: city, country, region, address, building
- date (when) - Calendar date or temporal reference
  - Subtypes: time, duration
- event (when) - Occurrence with start/end times
- assertion (why) - Claims, requirements, declarations
- hypothesis (why) - Testable explanations, theories

INSTRUCTIONS:
1. Review sample content from %s domain corpus
2. For each entity type:
   a) If extending global type, set parent_type (e.g., parent_type: global.person)
   b) Otherwise, create domain-specific type
   c) Assign w_category (who/what/where/when/why)
   d) Create extraction rules using unified structure:
      - NO type field (removed)
      - Use phrase_list for exact matching (fast)
      - Use instance_name for regex with (?P<name>...) capture
      - Add filters: proximity, semantic, dictionary (all optional)

EXTRACTION RULE STRUCTURE (unified, no type field):
{
  "instance_name": "(?P<name>regex pattern)",
  "phrase_list": ["exact phrase 1", "exact phrase 2"],  // OR within list
  "pattern": "pre-filter regex",
  "proximity": {
    "keywords": ["term1", "term2"],  // OR within keywords
    "max_distance": 100
  },
  "semantic": {
    "reference_concepts": ["concept1", "concept2"],  // OR within concepts
    "similarity_threshold": 0.70
  },
  "dictionary": {
    "require_unknown_words": true
  }
}

Sample content:
%s

Generate entity mappings in JSON format.
`, domain, domain, formatSamples(samples))
    
    return prompt
}
```

### Estimated Changes

**+150 lines** (prompt updates + global catalog loading)

---

## Summary of Changes

| Step | Component | File(s) | Change Type | Lines |
|------|-----------|---------|-------------|-------|
| 0 | Global catalogs | catalogs/global_*.go | NEW (5), DELETE (1) | +70 |
| 1 | Schema types | types.go | ADD fields | +3 |
| 2 | Extraction rules | types.go | REMOVE Type, UPDATE | -40 |
| 3 | Hierarchy logic | types.go | ADD validation | +150 |
| 4 | Extractor | extractor.go | UNIFY extraction | -40 |
| 5 | Materialization | extractor.go | ADD hierarchy | +150 |
| 6 | Domain catalogs | examples/ontologies/*.yaml | CONVERT (36) | +200 |
| 7 | Catalog loader | catalogs/loader.go | UPDATE schema | +40 |
| 8 | Unit tests | *_test.go | NEW tests | +400 |
| 9 | Builder | builder.go | UPDATE prompts | +150 |
| **TOTAL** | | | | **~1,083** |

---

## Validation Checklist

After each step:

- [ ] Code compiles: `go build ./...`
- [ ] Tests pass: `go test ./...`
- [ ] Lint passes: `golangci-lint run ./...`
- [ ] Format check: `gofmt -l .` (should be empty)

After all steps:

- [ ] Integration test passes
- [ ] Schema validation works
- [ ] Hierarchy computation correct
- [ ] Materialization generates expected entities
- [ ] Domain catalogs load successfully

---

## Next Steps

Proceed to:
- [07_TESTING_STRATEGY.md](07_TESTING_STRATEGY.md) - Testing procedures
- [08_MIGRATION_GUIDE.md](08_MIGRATION_GUIDE.md) - Catalog updates
