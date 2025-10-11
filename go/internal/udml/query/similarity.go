package query

import (
	"fmt"
	"math"
)

// SimilarityMetric defines the type of similarity calculation
type SimilarityMetric string

const (
	// CosineSimilarity measures the cosine of the angle between vectors (range: -1 to 1)
	CosineSimilarity SimilarityMetric = "cosine"

	// EuclideanDistance measures straight-line distance between vectors (lower is more similar)
	EuclideanDistance SimilarityMetric = "euclidean"

	// DotProduct measures the dot product of vectors (higher is more similar)
	DotProduct SimilarityMetric = "dot"
)

// CosineSimilarityFloat64 calculates cosine similarity between two float64 vectors
// Returns a value between -1 (opposite) and 1 (identical)
func CosineSimilarityFloat64(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have same length: %d != %d", len(a), len(b))
	}

	if len(a) == 0 {
		return 0, fmt.Errorf("vectors cannot be empty")
	}

	var dotProduct, normA, normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("zero vector detected")
	}

	return dotProduct / (normA * normB), nil
}

// CosineSimilarityFloat32 calculates cosine similarity between two float32 vectors
// Returns a value between -1 (opposite) and 1 (identical)
func CosineSimilarityFloat32(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have same length: %d != %d", len(a), len(b))
	}

	if len(a) == 0 {
		return 0, fmt.Errorf("vectors cannot be empty")
	}

	var dotProduct, normA, normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = float32(math.Sqrt(float64(normA)))
	normB = float32(math.Sqrt(float64(normB)))

	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("zero vector detected")
	}

	return dotProduct / (normA * normB), nil
}

// EuclideanDistanceFloat64 calculates Euclidean distance between two float64 vectors
// Returns distance (0 = identical, higher = more different)
func EuclideanDistanceFloat64(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have same length: %d != %d", len(a), len(b))
	}

	if len(a) == 0 {
		return 0, fmt.Errorf("vectors cannot be empty")
	}

	var sum float64
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return math.Sqrt(sum), nil
}

// EuclideanDistanceFloat32 calculates Euclidean distance between two float32 vectors
// Returns distance (0 = identical, higher = more different)
func EuclideanDistanceFloat32(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have same length: %d != %d", len(a), len(b))
	}

	if len(a) == 0 {
		return 0, fmt.Errorf("vectors cannot be empty")
	}

	var sum float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum))), nil
}

// DotProductFloat64 calculates dot product between two float64 vectors
// Higher values indicate more similarity
func DotProductFloat64(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have same length: %d != %d", len(a), len(b))
	}

	if len(a) == 0 {
		return 0, fmt.Errorf("vectors cannot be empty")
	}

	var result float64
	for i := 0; i < len(a); i++ {
		result += a[i] * b[i]
	}

	return result, nil
}

// DotProductFloat32 calculates dot product between two float32 vectors
// Higher values indicate more similarity
func DotProductFloat32(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have same length: %d != %d", len(a), len(b))
	}

	if len(a) == 0 {
		return 0, fmt.Errorf("vectors cannot be empty")
	}

	var result float32
	for i := 0; i < len(a); i++ {
		result += a[i] * b[i]
	}

	return result, nil
}

// ConvertFloat32ToFloat64 converts a float32 slice to float64
func ConvertFloat32ToFloat64(vec []float32) []float64 {
	result := make([]float64, len(vec))
	for i, v := range vec {
		result[i] = float64(v)
	}
	return result
}

// ConvertFloat64ToFloat32 converts a float64 slice to float32
func ConvertFloat64ToFloat32(vec []float64) []float32 {
	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = float32(v)
	}
	return result
}

// NormalizeFloat64 normalizes a vector to unit length
func NormalizeFloat64(vec []float64) ([]float64, error) {
	if len(vec) == 0 {
		return nil, fmt.Errorf("cannot normalize empty vector")
	}

	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	if norm == 0 {
		return nil, fmt.Errorf("cannot normalize zero vector")
	}

	result := make([]float64, len(vec))
	for i, v := range vec {
		result[i] = v / norm
	}

	return result, nil
}

// NormalizeFloat32 normalizes a vector to unit length
func NormalizeFloat32(vec []float32) ([]float32, error) {
	if len(vec) == 0 {
		return nil, fmt.Errorf("cannot normalize empty vector")
	}

	var norm float32
	for _, v := range vec {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm == 0 {
		return nil, fmt.Errorf("cannot normalize zero vector")
	}

	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = v / norm
	}

	return result, nil
}

// SimilarityResult represents a similarity search result
type SimilarityResult struct {
	ElementID  string
	Similarity float64
	Rank       int
}

// RankBySimilarity sorts similarity results by similarity score (descending)
type BySimilarity []SimilarityResult

func (s BySimilarity) Len() int           { return len(s) }
func (s BySimilarity) Less(i, j int) bool { return s[i].Similarity > s[j].Similarity }
func (s BySimilarity) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
