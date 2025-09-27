package parser

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	pdf "github.com/ledongthuc/pdf"
)

// SimplePDFParser handles PDF document parsing with text extraction
type SimplePDFParser struct {
	MaxContentPreview int
	MaxPages          int
	ExtractLinks      bool
}

// NewSimplePDFParser creates a new PDF parser
func NewSimplePDFParser() *SimplePDFParser {
	return &SimplePDFParser{
		MaxContentPreview: 100,
		MaxPages:          1000,
		ExtractLinks:      true,
	}
}

// Parse parses PDF content
func (p *SimplePDFParser) Parse(docID string, content interface{}) (*ParseResult, error) {
	var reader io.ReaderAt
	var size int64

	// Handle different input types
	switch v := content.(type) {
	case string:
		// File path
		file, err := os.Open(v)
		if err != nil {
			return nil, fmt.Errorf("failed to open PDF file: %v", err)
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to stat PDF file: %v", err)
		}
		size = stat.Size()
		reader = file

	case []byte:
		// Raw bytes
		reader = bytes.NewReader(v)
		size = int64(len(v))

	case io.ReaderAt:
		// Already a reader
		reader = v
		// Try to determine size
		if seeker, ok := v.(io.Seeker); ok {
			currentPos, _ := seeker.Seek(0, io.SeekCurrent)
			size, _ = seeker.Seek(0, io.SeekEnd)
			seeker.Seek(currentPos, io.SeekStart)
		} else {
			// Default size if we can't determine it
			size = 1 << 20 // 1MB default
		}

	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}

	// Initialize result
	result := &ParseResult{
		Document: Document{
			ID:      docID,
			DocType: "pdf",
		},
		Elements:      []Element{},
		Relationships: []Relationship{},
		Links:         []Link{},
	}

	// Parse PDF
	pdfReader, err := pdf.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PDF: %v", err)
	}

	// Create root element
	rootID := generateID("root_")
	rootElement := Element{
		ElementID:      rootID,
		ElementType:    "root",
		ContentPreview: "PDF Document",
		Position:       0,
		Depth:          0,
	}
	result.Elements = append(result.Elements, rootElement)

	// Create body element
	bodyID := generateID("body_")
	bodyElement := Element{
		ElementID:      bodyID,
		ElementType:    "body",
		ContentPreview: "Document Body",
		ParentID:       rootID,
		Position:       1,
		Depth:          1,
	}
	result.Elements = append(result.Elements, bodyElement)

	// Add root->body relationship
	result.Relationships = append(result.Relationships, Relationship{
		RelationshipID:   generateID("rel_"),
		RelationshipType: "contains",
		SourceElementID:  rootID,
		TargetElementID:  bodyID,
	})

	// Process pages
	pageCount := pdfReader.NumPage()
	if p.MaxPages > 0 && pageCount > p.MaxPages {
		pageCount = p.MaxPages
	}

	elementPosition := 2
	for pageNum := 1; pageNum <= pageCount; pageNum++ {
		// Get the page
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		// Create page element
		pageID := generateID(fmt.Sprintf("page_%d_", pageNum))
		pageElement := Element{
			ElementID:      pageID,
			ElementType:    "page",
			ContentPreview: fmt.Sprintf("Page %d", pageNum),
			ParentID:       bodyID,
			Position:       elementPosition,
			Depth:          2,
			ContentLocation: map[string]interface{}{
				"page_number": pageNum,
			},
		}
		result.Elements = append(result.Elements, pageElement)
		elementPosition++

		// Add body->page relationship
		result.Relationships = append(result.Relationships, Relationship{
			RelationshipID:   generateID("rel_"),
			RelationshipType: "contains",
			SourceElementID:  bodyID,
			TargetElementID:  pageID,
		})

		// Extract text from page
		pageText := p.extractPageText(page)
		if pageText == "" {
			continue
		}

		// Split into paragraphs
		paragraphs := p.splitIntoParagraphs(pageText)
		for _, para := range paragraphs {
			if strings.TrimSpace(para) == "" {
				continue
			}

			// Create paragraph element
			paraID := generateID("para_")
			paraElement := Element{
				ElementID:      paraID,
				ElementType:    "paragraph",
				Content:        para,
				ContentPreview: p.truncateText(para, p.MaxContentPreview),
				ParentID:       pageID,
				Position:       elementPosition,
				Depth:          3,
				ContentLocation: map[string]interface{}{
					"page_number": pageNum,
				},
			}
			result.Elements = append(result.Elements, paraElement)
			elementPosition++

			// Add page->paragraph relationship
			result.Relationships = append(result.Relationships, Relationship{
				RelationshipID:   generateID("rel_"),
				RelationshipType: "contains",
				SourceElementID:  pageID,
				TargetElementID:  paraID,
			})

			// Extract links
			if p.ExtractLinks {
				links := p.extractLinks(para, paraID)
				result.Links = append(result.Links, links...)
			}
		}
	}

	return result, nil
}

// extractPageText extracts text from a PDF page
func (p *SimplePDFParser) extractPageText(page pdf.Page) string {
	var textParts []string

	// Get page content
	content := page.Content()

	// Extract all text objects first
	for _, text := range content.Text {
		if text.S != "" {
			textParts = append(textParts, text.S)
		}
	}

	// Join all text parts and clean up
	fullText := strings.Join(textParts, "")
	return p.cleanCharacterSpacing(fullText)
}

// cleanCharacterSpacing fixes character spacing issues in PDF text
func (p *SimplePDFParser) cleanCharacterSpacing(text string) string {
	// Many PDFs extract with spaces between every character like "h e l l o"
	// This function attempts to fix that

	// Split text into tokens (by spaces)
	tokens := strings.Fields(text)
	if len(tokens) == 0 {
		return text
	}

	var result strings.Builder
	var currentWord strings.Builder

	for i, token := range tokens {
		// Check if this token is a single character (likely part of a spaced word)
		if len(token) == 1 {
			currentWord.WriteString(token)
		} else {
			// Multi-character token - finish current word and add this token
			if currentWord.Len() > 0 {
				if result.Len() > 0 {
					result.WriteString(" ")
				}
				result.WriteString(currentWord.String())
				currentWord.Reset()
			}

			if result.Len() > 0 {
				result.WriteString(" ")
			}
			result.WriteString(token)
		}

		// Check if next token looks like it starts a new word
		if currentWord.Len() > 0 && i < len(tokens)-1 {
			nextToken := tokens[i+1]
			// If next token is punctuation or looks like start of new word, finish current word
			if len(nextToken) > 1 || p.isPunctuation(nextToken) || p.isWordBoundary(token, nextToken) {
				if result.Len() > 0 {
					result.WriteString(" ")
				}
				result.WriteString(currentWord.String())
				currentWord.Reset()
			}
		}
	}

	// Add any remaining word
	if currentWord.Len() > 0 {
		if result.Len() > 0 {
			result.WriteString(" ")
		}
		result.WriteString(currentWord.String())
	}

	finalText := result.String()

	// Clean up multiple spaces
	finalText = regexp.MustCompile(`\s+`).ReplaceAllString(finalText, " ")
	finalText = strings.TrimSpace(finalText)

	// Add intelligent spacing for readability
	finalText = p.addIntelligentSpacing(finalText)

	return finalText
}

// addIntelligentSpacing adds spaces in logical places for readability
func (p *SimplePDFParser) addIntelligentSpacing(text string) string {
	// Add spaces before certain patterns to improve readability

	// Add space before uppercase letters that follow lowercase (camelCase)
	text = regexp.MustCompile(`([a-z])([A-Z])`).ReplaceAllString(text, "$1 $2")

	// Add space before opening parentheses
	text = regexp.MustCompile(`([a-zA-Z0-9])(\()`).ReplaceAllString(text, "$1 $2")

	// Add space after closing parentheses if followed by letter
	text = regexp.MustCompile(`(\))([a-zA-Z])`).ReplaceAllString(text, "$1 $2")

	// Add space before numbers that follow letters (like "VARCHAR255" -> "VARCHAR 255")
	text = regexp.MustCompile(`([a-zA-Z])(\d)`).ReplaceAllString(text, "$1 $2")

	// Add space after numbers that are followed by letters (like "255first" -> "255 first")
	text = regexp.MustCompile(`(\d)([a-zA-Z])`).ReplaceAllString(text, "$1 $2")

	// Clean up any double spaces created
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}

// isPunctuation checks if a token is punctuation
func (p *SimplePDFParser) isPunctuation(token string) bool {
	punctuation := ".,!?;:()[]{}\"'-"
	return len(token) == 1 && strings.Contains(punctuation, token)
}

// isWordBoundary determines if there should be a word boundary between two tokens
func (p *SimplePDFParser) isWordBoundary(current, next string) bool {
	// Heuristics for word boundaries:
	// - Current token is punctuation
	// - Next token is uppercase (might start new sentence)
	// - Pattern changes (like from letters to numbers)

	if len(current) == 1 && len(next) == 1 {
		// Check if we're transitioning from letter to number or vice versa
		currentIsLetter := (current >= "a" && current <= "z") || (current >= "A" && current <= "Z")
		currentIsDigit := current >= "0" && current <= "9"
		nextIsLetter := (next >= "a" && next <= "z") || (next >= "A" && next <= "Z")
		nextIsDigit := next >= "0" && next <= "9"

		// Word boundary if transitioning between letters and digits
		if (currentIsLetter && nextIsDigit) || (currentIsDigit && nextIsLetter) {
			return true
		}

		// Word boundary if next is uppercase (could be start of new word)
		if nextIsLetter && next >= "A" && next <= "Z" {
			return true
		}
	}

	return false
}

// splitIntoParagraphs splits text into paragraphs
func (p *SimplePDFParser) splitIntoParagraphs(text string) []string {
	// Split by double newlines
	parts := strings.Split(text, "\n\n")
	var paragraphs []string

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			// Also handle single newlines as paragraph breaks if they look like separate thoughts
			lines := strings.Split(trimmed, "\n")
			currentPara := ""

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// Check if line looks like a new paragraph (starts with capital, previous ended with period)
				if currentPara != "" &&
				   strings.HasSuffix(currentPara, ".") &&
				   len(line) > 0 &&
				   strings.ToUpper(line[:1]) == line[:1] {
					// Save current paragraph and start new one
					paragraphs = append(paragraphs, currentPara)
					currentPara = line
				} else {
					// Continue building current paragraph
					if currentPara != "" {
						currentPara += " "
					}
					currentPara += line
				}
			}

			if currentPara != "" {
				paragraphs = append(paragraphs, currentPara)
			}
		}
	}

	return paragraphs
}

// extractLinks extracts URLs and emails from text
func (p *SimplePDFParser) extractLinks(text string, sourceID string) []Link {
	var links []Link

	// URL pattern
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	urls := urlPattern.FindAllString(text, -1)
	for _, url := range urls {
		links = append(links, Link{
			LinkID:          generateID("link_"),
			SourceElementID: sourceID,
			LinkType:        "url",
			LinkTarget:      url,
		})
	}

	// Email pattern
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	emails := emailPattern.FindAllString(text, -1)
	for _, email := range emails {
		links = append(links, Link{
			LinkID:          generateID("link_"),
			SourceElementID: sourceID,
			LinkType:        "email",
			LinkTarget:      email,
		})
	}

	return links
}

// truncateText truncates text to specified length
func (p *SimplePDFParser) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}