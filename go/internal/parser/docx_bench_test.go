package parser

import (
	"path/filepath"
	"testing"
)

// BenchmarkDocxParser_Small benchmarks parsing a small DOCX document
func BenchmarkDocxParser_Small(b *testing.B) {
	parser := NewDocxParser()
	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "bench_small",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	// Warm up - parse once to load file into cache
	_, err := parser.Parse(request)
	if err != nil {
		b.Fatalf("Warmup parse failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(request)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// BenchmarkDocxParser_MergedCells benchmarks parsing document with merged cells
func BenchmarkDocxParser_MergedCells(b *testing.B) {
	parser := NewDocxParser()
	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_merged_cells.docx")

	request := DocxParseRequest{
		ID:       "bench_merged",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_merged_cells.docx"},
	}

	// Warm up
	_, err := parser.Parse(request)
	if err != nil {
		b.Fatalf("Warmup parse failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(request)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// BenchmarkDocxParser_RealWorld benchmarks parsing a real-world document
func BenchmarkDocxParser_RealWorld(b *testing.B) {
	parser := NewDocxParser()
	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "Cash Management Research Report.docx")

	request := DocxParseRequest{
		ID:       "bench_real_world",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "Cash Management Research Report.docx"},
	}

	// Warm up
	_, err := parser.Parse(request)
	if err != nil {
		b.Fatalf("Warmup parse failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(request)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// BenchmarkDocxParser_LargeProfessional benchmarks parsing a large professional document
func BenchmarkDocxParser_LargeProfessional(b *testing.B) {
	parser := NewDocxParser()
	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "Hasura Professional Service Offerings for Financial Services.docx")

	request := DocxParseRequest{
		ID:       "bench_large",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "Hasura Professional Service Offerings for Financial Services.docx"},
	}

	// Warm up
	_, err := parser.Parse(request)
	if err != nil {
		b.Fatalf("Warmup parse failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(request)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// BenchmarkDocxParser_ParallelParsing benchmarks concurrent parsing
func BenchmarkDocxParser_ParallelParsing(b *testing.B) {
	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	b.RunParallel(func(pb *testing.PB) {
		parser := NewDocxParser()
		request := DocxParseRequest{
			ID:       "bench_parallel",
			Content:  testFile,
			Metadata: map[string]interface{}{"filename": "test_sample.docx"},
		}

		for pb.Next() {
			_, err := parser.Parse(request)
			if err != nil {
				b.Fatalf("Parse failed: %v", err)
			}
		}
	})
}

// BenchmarkDocxParser_ElementExtraction benchmarks just element extraction
func BenchmarkDocxParser_ElementExtraction(b *testing.B) {
	parser := NewDocxParser()
	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "bench_elements",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	// Parse once to get the document
	response, err := parser.Parse(request)
	if err != nil {
		b.Fatalf("Initial parse failed: %v", err)
	}

	// Benchmark the ToParseResult conversion
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = response.ToParseResult()
	}
}

// BenchmarkDocxParser_WithHeadersFooters benchmarks with headers/footers extraction
func BenchmarkDocxParser_WithHeadersFooters(b *testing.B) {
	parser := NewDocxParser()
	parser.ExtractHeadersFooters = true
	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "bench_headers_footers",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(request)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// BenchmarkDocxParser_WithoutOptionalFeatures benchmarks with minimal features
func BenchmarkDocxParser_WithoutOptionalFeatures(b *testing.B) {
	parser := NewDocxParser()
	parser.ExtractHeadersFooters = false
	parser.ExtractComments = false
	parser.ExtractStyles = false
	parser.ExtractDates = false
	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "bench_minimal",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(request)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

// BenchmarkDocxParser_MemoryAllocation benchmarks memory allocations
func BenchmarkDocxParser_MemoryAllocation(b *testing.B) {
	parser := NewDocxParser()
	testAssets := "../../../tests/assets"
	testFile := filepath.Join(testAssets, "test_sample.docx")

	request := DocxParseRequest{
		ID:       "bench_memory",
		Content:  testFile,
		Metadata: map[string]interface{}{"filename": "test_sample.docx"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(request)
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}
