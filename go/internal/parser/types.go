package parser

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Universal Document Model Types
// These types are shared across ALL parsers to maintain consistency

// Document represents the top-level document metadata
type Document struct {
	ID       string                 `json:"id"`
	DocType  string                 `json:"doc_type"`
	Title    string                 `json:"title,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Element represents a universal document element that works across all formats
// Updated with UDML Phase 1: 6 query-optimized promoted fields for common access patterns
type Element struct {
	// Core universal fields
	ElementID       string                 `json:"element_id" parquet:"element_id"`
	ElementType     string                 `json:"element_type" parquet:"element_type"`
	ElementCategory string                 `json:"element_category" parquet:"element_category"`
	Content         string                 `json:"content,omitempty" parquet:"content"`
	ContentPreview  string                 `json:"content_preview" parquet:"content_preview"`
	ParentID        string                 `json:"parent_id,omitempty" parquet:"parent_id"`
	Position        int                    `json:"position" parquet:"position"`
	Depth           int                    `json:"depth" parquet:"depth"`

	// UDML Phase 1: Query-Optimized Promoted Fields (nullable, 70-95% NULL is acceptable)
	// These fields enable 60-1000x faster queries across all backends (Parquet, PostgreSQL, Neo4j, Elasticsearch)
	PageNumber    *int    `json:"page_number,omitempty" parquet:"page_number"`       // PDF/DOCX/PPTX page location (~30% populated)
	SectionLevel  *int    `json:"section_level,omitempty" parquet:"section_level"`   // Heading hierarchy level (~15% populated)
	RowIndex      *int    `json:"row_index,omitempty" parquet:"row_index"`           // Table row position (~20% populated)
	ColumnIndex   *int    `json:"column_index,omitempty" parquet:"column_index"`     // Table column position (~20% populated)
	TemporalType  *string `json:"temporal_type,omitempty" parquet:"temporal_type"`   // date/datetime/year/etc (~5-10% populated)
	TagName       *string `json:"tag_name,omitempty" parquet:"tag_name"`             // HTML/XML tag identifier (~25% populated)

	// UDML Phase 2: Code-Specific Promoted Fields (for source code parsing)
	FunctionName  *string `json:"function_name,omitempty" parquet:"function_name"`   // Function/method name (~10% populated in code files)
	ClassName     *string `json:"class_name,omitempty" parquet:"class_name"`         // Class/struct/interface name (~5% populated)
	Namespace     *string `json:"namespace,omitempty" parquet:"namespace"`           // Package/module path (~15% populated)
	LineNumber    *int    `json:"line_number,omitempty" parquet:"line_number"`       // Source line number (~40% populated in code)
	Signature     *string `json:"signature,omitempty" parquet:"signature"`           // Function signature or type definition (~10% populated)

	// JSON overflow (format-specific, rarely queried attributes)
	ContentLocation map[string]interface{} `json:"content_location,omitempty" parquet:"content_location"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" parquet:"metadata"`
}

// ParseResult is the universal output format for all parsers
type ParseResult struct {
	Document Document               `json:"document"`
	Elements []Element              `json:"elements"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}


// Common relationship types (used internally for parent-child hierarchy via parent_id)
const (
	RelationshipContains = "contains"
)

// Element categories for Universal Document Model
const (
	CategoryContainer = "container"
	CategoryContent   = "content"
	CategoryStructure = "structure"
	CategoryComponent = "component"
	CategoryMetadata  = "metadata"
)

// ElementTaxonomy represents the structure of the element_taxonomy.json file
type ElementTaxonomy struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	Categories  map[string]struct {
		Description  string   `json:"description"`
		ElementTypes []string `json:"element_types"`
	} `json:"categories"`
	DefaultCategory string `json:"default_category"`
}

var (
	taxonomyOnce     sync.Once
	taxonomy         *ElementTaxonomy
	typeToCategoryMap map[string]string
)

// loadTaxonomy loads the element taxonomy from JSON file
func loadTaxonomy() {
	taxonomyOnce.Do(func() {
		// Find the taxonomy file - look in project root
		possiblePaths := []string{
			"element_taxonomy.json",
			"../element_taxonomy.json",
			"../../element_taxonomy.json",
			"../../../element_taxonomy.json",
		}

		var taxonomyData []byte
		var err error

		for _, path := range possiblePaths {
			taxonomyData, err = os.ReadFile(path)
			if err == nil {
				break
			}
		}

		if err != nil {
			// Fallback to default if file not found
			taxonomy = &ElementTaxonomy{DefaultCategory: CategoryComponent}
			typeToCategoryMap = make(map[string]string)
			return
		}

		taxonomy = &ElementTaxonomy{}
		if err := json.Unmarshal(taxonomyData, taxonomy); err != nil {
			taxonomy = &ElementTaxonomy{DefaultCategory: CategoryComponent}
			typeToCategoryMap = make(map[string]string)
			return
		}

		// Build reverse map: element_type -> category
		typeToCategoryMap = make(map[string]string)
		for categoryName, categoryData := range taxonomy.Categories {
			for _, elementType := range categoryData.ElementTypes {
				typeToCategoryMap[elementType] = categoryName
			}
		}
	})
}

// GetElementCategory returns the category for a given element type
func GetElementCategory(elementType string) string {
	loadTaxonomy()

	if category, exists := typeToCategoryMap[elementType]; exists {
		return category
	}

	// Return default category for unknown types
	if taxonomy != nil && taxonomy.DefaultCategory != "" {
		return taxonomy.DefaultCategory
	}
	return CategoryComponent
}

// Helper functions

// generateID generates a unique ID with an optional prefix
func generateID(prefix string) string {
	b := make([]byte, 8)
	rand.Read(b)
	timestamp := time.Now().UnixNano() / 1000000 // milliseconds
	return fmt.Sprintf("%s_%d_%x", prefix, timestamp, b)
}

// ToJSON converts ParseResult to JSON string
func (r *ParseResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}