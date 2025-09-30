package parser

// Factory functions with consistent naming for all parsers

// NewDOCXParser creates a new DOCX parser (alias for NewDocxParser)
func NewDOCXParser() *DocxParser {
	return NewDocxParser()
}

// NewPPTXParser creates a new PPTX parser (alias for NewPptxParser)
func NewPPTXParser() *PptxParser {
	return NewPptxParser()
}
