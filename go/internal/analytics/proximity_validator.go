package analytics

import (
	"regexp"
	"strings"
	"unicode"

	snowball "github.com/kljensen/snowball/english"
)

// Match represents a regex match with its nearby proximity words
type Match struct {
	PhraseStart int
	PhraseEnd   int
	PhraseText  string
	NearbyWords []ProximityWord
}

// ProximityWord represents a word found near a matched phrase
type ProximityWord struct {
	Word      string
	Position  int // word index in document
	Distance  int // word distance from phrase
	ByteStart int
	ByteEnd   int
}

// WordPosition tracks a word's location in the document
type WordPosition struct {
	Word      string
	Stemmed   string
	Index     int // word index in document
	ByteStart int
	ByteEnd   int
}

// FindProximityMatches finds words within distance threshold of a regex match
//
// Parameters:
//   - text: The full content text to search
//   - pattern: Regex pattern to find entities/phrases
//   - targetWords: Words that must appear near the matched phrase
//   - threshold: Maximum word distance allowed
//
// Returns: Slice of matches where at least one target word appears within threshold
func FindProximityMatches(text string, pattern string, targetWords []string, threshold int) ([]Match, error) {
	// 1. Find the phrase with regex
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	matches := re.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	// 2. Tokenize the entire text into words with positions
	words := tokenizeWithPositions(text)

	// 3. Stem target words
	stemmedTargets := make(map[string]bool)
	for _, word := range targetWords {
		stemmed := snowball.Stem(strings.ToLower(word), false)
		stemmedTargets[stemmed] = true
	}

	var results []Match

	// 4. For each regex match, find nearby words
	for _, match := range matches {
		startByte, endByte := match[0], match[1]

		// Find word positions around the match
		startWordIdx := findWordIndexAtByte(words, startByte)
		endWordIdx := findWordIndexAtByte(words, endByte)

		// Search within threshold distance
		proximityWords := findWordsInRange(words, startWordIdx, endWordIdx, threshold, stemmedTargets)

		if len(proximityWords) > 0 {
			results = append(results, Match{
				PhraseStart: startByte,
				PhraseEnd:   endByte,
				PhraseText:  text[startByte:endByte],
				NearbyWords: proximityWords,
			})
		}
	}

	return results, nil
}

// tokenizeWithPositions splits text into words with their positions
func tokenizeWithPositions(text string) []WordPosition {
	var words []WordPosition
	wordIdx := 0
	inWord := false
	wordStart := 0
	currentWord := strings.Builder{}

	for i, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if !inWord {
				wordStart = i
				inWord = true
			}
			currentWord.WriteRune(r)
		} else {
			if inWord {
				word := currentWord.String()
				stemmed := snowball.Stem(strings.ToLower(word), false)
				words = append(words, WordPosition{
					Word:      word,
					Stemmed:   stemmed,
					Index:     wordIdx,
					ByteStart: wordStart,
					ByteEnd:   i,
				})
				wordIdx++
				currentWord.Reset()
				inWord = false
			}
		}
	}

	// Handle last word
	if inWord {
		word := currentWord.String()
		stemmed := snowball.Stem(strings.ToLower(word), false)
		words = append(words, WordPosition{
			Word:      word,
			Stemmed:   stemmed,
			Index:     wordIdx,
			ByteStart: wordStart,
			ByteEnd:   len(text),
		})
	}

	return words
}

// findWordIndexAtByte finds the word index at a given byte position
func findWordIndexAtByte(words []WordPosition, bytePos int) int {
	for i, word := range words {
		if bytePos >= word.ByteStart && bytePos <= word.ByteEnd {
			return i
		}
		if bytePos < word.ByteStart {
			return i
		}
	}
	if len(words) > 0 {
		return len(words) - 1
	}
	return 0
}

// findWordsInRange searches for target words within threshold distance of matched phrase
func findWordsInRange(words []WordPosition, startIdx, endIdx, threshold int, targetStems map[string]bool) []ProximityWord {
	var results []ProximityWord

	// Define search range
	searchStart := max(0, startIdx-threshold)
	searchEnd := min(len(words)-1, endIdx+threshold)

	for i := searchStart; i <= searchEnd; i++ {
		// Skip words within the matched phrase itself
		if i >= startIdx && i <= endIdx {
			continue
		}

		word := words[i]
		if targetStems[word.Stemmed] {
			// Calculate word distance (how many words away)
			var distance int
			if i < startIdx {
				distance = startIdx - i
			} else {
				distance = i - endIdx
			}

			if distance <= threshold {
				results = append(results, ProximityWord{
					Word:      word.Word,
					Position:  word.Index,
					Distance:  distance,
					ByteStart: word.ByteStart,
					ByteEnd:   word.ByteEnd,
				})
			}
		}
	}

	return results
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
