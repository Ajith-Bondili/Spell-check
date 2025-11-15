package spellcheck

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Ajith-Bondili/spell-check/internal/types"
)

// SymSpell implements the SymSpell algorithm for fast spell checking
// Paper: https://github.com/wolfgarbe/SymSpell
type SymSpell struct {
	// Maps words to their frequency in the corpus
	// e.g., "the" → 23,135,851,162 (very common)
	dictionary map[string]int64

	// Maps potential misspellings to correct words
	// e.g., "teh" → ["the", "tea"]
	// This is the SECRET SAUCE - pre-computed for speed!
	deletes map[string][]string

	// Maximum edit distance to consider
	// 1 = catches 80% of typos, super fast
	// 2 = catches 99% of typos, still fast
	maxEditDistance int

	// Longest word in dictionary (for optimization)
	maxLength int
}

// NewSymSpell creates a new SymSpell instance
func NewSymSpell(maxEditDistance int) *SymSpell {
	return &SymSpell{
		dictionary:      make(map[string]int64),
		deletes:         make(map[string][]string),
		maxEditDistance: maxEditDistance,
		maxLength:       0,
	}
}

// LoadDictionary loads a frequency dictionary from a text file
// Format: word frequency
// Example:
//   the 23135851162
//   of 13151942776
//   and 12997637966
func (s *SymSpell) LoadDictionary(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open dictionary: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse "word frequency"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue // Skip malformed lines
		}

		word := strings.ToLower(parts[0])
		frequency, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			// If we can't parse frequency, use 1
			frequency = 1
		}

		// Add to dictionary
		s.AddWord(word, frequency)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading dictionary: %w", err)
	}

	return nil
}

// AddWord adds a word to the dictionary and generates deletion variants
func (s *SymSpell) AddWord(word string, frequency int64) {
	word = strings.ToLower(word)

	// Update dictionary
	if existing, found := s.dictionary[word]; !found || frequency > existing {
		s.dictionary[word] = frequency
	}

	// Update max length
	if len(word) > s.maxLength {
		s.maxLength = len(word)
	}

	// Generate all deletion variants up to maxEditDistance
	// This is the PRE-COMPUTATION step that makes lookups fast
	deletes := s.edits(word, 0, make(map[string]bool))

	for delete := range deletes {
		// Add this word as a suggestion for this deletion
		suggestions := s.deletes[delete]

		// Only add if not already present
		found := false
		for _, existing := range suggestions {
			if existing == word {
				found = true
				break
			}
		}

		if !found {
			s.deletes[delete] = append(suggestions, word)
		}
	}
}

// edits generates all strings within editDistance deletes from word
// This is RECURSIVE and builds up all possible deletions
func (s *SymSpell) edits(word string, depth int, result map[string]bool) map[string]bool {
	depth++

	if len(word) <= 1 {
		return result
	}

	// For each position in the word, try deleting that character
	for i := 0; i < len(word); i++ {
		// Delete character at position i
		delete := word[:i] + word[i+1:]

		if !result[delete] {
			result[delete] = true

			// Recurse if we haven't reached max depth
			if depth < s.maxEditDistance {
				s.edits(delete, depth, result)
			}
		}
	}

	return result
}

// Lookup finds suggestions for a potentially misspelled word
func (s *SymSpell) Lookup(word string) []types.Candidate {
	word = strings.ToLower(word)

	// If word is in dictionary, it might be correct
	// But we still return candidates in case it's a real-word error
	candidates := make(map[string]*types.Candidate)

	if freq, found := s.dictionary[word]; found {
		// Word exists, but might still be wrong in context
		// e.g., "there" exists but "their" might be correct
		candidates[word] = &types.Candidate{
			Word:         word,
			Confidence:   1.0, // Perfect match
			EditDistance: 0,
			Frequency:    freq,
		}
	}

	// Generate deletions of the input word
	deletes := s.edits(word, 0, make(map[string]bool))

	// For each deletion, find all words that could match
	for delete := range deletes {
		if suggestions, found := s.deletes[delete]; found {
			for _, suggestion := range suggestions {
				// Skip if we already have this candidate
				if _, exists := candidates[suggestion]; exists {
					continue
				}

				// Calculate actual edit distance
				editDist := editDistance(word, suggestion)

				if editDist <= s.maxEditDistance {
					freq := s.dictionary[suggestion]

					// Confidence based on edit distance and frequency
					// Closer match + higher frequency = higher confidence
					confidence := s.calculateConfidence(editDist, freq)

					candidates[suggestion] = &types.Candidate{
						Word:         suggestion,
						Confidence:   confidence,
						EditDistance: editDist,
						Frequency:    freq,
					}
				}
			}
		}
	}

	// Convert map to sorted slice
	result := make([]types.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, *candidate)
	}

	// Sort by confidence (highest first), then by frequency as tiebreaker
	// We'll use a simple bubble sort for now (will optimize later)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			// Primary: higher confidence wins
			if result[j].Confidence > result[i].Confidence {
				result[i], result[j] = result[j], result[i]
			} else if result[j].Confidence == result[i].Confidence {
				// Tiebreaker: if confidences are equal, prefer higher frequency
				if result[j].Frequency > result[i].Frequency {
					result[i], result[j] = result[j], result[i]
				}
			}
		}
	}

	return result
}

// calculateConfidence computes confidence score based on edit distance and frequency
func (s *SymSpell) calculateConfidence(editDist int, frequency int64) float64 {
	// Start with base confidence based on edit distance
	var baseConfidence float64
	switch editDist {
	case 0:
		baseConfidence = 1.0 // Perfect match
	case 1:
		baseConfidence = 0.70 // One character off (high confidence for single typos)
	case 2:
		baseConfidence = 0.45 // Two characters off (more conservative)
	default:
		baseConfidence = 0.20 // Further away
	}

	// Boost confidence for high-frequency words
	// Balanced to catch obvious typos without being too aggressive
	frequencyBoost := 0.0
	if frequency > 5000000000 {
		// Top ~10 most common words (like "the", "be", "and", "of")
		// These get the biggest boost - typos are extremely likely
		frequencyBoost = 0.32
	} else if frequency > 1000000000 {
		// Top ~100 most common words
		frequencyBoost = 0.24
	} else if frequency > 100000000 {
		// Top ~500 words
		frequencyBoost = 0.16
	} else if frequency > 10000000 {
		// Top ~2000 words
		frequencyBoost = 0.10
	} else if frequency > 1000000 {
		// Common words
		frequencyBoost = 0.05
	} else if frequency > 100000 {
		frequencyBoost = 0.02
	}

	confidence := baseConfidence + frequencyBoost

	// Cap at 0.95 for non-exact matches
	// This prevents typos from being "too confident"
	if editDist > 0 && confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}

// editDistance calculates Levenshtein distance between two strings
// This is the MINIMUM number of single-character edits needed to change one into the other
func editDistance(s1, s2 string) int {
	// Handle empty strings
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix for dynamic programming
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
