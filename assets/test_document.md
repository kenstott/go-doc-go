# Test Document for Embedding Validation

## Introduction

This is a test document to validate the embedding creation process in Go-Doc-Go. 

### Section 1: Content Structure

The document parser should create a hierarchical structure with:
- Root element containing the document
- Section headers as container elements  
- Paragraphs as leaf elements that get embeddings

### Section 2: Embedding Requirements

Only leaf elements should receive embeddings. Container elements like:
- Root
- Headers
- Lists
- Tables

Should be skipped during embedding generation.

## Conclusion

The contextual embedding generator should create graphlets that include the complete parent hierarchy for each leaf element.