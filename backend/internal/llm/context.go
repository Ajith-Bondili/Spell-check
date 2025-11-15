package llm

import (
	"strings"

	"github.com/Ajith-Bondili/spell-check/internal/types"
)

// ContextAnalyzer analyzes text context to help with disambiguation
type ContextAnalyzer struct {
	// Common confusables and their rules
	confusables map[string][]Confusable
}

// Confusable represents a commonly confused word pair
type Confusable struct {
	Word     string
	Patterns []string // Patterns that indicate this word should be used
	Examples []string // Example sentences
}

// NewContextAnalyzer creates a new context analyzer
func NewContextAnalyzer() *ContextAnalyzer {
	ca := &ContextAnalyzer{
		confusables: make(map[string][]Confusable),
	}

	ca.loadConfusables()
	return ca
}

// loadConfusables loads common confusable word pairs
func (ca *ContextAnalyzer) loadConfusables() {
	// their/there/they're
	ca.confusables["their"] = []Confusable{
		{
			Word: "their",
			Patterns: []string{
				"their house", "their car", "their book",
				"their own", "their family", "their friend",
			},
			Examples: []string{"I went to their house", "Their dog is cute"},
		},
		{
			Word: "there",
			Patterns: []string{
				"over there", "there is", "there are", "there was",
				"go there", "right there", "out there",
			},
			Examples: []string{"I went there yesterday", "There is a problem"},
		},
		{
			Word: "they're",
			Patterns: []string{
				"they're going", "they're coming", "they're here",
				"they're not", "they're ready", "they're happy",
			},
			Examples: []string{"They're going to the store", "They're happy"},
		},
	}

	// to/too/two
	ca.confusables["to"] = []Confusable{
		{
			Word: "to",
			Patterns: []string{
				"to go", "to be", "to have", "want to", "going to",
				"to the", "to a", "to an", "listen to", "talk to",
			},
			Examples: []string{"I want to go", "Listen to me"},
		},
		{
			Word: "too",
			Patterns: []string{
				"too much", "too many", "too late", "too early",
				"me too", "you too", "too big", "too small",
			},
			Examples: []string{"It's too late", "Me too"},
		},
		{
			Word: "two",
			Patterns: []string{
				"two people", "two dogs", "two books", "two days",
				"two of", "two more", "one or two",
			},
			Examples: []string{"I have two dogs", "Two people came"},
		},
	}

	// your/you're
	ca.confusables["your"] = []Confusable{
		{
			Word: "your",
			Patterns: []string{
				"your house", "your car", "your name", "your book",
				"your friend", "your family", "your dog",
			},
			Examples: []string{"What's your name?", "Your car is nice"},
		},
		{
			Word: "you're",
			Patterns: []string{
				"you're going", "you're right", "you're welcome",
				"you're not", "you're here", "you're sure",
				"you're absolutely", "you're so", "you're very",
			},
			Examples: []string{"You're right", "You're welcome"},
		},
	}

	// its/it's
	ca.confusables["its"] = []Confusable{
		{
			Word: "its",
			Patterns: []string{
				"its own", "its tail", "its color", "its name",
				"on its", "in its", "with its",
			},
			Examples: []string{"The dog wagged its tail", "Each has its own"},
		},
		{
			Word: "it's",
			Patterns: []string{
				"it's a", "it's the", "it's not", "it's been",
				"it's going", "it's time", "it's okay",
			},
			Examples: []string{"It's a nice day", "It's going well"},
		},
	}

	// affect/effect
	ca.confusables["affect"] = []Confusable{
		{
			Word: "affect",
			Patterns: []string{
				"will affect", "can affect", "may affect", "might affect",
				"doesn't affect", "won't affect", "could affect",
			},
			Examples: []string{"This will affect the outcome", "Don't let it affect you"},
		},
		{
			Word: "effect",
			Patterns: []string{
				"the effect", "an effect", "no effect", "side effect",
				"take effect", "in effect", "has an effect",
			},
			Examples: []string{"The effect was dramatic", "It has no effect"},
		},
	}

	// then/than
	ca.confusables["then"] = []Confusable{
		{
			Word: "then",
			Patterns: []string{
				"and then", "back then", "since then", "until then",
				"then we", "then I", "then he", "then she",
			},
			Examples: []string{"First we eat, then we go", "Back then it was different"},
		},
		{
			Word: "than",
			Patterns: []string{
				"better than", "more than", "less than", "rather than",
				"other than", "faster than", "bigger than",
			},
			Examples: []string{"Better than yesterday", "More than enough"},
		},
	}

	// lose/loose
	ca.confusables["lose"] = []Confusable{
		{
			Word: "lose",
			Patterns: []string{
				"lose weight", "don't lose", "can't lose", "won't lose",
				"will lose", "might lose", "could lose",
			},
			Examples: []string{"Don't lose hope", "I might lose"},
		},
		{
			Word: "loose",
			Patterns: []string{
				"loose fit", "came loose", "break loose", "hang loose",
				"is loose", "too loose", "very loose",
			},
			Examples: []string{"The screw is loose", "It's too loose"},
		},
	}

	// Create bidirectional mappings
	// E.g., if "their" → [their, there, they're], also add:
	//   "there" → [their, there, they're]
	//   "they're" → [their, there, they're]
	ca.createBidirectionalMappings()
}

// createBidirectionalMappings ensures all confusable words can look up their variants
func (ca *ContextAnalyzer) createBidirectionalMappings() {
	// Collect all unique confusable groups
	groups := make(map[string][]Confusable)

	// For each existing mapping, note all words in that group
	for _, confList := range ca.confusables {
		// Create a key from all words in this group (sorted for consistency)
		words := make([]string, len(confList))
		for i, conf := range confList {
			words[i] = conf.Word
		}

		// Use first word as group key
		groupKey := words[0]
		groups[groupKey] = confList
	}

	// Now add mappings for ALL words in each group
	for _, confList := range groups {
		for _, conf := range confList {
			// Each word in the group should map to the entire group
			ca.confusables[conf.Word] = confList
		}
	}
}

// AnalyzeContext analyzes the context around a word
// Returns enhanced candidates with context-based confidence adjustments
func (ca *ContextAnalyzer) AnalyzeContext(word string, context string, candidates []types.Candidate) []types.Candidate {
	if len(candidates) == 0 {
		return candidates
	}

	// Normalize
	word = strings.ToLower(strings.TrimSpace(word))
	context = strings.ToLower(context)

	// Check if the INPUT word itself is a confusable
	// This is CRITICAL: we should only apply confusable logic if the user
	// typed a word that's part of a confusable group
	confusables, isInputConfusable := ca.confusables[word]

	// If the input word is NOT a confusable, only apply confusable logic
	// if the word appears to be a typo (edit distance > 0)
	if !isInputConfusable {
		// Check if word appears in dictionary (edit distance 0)
		wordInDictionary := false
		for _, candidate := range candidates {
			if candidate.Word == word && candidate.EditDistance == 0 {
				wordInDictionary = true
				break
			}
		}

		// If word is in dictionary (correctly spelled), don't apply confusable logic
		// This prevents "the" from being penalized when "then" is a candidate
		if wordInDictionary {
			return candidates
		}

		// Word is NOT in dictionary (likely a typo)
		// Check if any candidate is a confusable
		hasConfusableCandidate := false
		for _, candidate := range candidates {
			if _, found := ca.confusables[candidate.Word]; found {
				confusables = ca.confusables[candidate.Word]
				hasConfusableCandidate = true
				break
			}
		}

		if !hasConfusableCandidate {
			return candidates // No confusables to analyze
		}
	}

	// Score each confusable based on context
	scores := make(map[string]float64)
	for _, conf := range confusables {
		scores[conf.Word] = ca.scoreConfusable(conf, context)
	}

	// Adjust candidate confidences based on context scores
	result := make([]types.Candidate, len(candidates))

	for i, candidate := range candidates {
		result[i] = candidate

		// If this candidate is one of the confusables, boost its confidence
		if contextScore, found := scores[candidate.Word]; found {
			// For confusables, context is KING!
			// Use 85% context, 15% spelling
			// This ensures that "their house" beats "they house" even if
			// "they" has better edit distance
			blendedConfidence := (contextScore * 0.85) + (candidate.Confidence * 0.15)
			result[i].Confidence = blendedConfidence
		} else {
			// If confusables exist but this isn't one of them,
			// it's probably a spelling mistake rather than context error
			// Only apply penalty if the input word is a confusable
			if isInputConfusable {
				result[i].Confidence = candidate.Confidence * 0.55
			}
		}
	}

	// Re-sort by new confidence scores
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Confidence > result[i].Confidence {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// scoreConfusable scores how well a confusable matches the context
func (ca *ContextAnalyzer) scoreConfusable(conf Confusable, context string) float64 {
	score := 0.0
	matchCount := 0

	// Check each pattern
	for _, pattern := range conf.Patterns {
		// Try exact match first
		if strings.Contains(context, pattern) {
			matchCount++
			// Give higher weight to longer, more specific patterns
			patternWeight := float64(len(strings.Fields(pattern))) * 0.2
			score += 0.3 + patternWeight
			continue
		}

		// Try flexible match: replace the target word with wildcard
		// E.g., "their house" → check if context has "* house"
		if ca.flexiblePatternMatch(conf.Word, pattern, context) {
			matchCount++
			patternWeight := float64(len(strings.Fields(pattern))) * 0.15
			score += 0.25 + patternWeight
		}
	}

	// Cap the score at 1.0
	if score > 1.0 {
		score = 1.0
	}

	// If no patterns matched, give a low baseline score
	if matchCount == 0 {
		score = 0.3
	}

	return score
}

// flexiblePatternMatch checks if the pattern matches after replacing target word
// E.g., pattern="their house", context="there house" → extract "house" and check
func (ca *ContextAnalyzer) flexiblePatternMatch(targetWord, pattern, context string) bool {
	// Extract the non-target-word part of the pattern
	// E.g., "their house" → "house"
	patternWords := strings.Fields(pattern)
	if len(patternWords) == 0 {
		return false
	}

	// Find words in pattern that aren't the target word
	significantWords := []string{}
	for _, word := range patternWords {
		if !strings.EqualFold(word, targetWord) {
			significantWords = append(significantWords, word)
		}
	}

	if len(significantWords) == 0 {
		return false // Pattern was just the target word
	}

	// Check if ALL significant words appear in context
	foundAll := true
	for _, sigWord := range significantWords {
		if !strings.Contains(context, sigWord) {
			foundAll = false
			break
		}
	}

	return foundAll
}

// ExtractWordContext extracts the context around a word in a sentence
// Returns the sentence fragment that's most relevant
func ExtractWordContext(sentence string, word string, windowSize int) string {
	words := strings.Fields(sentence)
	word = strings.ToLower(word)

	// Find the word position
	wordIndex := -1
	for i, w := range words {
		if strings.ToLower(strings.Trim(w, ".,!?;:")) == word {
			wordIndex = i
			break
		}
	}

	if wordIndex == -1 {
		return sentence // Word not found, return whole sentence
	}

	// Extract window around the word
	start := wordIndex - windowSize
	if start < 0 {
		start = 0
	}

	end := wordIndex + windowSize + 1
	if end > len(words) {
		end = len(words)
	}

	return strings.Join(words[start:end], " ")
}
