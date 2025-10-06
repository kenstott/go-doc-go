package resolver

// ParserResolver defines the interface for parsers that can resolve content from locations
type ParserResolver interface {
	// ResolveElementText extracts plain text for a specific element using content_location
	// Returns the text content of the element
	ResolveElementText(contentLocation map[string]interface{}, sourceContent string) (string, error)

	// ResolveElementContent extracts raw content for a specific element using content_location
	// Returns the native format content (e.g., HTML for HTML parser)
	ResolveElementContent(contentLocation map[string]interface{}, sourceContent string) (string, error)

	// SupportsLocation checks if this parser can resolve the given content location
	SupportsLocation(contentLocation map[string]interface{}) bool
}

// ContentResolver resolves element content from content_location pointers
type ContentResolver interface {
	// ResolveContent resolves element content using the appropriate parser
	// text=true returns plain text, text=false returns native format
	ResolveContent(contentLocation map[string]interface{}, text bool) (string, error)
}
