package parser

import (
	"strings"
	"testing"
)

func TestNewCSVParser(t *testing.T) {
	parser := NewCSVParser()
	if parser == nil {
		t.Fatal("Failed to create parser")
	}
	if parser.MaxContentPreview != 100 {
		t.Error("MaxContentPreview not set correctly")
	}
	if !parser.ExtractHeader {
		t.Error("ExtractHeader should be true by default")
	}
	if parser.Delimiter != ',' {
		t.Error("Delimiter should be comma by default")
	}
	if parser.MaxRows != 1000 {
		t.Error("MaxRows not set correctly")
	}
}

func TestParseSimpleCSV(t *testing.T) {
	parser := NewCSVParser()

	csvContent := `name,age,city
John Doe,30,New York
Jane Smith,25,Los Angeles`

	request := CSVParseRequest{
		ID:      "test_doc",
		Content: csvContent,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check document
	if response.Document["doc_id"] != "test_doc" {
		t.Error("Document ID not set correctly")
	}
	if response.Document["doc_type"] != "csv" {
		t.Error("Document type not set correctly")
	}

	// Should have root, table, header row, 2 data rows, and cells
	expectedMinElements := 1 + 1 + 1 + 2 + (3 * 3) // root + table + header + 2 rows + 9 cells
	if len(response.Elements) < expectedMinElements {
		t.Errorf("Expected at least %d elements, got %d", expectedMinElements, len(response.Elements))
	}

	// Root element should be first
	root := response.Elements[0]
	if root.ElementType != CSVElementTypeRoot {
		t.Error("First element should be root")
	}
	if root.ElementID == "" {
		t.Error("Root element should have ID")
	}
}

func TestParseCSVWithHeader(t *testing.T) {
	parser := NewCSVParser()

	csvContent := `product_id,product_name,price,category
1,Widget A,9.99,Electronics
2,Widget B,14.99,Electronics
3,Gadget C,29.99,Tools`

	request := CSVParseRequest{
		ID:      "test_products",
		Content: csvContent,
		Metadata: map[string]interface{}{
			"source": "products.csv",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check for header row element
	foundHeaderRow := false
	for _, element := range response.Elements {
		if element.ElementType == CSVElementTypeTableHeaderRow {
			foundHeaderRow = true
			if element.Metadata["row"] != 0 {
				t.Error("Header row should have row index 0")
			}
			break
		}
	}

	if !foundHeaderRow {
		t.Error("Header row element not found")
	}

	// Check for table cells with header information
	foundCellWithHeader := false
	for _, element := range response.Elements {
		if element.ElementType == CSVElementTypeTableCell {
			if header, ok := element.Metadata["header"]; ok && header != "" {
				foundCellWithHeader = true
				break
			}
		}
	}

	if !foundCellWithHeader {
		t.Error("Should have cells with header information")
	}
}

func TestParseCSVWithoutHeader(t *testing.T) {
	parser := NewCSVParser()
	parser.ExtractHeader = false

	csvContent := `John,30,Engineer
Jane,25,Designer
Bob,35,Manager`

	request := CSVParseRequest{
		ID:      "test_no_header",
		Content: csvContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should not have header row element
	for _, element := range response.Elements {
		if element.ElementType == CSVElementTypeTableHeaderRow {
			t.Error("Should not have header row when ExtractHeader is false")
		}
	}

	// Should still have table rows
	rowCount := 0
	for _, element := range response.Elements {
		if element.ElementType == CSVElementTypeTableRow {
			rowCount++
		}
	}

	if rowCount != 3 {
		t.Errorf("Expected 3 data rows, got %d", rowCount)
	}
}

func TestParseCSVWithDifferentDelimiter(t *testing.T) {
	parser := NewCSVParser()
	parser.Delimiter = ';'

	csvContent := `name;age;city
John Doe;30;New York
Jane Smith;25;Los Angeles`

	request := CSVParseRequest{
		ID:      "test_semicolon",
		Content: csvContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check that data was parsed correctly
	foundCorrectCell := false
	for _, element := range response.Elements {
		if element.ElementType == CSVElementTypeTableCell {
			if element.Text == "John Doe" {
				foundCorrectCell = true
				break
			}
		}
	}

	if !foundCorrectCell {
		t.Error("CSV with semicolon delimiter not parsed correctly")
	}
}

func TestParseCSVWithQuotes(t *testing.T) {
	parser := NewCSVParser()

	csvContent := `name,description,price
"Widget A","A great widget, with features",9.99
"Widget B","Another widget, even better",14.99`

	request := CSVParseRequest{
		ID:      "test_quotes",
		Content: csvContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check that quoted content was parsed correctly
	foundQuotedContent := false
	for _, element := range response.Elements {
		if element.ElementType == CSVElementTypeTableCell {
			if strings.Contains(element.Text, "A great widget, with features") {
				foundQuotedContent = true
				break
			}
		}
	}

	if !foundQuotedContent {
		t.Error("Quoted CSV content not parsed correctly")
	}
}

func TestParseEmptyCSV(t *testing.T) {
	parser := NewCSVParser()

	request := CSVParseRequest{
		ID:      "empty_test",
		Content: "",
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have at least root element
	if len(response.Elements) < 1 {
		t.Error("Empty CSV should have at least root element")
	}

	if response.Elements[0].ElementType != CSVElementTypeRoot {
		t.Error("First element should be root")
	}
}

func TestParseCSVWithMissingFields(t *testing.T) {
	parser := NewCSVParser()

	csvContent := `name,age,city
John Doe,30,New York
Jane Smith,,Los Angeles
Bob Wilson,35,`

	request := CSVParseRequest{
		ID:      "test_missing",
		Content: csvContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should handle missing fields gracefully
	emptyFieldCount := 0
	for _, element := range response.Elements {
		if element.ElementType == CSVElementTypeTableCell {
			if element.Text == "" {
				emptyFieldCount++
			}
		}
	}

	if emptyFieldCount != 2 {
		t.Errorf("Expected 2 empty fields, got %d", emptyFieldCount)
	}
}

func TestColumnTypeDetection(t *testing.T) {
	parser := NewCSVParser()

	// Test integer detection
	intValues := []string{"1", "2", "3", "100"}
	if parser.detectColumnType(intValues) != "integer" {
		t.Error("Should detect integer type")
	}

	// Test float detection
	floatValues := []string{"1.5", "2.7", "3.14"}
	if parser.detectColumnType(floatValues) != "float" {
		t.Error("Should detect float type")
	}

	// Test boolean detection
	boolValues := []string{"true", "false", "yes", "no"}
	if parser.detectColumnType(boolValues) != "boolean" {
		t.Error("Should detect boolean type")
	}

	// Test date detection
	dateValues := []string{"2023-01-01", "2023-12-31", "2024/06/15"}
	if parser.detectColumnType(dateValues) != "date" {
		t.Error("Should detect date type")
	}

	// Test string detection
	stringValues := []string{"hello", "world", "test"}
	if parser.detectColumnType(stringValues) != "string" {
		t.Error("Should detect string type")
	}
}

func TestLinkExtraction(t *testing.T) {
	parser := NewCSVParser()

	csvContent := `name,website,email
Company A,https://example.com,contact@example.com
Company B,https://test.org,info@test.org`

	request := CSVParseRequest{
		ID:      "test_links",
		Content: csvContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should extract URLs and emails
	if len(response.Links) < 4 {
		t.Errorf("Expected at least 4 links, got %d", len(response.Links))
	}

	// Check link types
	hasURL := false
	hasEmail := false
	for _, link := range response.Links {
		if link.LinkType == "url" && strings.HasPrefix(link.LinkTarget, "https://") {
			hasURL = true
		}
		if link.LinkType == "email" && strings.HasPrefix(link.LinkTarget, "mailto:") {
			hasEmail = true
		}
	}

	if !hasURL {
		t.Error("Should have extracted URL links")
	}
	if !hasEmail {
		t.Error("Should have extracted email links")
	}
}

func TestMaxRowsLimit(t *testing.T) {
	parser := NewCSVParser()
	parser.MaxRows = 2

	csvContent := `id,name
1,Row 1
2,Row 2
3,Row 3
4,Row 4
5,Row 5`

	request := CSVParseRequest{
		ID:      "test_max_rows",
		Content: csvContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Count data rows (excluding header)
	dataRowCount := 0
	for _, element := range response.Elements {
		if element.ElementType == CSVElementTypeTableRow {
			dataRowCount++
		}
	}

	if dataRowCount > 2 {
		t.Errorf("Expected max 2 data rows due to MaxRows limit, got %d", dataRowCount)
	}
}

func TestRelationshipCreation(t *testing.T) {
	parser := NewCSVParser()

	csvContent := `name,age
John,30
Jane,25`

	request := CSVParseRequest{
		ID:      "test_relationships",
		Content: csvContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have relationships
	if len(response.Relationships) == 0 {
		t.Error("Should have relationships between elements")
	}

	// Check relationship types
	hasContainsRelationship := false
	for _, rel := range response.Relationships {
		if rel.RelationshipType == "contains" {
			hasContainsRelationship = true
		}
		if rel.Confidence != 1.0 {
			t.Error("Relationship confidence should be 1.0")
		}
	}

	if !hasContainsRelationship {
		t.Error("Should have 'contains' relationships")
	}
}

func TestContentPreviewTruncation(t *testing.T) {
	parser := NewCSVParser()
	parser.MaxContentPreview = 20

	longText := "This is a very long text that should be truncated to fit the preview limit"
	preview := parser.truncateContent(longText)

	if len(preview) > 20 {
		t.Errorf("Preview too long: %d characters", len(preview))
	}
	if !strings.HasSuffix(preview, "...") {
		t.Error("Truncated preview should end with ...")
	}
}

func TestStripWhitespace(t *testing.T) {
	parser := NewCSVParser()
	parser.StripWhitespace = true

	csvContent := ` name , age , city
 John Doe , 30 , New York
 Jane Smith , 25 , Los Angeles `

	request := CSVParseRequest{
		ID:      "test_whitespace",
		Content: csvContent,
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check that whitespace was stripped
	foundTrimmedCell := false
	for _, element := range response.Elements {
		if element.ElementType == CSVElementTypeTableCell {
			if element.Text == "John Doe" {
				foundTrimmedCell = true
				break
			}
		}
	}

	if !foundTrimmedCell {
		t.Error("Whitespace should have been stripped from cells")
	}
}

func TestCSVSerialization(t *testing.T) {
	parser := NewCSVParser()

	request := CSVParseRequest{
		ID:      "test_csv",
		Content: "name,age\nJohn,30",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Test response to JSON
	jsonStr, err := response.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if !strings.Contains(jsonStr, "test_csv") {
		t.Error("JSON should contain document ID")
	}

	// Test request from JSON
	requestJSON := `{"id":"test","content":"name,age\ntest,25","metadata":{"key":"value"}}`
	var newRequest CSVParseRequest
	err = newRequest.FromJSON(requestJSON)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if newRequest.ID != "test" {
		t.Error("ID not parsed correctly from JSON")
	}
	if newRequest.Content != "name,age\ntest,25" {
		t.Error("Content not parsed correctly from JSON")
	}
}

func TestComplexRealWorldCSV(t *testing.T) {
	parser := NewCSVParser()

	// Simulate a more complex real-world CSV structure
	csvContent := `employee_id,first_name,last_name,email,department,salary,hire_date,manager_id
1001,John,Smith,john.smith@company.com,Engineering,75000,2023-01-15,2001
1002,Jane,Doe,jane.doe@company.com,Marketing,65000,2023-02-01,2002
1003,Bob,Johnson,bob.johnson@company.com,Engineering,80000,2023-01-20,2001
1004,Alice,Williams,alice.williams@company.com,Sales,70000,2023-03-01,2003
1005,Charlie,Brown,charlie.brown@company.com,Engineering,85000,2023-01-10,2001`

	request := CSVParseRequest{
		ID:      "employee_data",
		Content: csvContent,
		Metadata: map[string]interface{}{
			"source": "hr_system",
		},
	}

	response, err := parser.Parse(request)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have many elements due to complex structure
	if len(response.Elements) < 50 {
		t.Errorf("Expected many elements for complex CSV, got %d", len(response.Elements))
	}

	// Should have extracted links (emails)
	if len(response.Links) < 5 {
		t.Errorf("Expected at least 5 email links, got %d", len(response.Links))
	}

	// Check for specific content
	foundEmployeeName := false
	foundEmail := false
	for _, element := range response.Elements {
		if element.ElementType == CSVElementTypeTableCell {
			if strings.Contains(element.Text, "John") {
				foundEmployeeName = true
			}
		}
	}

	for _, link := range response.Links {
		if strings.Contains(link.LinkTarget, "@company.com") {
			foundEmail = true
		}
	}

	if !foundEmployeeName {
		t.Error("Should have found employee name in cells")
	}
	if !foundEmail {
		t.Error("Should have found company email in links")
	}

	// Check metadata
	metadata := response.Document["metadata"].(map[string]interface{})
	if metadata["row_count"].(int) != 6 { // 5 data rows + 1 header
		t.Error("Row count not calculated correctly")
	}
	if metadata["column_count"].(int) != 8 {
		t.Error("Column count not calculated correctly")
	}
}