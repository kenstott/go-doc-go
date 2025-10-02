package embeddings

import (
	"strings"

	"github.com/kennethstott/go-doc-go/internal/parser"
)

// ContextualTextBuilder builds context-aware text for embeddings
type ContextualTextBuilder struct {
	predecessorCount int
	successorCount   int
	maxTokens        int
	safeMaxTokens    int

	// Token budget ratios
	elementRatio  float64
	parentsRatio  float64
	siblingsRatio float64
	childrenRatio float64
}

// NewContextualTextBuilder creates a new contextual text builder
func NewContextualTextBuilder(config Config) *ContextualTextBuilder {
	maxTokens := 16384 // Default
	if config.ChunkSize > 0 {
		maxTokens = config.ChunkSize
	}

	return &ContextualTextBuilder{
		predecessorCount: config.PredecessorCount,
		successorCount:   config.SuccessorCount,
		maxTokens:        maxTokens,
		safeMaxTokens:    int(float64(maxTokens) * 0.95),
		elementRatio:     0.40,
		parentsRatio:     0.25,
		siblingsRatio:    0.20,
		childrenRatio:    0.15,
	}
}

// BuildContextualText creates a context-enriched text string for an element
func (b *ContextualTextBuilder) BuildContextualText(
	element parser.Element,
	allElements []parser.Element,
	relationships []parser.Relationship,
) string {
	// Build element map (for parent traversal)
	elementMap := buildElementMap(allElements)

	// Get element's full content
	elementText := element.ContentPreview
	if element.Content != "" {
		elementText = element.Content
	}

	// Calculate token budgets
	elementBudget := int(float64(b.safeMaxTokens) * b.elementRatio)
	parentBudget := int(float64(b.safeMaxTokens) * b.parentsRatio)
	siblingBudget := int(float64(b.safeMaxTokens) * b.siblingsRatio)

	// Truncate main element if needed
	elementProcessed := b.truncateToTokens(elementText, elementBudget)

	// Build parent chain (bottom-up: immediate parent to root)
	parentTexts := b.collectParentTexts(element, elementMap, parentBudget)

	// Build sibling texts using array positions (like Python)
	siblingTexts := b.collectSiblingTexts(element, allElements, siblingBudget)

	// Combine: element first, then parents, then siblings
	var parts []string

	// Main element
	parts = append(parts, elementProcessed)

	// Parents (already ordered from immediate to root)
	for _, parentText := range parentTexts {
		if parentText != "" {
			parts = append(parts, parentText)
		}
	}

	// Siblings
	for _, siblingText := range siblingTexts {
		if siblingText != "" {
			parts = append(parts, siblingText)
		}
	}

	// Join with newlines
	combined := strings.Join(parts, "\n")

	// Final safety check
	if b.countTokens(combined) > b.safeMaxTokens {
		combined = b.truncateToTokens(combined, b.safeMaxTokens)
	}

	return combined
}

// collectParentTexts collects parent texts from immediate parent to root
func (b *ContextualTextBuilder) collectParentTexts(
	element parser.Element,
	elementMap map[string]parser.Element,
	budget int,
) []string {
	var parentTexts []string
	usedTokens := 0

	currentID := element.ParentID
	for currentID != "" {
		parent, ok := elementMap[currentID]
		if !ok {
			break
		}

		// Skip root elements (like Python does)
		if parent.ElementType == "root" || parent.ElementType == "document_root" {
			currentID = parent.ParentID
			continue
		}

		// Skip structural-only containers (like Python does)
		if isStructuralOnlyContainer(parent) {
			currentID = parent.ParentID
			continue
		}

		// Skip parents without content
		if parent.ContentPreview == "" && parent.Content == "" {
			currentID = parent.ParentID
			continue
		}

		parentText := parent.ContentPreview
		if parent.Content != "" {
			parentText = parent.Content
		}

		tokens := b.countTokens(parentText)
		if usedTokens+tokens <= budget {
			parentTexts = append(parentTexts, parentText)
			usedTokens += tokens
		} else if usedTokens < budget {
			// Truncate to fit
			remaining := budget - usedTokens
			if remaining > 50 {
				truncated := b.truncateToTokens(parentText, remaining)
				parentTexts = append(parentTexts, truncated)
			}
			break
		} else {
			break
		}

		currentID = parent.ParentID
	}

	// Reverse to match Python's order (root → immediate parent)
	for i, j := 0, len(parentTexts)-1; i < j; i, j = i+1, j-1 {
		parentTexts[i], parentTexts[j] = parentTexts[j], parentTexts[i]
	}

	return parentTexts
}

// collectSiblingTexts collects predecessor and successor texts using array positions (like Python)
func (b *ContextualTextBuilder) collectSiblingTexts(
	element parser.Element,
	allElements []parser.Element,
	budget int,
) []string {
	var siblingTexts []string
	usedTokens := 0

	// Find current element position in array
	currentIndex := -1
	for i, elem := range allElements {
		if elem.ElementID == element.ElementID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return siblingTexts
	}

	// Collect predecessors (elements before current in array)
	predCount := 0
	for i := currentIndex - 1; i >= 0 && predCount < b.predecessorCount; i-- {
		pred := allElements[i]

		// Filter like Python: skip roots and elements without content
		if pred.ElementType == "root" || pred.ContentPreview == "" {
			continue
		}

		// Skip structural-only containers (like Python)
		if isStructuralOnlyContainer(pred) {
			continue
		}

		predText := pred.ContentPreview
		if pred.Content != "" {
			predText = pred.Content
		}

		tokens := b.countTokens(predText)
		if usedTokens+tokens <= budget {
			// Prepend to maintain order (oldest first)
			siblingTexts = append([]string{predText}, siblingTexts...)
			usedTokens += tokens
			predCount++
		} else {
			break
		}
	}

	// Collect successors (elements after current in array)
	succCount := 0
	for i := currentIndex + 1; i < len(allElements) && succCount < b.successorCount; i++ {
		succ := allElements[i]

		// Filter like Python: skip roots and elements without content
		if succ.ElementType == "root" || succ.ContentPreview == "" {
			continue
		}

		// Skip structural-only containers (like Python)
		if isStructuralOnlyContainer(succ) {
			continue
		}

		succText := succ.ContentPreview
		if succ.Content != "" {
			succText = succ.Content
		}

		tokens := b.countTokens(succText)
		if usedTokens+tokens <= budget {
			siblingTexts = append(siblingTexts, succText)
			usedTokens += tokens
			succCount++
		} else {
			break
		}
	}

	return siblingTexts
}

// countTokens estimates token count (approximate: 1 token ≈ 4 chars)
func (b *ContextualTextBuilder) countTokens(text string) int {
	return len(text) / 4
}

// truncateToTokens truncates text to fit within token budget
func (b *ContextualTextBuilder) truncateToTokens(text string, maxTokens int) string {
	currentTokens := b.countTokens(text)
	if currentTokens <= maxTokens {
		return text
	}

	// Approximate character limit
	maxChars := maxTokens * 4
	if len(text) <= maxChars {
		return text
	}

	return text[:maxChars-3] + "..."
}

// Helper functions

func buildElementMap(elements []parser.Element) map[string]parser.Element {
	elementMap := make(map[string]parser.Element)

	for _, elem := range elements {
		elementMap[elem.ElementID] = elem
	}

	return elementMap
}

func buildHierarchy(elements []parser.Element) map[string][]string {
	hierarchy := make(map[string][]string)

	for _, elem := range elements {
		if elem.ParentID != "" {
			hierarchy[elem.ParentID] = append(hierarchy[elem.ParentID], elem.ElementID)
		}
	}

	return hierarchy
}

func isStructuralOnlyContainer(element parser.Element) bool {
	// Match Python's _is_structural_only_container logic
	structuralTypes := map[string]bool{
		"table":   true,
		"span":    true,
		"article": true,
		"section": true,
		"nav":     true,
		"aside":   true,
		"tbody":   true,
		"thead":   true,
	}

	if structuralTypes[element.ElementType] {
		content := strings.TrimSpace(element.ContentPreview)
		return content == "" || content == "..." || len(content) < 10
	}

	return false
}

// ShouldEmbed determines if an element should receive an embedding
func ShouldEmbed(element parser.Element, allElements []parser.Element) bool {
	// Skip elements that should use their parent's embedding
	skipTypes := map[string]bool{
		"table_cell":   true,
		"json_item":    true,
		"json_field":   true,
		"table":        true, // Always a container
		"header":       true, // Headers are for structure, not content embedding
		"root":         true, // Document root is not embedded
		"document_root": true, // Also skip document_root
	}

	if skipTypes[element.ElementType] {
		return false
	}

	// Only embed leaf elements (no children)
	// Exception: table_row and table_header_row can have children but still get embeddings
	hasChildrenExceptions := map[string]bool{
		"table_header_row": true,
		"table_row":        true,
	}

	if !hasChildrenExceptions[element.ElementType] {
		// Check if element has children
		for _, e := range allElements {
			if e.ParentID == element.ElementID {
				return false // Has children, skip
			}
		}
	}

	// Python resolves content later and checks then, so we don't check here
	// Content will be resolved in BuildContextualText if needed
	return true
}
