package analytics

import (
	"testing"

	snowball "github.com/kljensen/snowball/english"
)

func TestTokenizeWithPositions(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantLen  int
		wantWord []struct {
			word      string
			stemmed   string
			index     int
			byteStart int
			byteEnd   int
		}
	}{
		{
			name:    "simple sentence",
			text:    "The cat runs quickly",
			wantLen: 4,
			wantWord: []struct {
				word      string
				stemmed   string
				index     int
				byteStart int
				byteEnd   int
			}{
				{"The", "the", 0, 0, 3},
				{"cat", "cat", 1, 4, 7},
				{"runs", "run", 2, 8, 12},
				{"quickly", "quick", 3, 13, 20},
			},
		},
		{
			name:    "empty text",
			text:    "",
			wantLen: 0,
		},
		{
			name:    "single word",
			text:    "hello",
			wantLen: 1,
			wantWord: []struct {
				word      string
				stemmed   string
				index     int
				byteStart int
				byteEnd   int
			}{
				{"hello", "hello", 0, 0, 5},
			},
		},
		{
			name:    "words with numbers",
			text:    "test123 abc456",
			wantLen: 2,
			wantWord: []struct {
				word      string
				stemmed   string
				index     int
				byteStart int
				byteEnd   int
			}{
				{"test123", "test123", 0, 0, 7},
				{"abc456", "abc456", 1, 8, 14},
			},
		},
		{
			name:    "punctuation separation",
			text:    "Hello, world! How are you?",
			wantLen: 5,
			wantWord: []struct {
				word      string
				stemmed   string
				index     int
				byteStart int
				byteEnd   int
			}{
				{"Hello", "hello", 0, 0, 5},
				{"world", "world", 1, 7, 12},
				{"How", "how", 2, 14, 17},
				{"are", "are", 3, 18, 21},
				{"you", "you", 4, 22, 25},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenizeWithPositions(tt.text)

			if len(got) != tt.wantLen {
				t.Errorf("tokenizeWithPositions() returned %d words, want %d", len(got), tt.wantLen)
				return
			}

			for i, want := range tt.wantWord {
				if i >= len(got) {
					break
				}
				if got[i].Word != want.word {
					t.Errorf("word[%d].Word = %q, want %q", i, got[i].Word, want.word)
				}
				if got[i].Stemmed != want.stemmed {
					t.Errorf("word[%d].Stemmed = %q, want %q", i, got[i].Stemmed, want.stemmed)
				}
				if got[i].Index != want.index {
					t.Errorf("word[%d].Index = %d, want %d", i, got[i].Index, want.index)
				}
				if got[i].ByteStart != want.byteStart {
					t.Errorf("word[%d].ByteStart = %d, want %d", i, got[i].ByteStart, want.byteStart)
				}
				if got[i].ByteEnd != want.byteEnd {
					t.Errorf("word[%d].ByteEnd = %d, want %d", i, got[i].ByteEnd, want.byteEnd)
				}
			}
		})
	}
}

func TestFindWordIndexAtByte(t *testing.T) {
	words := []WordPosition{
		{Word: "The", Index: 0, ByteStart: 0, ByteEnd: 3},
		{Word: "cat", Index: 1, ByteStart: 4, ByteEnd: 7},
		{Word: "runs", Index: 2, ByteStart: 8, ByteEnd: 12},
	}

	tests := []struct {
		name    string
		words   []WordPosition
		bytePos int
		want    int
	}{
		{
			name:    "start of first word",
			words:   words,
			bytePos: 0,
			want:    0,
		},
		{
			name:    "middle of second word",
			words:   words,
			bytePos: 5,
			want:    1,
		},
		{
			name:    "end of last word",
			words:   words,
			bytePos: 12,
			want:    2,
		},
		{
			name:    "before first word",
			words:   words,
			bytePos: -1,
			want:    0,
		},
		{
			name:    "after last word",
			words:   words,
			bytePos: 20,
			want:    2,
		},
		{
			name:    "empty word list",
			words:   []WordPosition{},
			bytePos: 5,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findWordIndexAtByte(tt.words, tt.bytePos)
			if got != tt.want {
				t.Errorf("findWordIndexAtByte() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFindWordsInRange(t *testing.T) {
	words := []WordPosition{
		{Word: "The", Stemmed: "the", Index: 0},
		{Word: "medical", Stemmed: "medic", Index: 1},
		{Word: "doctor", Stemmed: "doctor", Index: 2},
		{Word: "examined", Stemmed: "examin", Index: 3},
		{Word: "patient", Stemmed: "patient", Index: 4},
		{Word: "carefully", Stemmed: "care", Index: 5},
		{Word: "noting", Stemmed: "note", Index: 6},
		{Word: "symptoms", Stemmed: "symptom", Index: 7},
	}

	tests := []struct {
		name          string
		words         []WordPosition
		startIdx      int
		endIdx        int
		threshold     int
		targetStems   map[string]bool
		wantCount     int
		wantDistances []int
	}{
		{
			name:        "find nearby medical terms",
			words:       words,
			startIdx:    2, // "doctor"
			endIdx:      2,
			threshold:   2,
			targetStems: map[string]bool{"medic": true, "patient": true},
			wantCount:   2,
			wantDistances: []int{1, 2}, // "medical" at distance 1, "patient" at distance 2
		},
		{
			name:        "threshold too small",
			words:       words,
			startIdx:    2,
			endIdx:      2,
			threshold:   0,
			targetStems: map[string]bool{"medic": true},
			wantCount:   0,
		},
		{
			name:        "no matching targets",
			words:       words,
			startIdx:    2,
			endIdx:      2,
			threshold:   3,
			targetStems: map[string]bool{"nonexistent": true},
			wantCount:   0,
		},
		{
			name:        "large threshold",
			words:       words,
			startIdx:    3,
			endIdx:      3,
			threshold:   10,
			targetStems: map[string]bool{"medic": true, "symptom": true},
			wantCount:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findWordsInRange(tt.words, tt.startIdx, tt.endIdx, tt.threshold, tt.targetStems)

			if len(got) != tt.wantCount {
				t.Errorf("findWordsInRange() returned %d words, want %d", len(got), tt.wantCount)
				return
			}

			for i, want := range tt.wantDistances {
				if i >= len(got) {
					break
				}
				if got[i].Distance != want {
					t.Errorf("word[%d].Distance = %d, want %d", i, got[i].Distance, want)
				}
			}
		})
	}
}

func TestFindProximityMatches(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		pattern     string
		targetWords []string
		threshold   int
		wantMatches int
		wantErr     bool
	}{
		{
			name:        "medical term with nearby context",
			text:        "The medical doctor examined the patient carefully and noted the symptoms.",
			pattern:     `doctor`,
			targetWords: []string{"medical", "patient"},
			threshold:   3,
			wantMatches: 1,
			wantErr:     false,
		},
		{
			name:        "no proximity match - too far",
			text:        "Medical care is important. The system processes documents. A doctor examined patients.",
			pattern:     `doctor`,
			targetWords: []string{"medical"},
			threshold:   3,
			wantMatches: 0,
			wantErr:     false,
		},
		{
			name:        "multiple matches in text",
			text:        "The doctor examined the patient. Another doctor treated a different patient.",
			pattern:     `doctor`,
			targetWords: []string{"patient"},
			threshold:   5,
			wantMatches: 2,
			wantErr:     false,
		},
		{
			name:        "stemming variants match",
			text:        "The doctors were studying medical cases carefully.",
			pattern:     `doctors`,
			targetWords: []string{"study", "medicine"}, // stems to "studi", "medicin"
			threshold:   3,
			wantMatches: 1,
			wantErr:     false,
		},
		{
			name:        "no pattern matches",
			text:        "Some text without the entity",
			pattern:     `nonexistent`,
			targetWords: []string{"word"},
			threshold:   5,
			wantMatches: 0,
			wantErr:     false,
		},
		{
			name:        "invalid regex pattern",
			text:        "Some text",
			pattern:     `[invalid(`,
			targetWords: []string{"word"},
			threshold:   5,
			wantMatches: 0,
			wantErr:     true,
		},
		{
			name:        "empty text",
			text:        "",
			pattern:     `doctor`,
			targetWords: []string{"medical"},
			threshold:   3,
			wantMatches: 0,
			wantErr:     false,
		},
		{
			name:        "zero threshold",
			text:        "The doctor examined patient",
			pattern:     `doctor`,
			targetWords: []string{"patient"},
			threshold:   0,
			wantMatches: 0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindProximityMatches(tt.text, tt.pattern, tt.targetWords, tt.threshold)

			if (err != nil) != tt.wantErr {
				t.Errorf("FindProximityMatches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if len(got) != tt.wantMatches {
				t.Errorf("FindProximityMatches() returned %d matches, want %d", len(got), tt.wantMatches)
				return
			}

			// Verify each match has proximity words
			for i, match := range got {
				if len(match.NearbyWords) == 0 {
					t.Errorf("match[%d] has no nearby words", i)
				}
				if match.PhraseText == "" {
					t.Errorf("match[%d] has empty PhraseText", i)
				}
			}
		})
	}
}

func TestFindProximityMatches_RealWorldScenario(t *testing.T) {
	// Simulating a Wikipedia medical mining scenario
	text := `Evolution of Diagnostic Neuroradiology from 1904 to 1999. Radiology journal
published this article with doi:10.1148/radiology.217.2.r00nv45309. The field of
neuroradiology has advanced significantly over the past century with improvements in
imaging technology and diagnostic techniques.`

	pattern := `Neuroradiology`
	targetWords := []string{"journal", "doi", "radiology", "diagnostic"}
	threshold := 20

	matches, err := FindProximityMatches(text, pattern, targetWords, threshold)

	if err != nil {
		t.Fatalf("FindProximityMatches() error = %v", err)
	}

	if len(matches) == 0 {
		t.Fatal("Expected at least one match")
	}

	// First match should be "Diagnostic Neuroradiology"
	match := matches[0]
	if match.PhraseText != "Neuroradiology" {
		t.Errorf("PhraseText = %q, want %q", match.PhraseText, "Neuroradiology")
	}

	// Should find nearby words
	if len(match.NearbyWords) == 0 {
		t.Error("Expected nearby words to be found")
	}

	// Verify at least one target word is within threshold
	foundTargets := make(map[string]bool)
	for _, pw := range match.NearbyWords {
		stemmed := snowball.Stem(pw.Word, false)
		foundTargets[stemmed] = true

		if pw.Distance > threshold {
			t.Errorf("ProximityWord %q at distance %d exceeds threshold %d",
				pw.Word, pw.Distance, threshold)
		}
	}

	if len(foundTargets) == 0 {
		t.Error("No target words found within threshold")
	}

	t.Logf("Found %d proximity words: %v", len(match.NearbyWords), foundTargets)
}

func TestMaxMin(t *testing.T) {
	tests := []struct {
		name string
		a    int
		b    int
		max  int
		min  int
	}{
		{"positive numbers", 5, 10, 10, 5},
		{"negative numbers", -5, -10, -5, -10},
		{"mixed signs", -5, 10, 10, -5},
		{"equal values", 7, 7, 7, 7},
		{"zero and positive", 0, 10, 10, 0},
		{"zero and negative", 0, -10, 0, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMax := max(tt.a, tt.b)
			if gotMax != tt.max {
				t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, gotMax, tt.max)
			}

			gotMin := min(tt.a, tt.b)
			if gotMin != tt.min {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, gotMin, tt.min)
			}
		})
	}
}
