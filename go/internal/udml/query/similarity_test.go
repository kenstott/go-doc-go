package query

import (
	"math"
	"testing"
)

func TestCosineSimilarityFloat64(t *testing.T) {
	tests := []struct {
		name    string
		a       []float64
		b       []float64
		want    float64
		wantErr bool
	}{
		{
			name: "identical vectors",
			a:    []float64{1, 2, 3},
			b:    []float64{1, 2, 3},
			want: 1.0,
		},
		{
			name: "opposite vectors",
			a:    []float64{1, 0, 0},
			b:    []float64{-1, 0, 0},
			want: -1.0,
		},
		{
			name: "orthogonal vectors",
			a:    []float64{1, 0},
			b:    []float64{0, 1},
			want: 0.0,
		},
		{
			name: "similar vectors",
			a:    []float64{1, 2, 3},
			b:    []float64{2, 4, 6},
			want: 1.0,
		},
		{
			name:    "different lengths",
			a:       []float64{1, 2},
			b:       []float64{1, 2, 3},
			wantErr: true,
		},
		{
			name:    "empty vectors",
			a:       []float64{},
			b:       []float64{},
			wantErr: true,
		},
		{
			name:    "zero vector",
			a:       []float64{0, 0, 0},
			b:       []float64{1, 2, 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CosineSimilarityFloat64(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("CosineSimilarityFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("CosineSimilarityFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCosineSimilarityFloat32(t *testing.T) {
	tests := []struct {
		name    string
		a       []float32
		b       []float32
		want    float32
		wantErr bool
	}{
		{
			name: "identical vectors",
			a:    []float32{1, 2, 3},
			b:    []float32{1, 2, 3},
			want: 1.0,
		},
		{
			name: "opposite vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{-1, 0, 0},
			want: -1.0,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{1, 0},
			b:    []float32{0, 1},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CosineSimilarityFloat32(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("CosineSimilarityFloat32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && math.Abs(float64(got-tt.want)) > 1e-6 {
				t.Errorf("CosineSimilarityFloat32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEuclideanDistanceFloat64(t *testing.T) {
	tests := []struct {
		name    string
		a       []float64
		b       []float64
		want    float64
		wantErr bool
	}{
		{
			name: "identical vectors",
			a:    []float64{1, 2, 3},
			b:    []float64{1, 2, 3},
			want: 0.0,
		},
		{
			name: "unit distance",
			a:    []float64{0, 0},
			b:    []float64{1, 0},
			want: 1.0,
		},
		{
			name: "diagonal distance",
			a:    []float64{0, 0},
			b:    []float64{3, 4},
			want: 5.0, // 3-4-5 triangle
		},
		{
			name:    "different lengths",
			a:       []float64{1, 2},
			b:       []float64{1, 2, 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EuclideanDistanceFloat64(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("EuclideanDistanceFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("EuclideanDistanceFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDotProductFloat64(t *testing.T) {
	tests := []struct {
		name    string
		a       []float64
		b       []float64
		want    float64
		wantErr bool
	}{
		{
			name: "simple dot product",
			a:    []float64{1, 2, 3},
			b:    []float64{4, 5, 6},
			want: 32.0, // 1*4 + 2*5 + 3*6 = 4 + 10 + 18 = 32
		},
		{
			name: "orthogonal vectors",
			a:    []float64{1, 0},
			b:    []float64{0, 1},
			want: 0.0,
		},
		{
			name: "negative values",
			a:    []float64{1, -2},
			b:    []float64{3, 4},
			want: -5.0, // 1*3 + (-2)*4 = 3 - 8 = -5
		},
		{
			name:    "different lengths",
			a:       []float64{1, 2},
			b:       []float64{1, 2, 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DotProductFloat64(tt.a, tt.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("DotProductFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("DotProductFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertFloat32ToFloat64(t *testing.T) {
	input := []float32{1.5, 2.5, 3.5}
	expected := []float64{1.5, 2.5, 3.5}

	result := ConvertFloat32ToFloat64(input)

	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}

	for i := range expected {
		if math.Abs(result[i]-expected[i]) > 1e-6 {
			t.Errorf("At index %d: expected %v, got %v", i, expected[i], result[i])
		}
	}
}

func TestConvertFloat64ToFloat32(t *testing.T) {
	input := []float64{1.5, 2.5, 3.5}
	expected := []float32{1.5, 2.5, 3.5}

	result := ConvertFloat64ToFloat32(input)

	if len(result) != len(expected) {
		t.Errorf("Expected length %d, got %d", len(expected), len(result))
	}

	for i := range expected {
		if math.Abs(float64(result[i]-expected[i])) > 1e-6 {
			t.Errorf("At index %d: expected %v, got %v", i, expected[i], result[i])
		}
	}
}

func TestNormalizeFloat64(t *testing.T) {
	tests := []struct {
		name    string
		vec     []float64
		wantErr bool
	}{
		{
			name: "simple vector",
			vec:  []float64{3, 4}, // Should become {0.6, 0.8}
		},
		{
			name: "already normalized",
			vec:  []float64{1, 0, 0},
		},
		{
			name:    "zero vector",
			vec:     []float64{0, 0, 0},
			wantErr: true,
		},
		{
			name:    "empty vector",
			vec:     []float64{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeFloat64(tt.vec)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeFloat64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check that the result has unit length
				var norm float64
				for _, v := range got {
					norm += v * v
				}
				norm = math.Sqrt(norm)

				if math.Abs(norm-1.0) > 1e-10 {
					t.Errorf("Normalized vector should have unit length, got %v", norm)
				}
			}
		})
	}
}

func TestNormalizeFloat32(t *testing.T) {
	tests := []struct {
		name    string
		vec     []float32
		wantErr bool
	}{
		{
			name: "simple vector",
			vec:  []float32{3, 4}, // Should become {0.6, 0.8}
		},
		{
			name:    "zero vector",
			vec:     []float32{0, 0, 0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeFloat32(tt.vec)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeFloat32() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check that the result has unit length
				var norm float32
				for _, v := range got {
					norm += v * v
				}
				norm = float32(math.Sqrt(float64(norm)))

				if math.Abs(float64(norm-1.0)) > 1e-6 {
					t.Errorf("Normalized vector should have unit length, got %v", norm)
				}
			}
		})
	}
}

func TestApplySimilarityFilter(t *testing.T) {
	// Create test data with mock embeddings
	rows := []Row{
		{
			"element_id": "elem1",
			"content":    "First document",
			"embedding":  []float64{1.0, 0.0, 0.0},
		},
		{
			"element_id": "elem2",
			"content":    "Second document",
			"embedding":  []float64{0.9, 0.1, 0.0},
		},
		{
			"element_id": "elem3",
			"content":    "Third document",
			"embedding":  []float64{0.0, 1.0, 0.0},
		},
		{
			"element_id": "elem4",
			"content":    "Fourth document",
			"embedding":  []float64{-1.0, 0.0, 0.0},
		},
	}

	results := &QueryResult{
		Rows:     rows,
		RowCount: len(rows),
	}

	t.Run("filter with threshold", func(t *testing.T) {
		pred := NewSimilarity(
			"embedding",
			[]float32{1.0, 0.0, 0.0}, // Query vector
			0.8,                       // Threshold (cosine similarity)
			0,                         // TopK (0 = no limit)
		)

		filtered, err := ApplySimilarityFilter(results, pred)
		if err != nil {
			t.Fatalf("ApplySimilarityFilter() error = %v", err)
		}

		// Should return elem1 (1.0) and elem2 (0.9947...)
		if filtered.RowCount < 2 {
			t.Errorf("Expected at least 2 results with threshold 0.8, got %d", filtered.RowCount)
		}

		// First result should be elem1 (highest similarity)
		if filtered.Rows[0]["element_id"] != "elem1" {
			t.Errorf("Expected first result to be elem1, got %v", filtered.Rows[0]["element_id"])
		}
	})

	t.Run("filter with topK", func(t *testing.T) {
		pred := NewSimilarity(
			"embedding",
			[]float32{1.0, 0.0, 0.0},
			0.0, // No threshold
			2,   // TopK = 2
		)

		filtered, err := ApplySimilarityFilter(results, pred)
		if err != nil {
			t.Fatalf("ApplySimilarityFilter() error = %v", err)
		}

		if filtered.RowCount != 2 {
			t.Errorf("Expected exactly 2 results with topK=2, got %d", filtered.RowCount)
		}
	})

	t.Run("results include similarity score", func(t *testing.T) {
		pred := NewSimilarity(
			"embedding",
			[]float32{1.0, 0.0, 0.0},
			0.0,
			1,
		)

		filtered, err := ApplySimilarityFilter(results, pred)
		if err != nil {
			t.Fatalf("ApplySimilarityFilter() error = %v", err)
		}

		if len(filtered.Rows) == 0 {
			t.Fatal("Expected at least one result")
		}

		// Check that similarity score was added
		if _, ok := filtered.Rows[0]["_similarity"]; !ok {
			t.Error("Expected _similarity field in result")
		}
	})

	t.Run("no predicate returns original", func(t *testing.T) {
		filtered, err := ApplySimilarityFilter(results, nil)
		if err != nil {
			t.Fatalf("ApplySimilarityFilter() error = %v", err)
		}

		if filtered.RowCount != results.RowCount {
			t.Errorf("Expected %d rows, got %d", results.RowCount, filtered.RowCount)
		}
	})
}

func BenchmarkCosineSimilarityFloat64(b *testing.B) {
	vec1 := make([]float64, 384) // Common embedding size
	vec2 := make([]float64, 384)

	for i := range vec1 {
		vec1[i] = float64(i) / 384.0
		vec2[i] = float64(i+1) / 384.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarityFloat64(vec1, vec2)
	}
}

func BenchmarkEuclideanDistanceFloat64(b *testing.B) {
	vec1 := make([]float64, 384)
	vec2 := make([]float64, 384)

	for i := range vec1 {
		vec1[i] = float64(i) / 384.0
		vec2[i] = float64(i+1) / 384.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EuclideanDistanceFloat64(vec1, vec2)
	}
}
