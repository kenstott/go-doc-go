package parser

import (
	"encoding/xml"
	"log"
	"strings"
)

// DocxRel represents a DOCX relationship from the .rels file
type DocxRel struct {
	XMLName xml.Name `xml:"Relationship"`
	ID      string   `xml:"Id,attr"`
	Type    string   `xml:"Type,attr"`
	Target  string   `xml:"Target,attr"`
}

// DocxRelationships represents the relationships collection
type DocxRelationships struct {
	XMLName       xml.Name  `xml:"Relationships"`
	Relationships []DocxRel `xml:"Relationship"`
}

// parseRelationships parses the document relationships XML and extracts hyperlinks
func (p *DocxParser) parseRelationships(relsXML []byte, hyperlinks map[string]string) {
	var rels DocxRelationships
	err := xml.Unmarshal(relsXML, &rels)
	if err != nil {
		log.Printf("Failed to parse relationships XML: %v", err)
		return
	}

	// Extract hyperlinks (Type contains "hyperlink")
	for _, rel := range rels.Relationships {
		if strings.Contains(rel.Type, "hyperlink") {
			hyperlinks[rel.ID] = rel.Target
		}
	}
}

// extractHyperlinksFromText searches document XML for hyperlink elements and creates links
// This is a simplified implementation - full implementation would need to parse the
// document structure to find <w:hyperlink> elements
func (p *DocxParser) extractHyperlinksFromText(docXML string, hyperlinks map[string]string, response *DocxParseResponse) {
	// Search for hyperlink tags in the raw XML
	// Format: <w:hyperlink r:id="rId5">...</w:hyperlink>
	// This is a simple regex-based approach for now

	for _, target := range hyperlinks {
		// Create a link entry
		link := DocxLink{
			SourceID:   "", // We don't know the exact element ID without full XML parsing
			LinkText:   target,
			LinkTarget: target,
			LinkType:   "hyperlink",
		}
		response.Links = append(response.Links, link)
	}
}
