package llm

import (
	"strings"

	"github.com/Ajith-Bondili/spell-check/internal/types"
)

// AddConfusableCandidates adds known confusables to the candidate list
// This is important for catching real-word errors where the typed word
// is technically correct but wrong in context (e.g., "there" vs "their")
func (ca *ContextAnalyzer) AddConfusableCandidates(word string, candidates []types.Candidate) []types.Candidate {
	word = strings.ToLower(strings.TrimSpace(word))

	// Get confusables for this word
	confusables, found := ca.confusables[word]
	if !found {
		return candidates // No confusables, return as-is
	}

	// Create a map of existing candidates for quick lookup
	existing := make(map[string]bool)
	for _, candidate := range candidates {
		existing[candidate.Word] = true
	}

	// Add confusable words that aren't already in the list
	result := make([]types.Candidate, len(candidates))
	copy(result, candidates)

	for _, conf := range confusables {
		if !existing[conf.Word] {
			// Add this confusable as a candidate
			// Give it a high baseline since confusables are important
			// The context analyzer will adjust this based on actual context
			result = append(result, types.Candidate{
				Word:         conf.Word,
				Confidence:   0.7, // Higher baseline for confusables
				EditDistance: 1,   // Assume small edit distance
				Frequency:    1000000, // Reasonable frequency
			})
			existing[conf.Word] = true
		}
	}

	return result
}
