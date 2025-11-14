package llm

import (
	"strings"
	"testing"

	"github.com/Ajith-Bondili/spell-check/internal/types"
)

func TestContextAnalyzer_TheirThereTheyre(t *testing.T) {
	ca := NewContextAnalyzer()

	tests := []struct {
		name            string
		sentence        string
		word            string
		expectedTop     string
		minConfidence   float64
	}{
		{
			name:          "Their - possession",
			sentence:      "i went to their house yesterday",
			word:          "their",
			expectedTop:   "their",
			minConfidence: 0.5,
		},
		{
			name:          "There - location",
			sentence:      "i will go there tomorrow",
			word:          "there",
			expectedTop:   "there",
			minConfidence: 0.5,
		},
		{
			name:          "They're - contraction",
			sentence:      "they're going to the store",
			word:          "theyre",
			expectedTop:   "they're",
			minConfidence: 0.5,
		},
		{
			name:          "There is/are",
			sentence:      "there are many options available",
			word:          "there",
			expectedTop:   "there",
			minConfidence: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create candidate list with all three options
			candidates := []types.Candidate{
				{Word: "their", Confidence: 0.5, EditDistance: 1, Frequency: 1000000},
				{Word: "there", Confidence: 0.5, EditDistance: 1, Frequency: 1000000},
				{Word: "they're", Confidence: 0.5, EditDistance: 2, Frequency: 500000},
			}

			// Analyze context
			result := ca.AnalyzeContext(tt.word, tt.sentence, candidates)

			if len(result) == 0 {
				t.Fatal("Expected results, got none")
			}

			// Check that the top result is what we expect
			if result[0].Word != tt.expectedTop {
				t.Errorf("Expected top result '%s', got '%s'", tt.expectedTop, result[0].Word)
			}

			// Check confidence is reasonable
			if result[0].Confidence < tt.minConfidence {
				t.Errorf("Expected confidence >= %f, got %f", tt.minConfidence, result[0].Confidence)
			}

			t.Logf("✓ Context: '%s' → chose '%s' (confidence: %.2f)",
				tt.sentence, result[0].Word, result[0].Confidence)
		})
	}
}

func TestContextAnalyzer_ToTooTwo(t *testing.T) {
	ca := NewContextAnalyzer()

	tests := []struct {
		name          string
		sentence      string
		word          string
		expectedTop   string
	}{
		{
			name:        "To - infinitive",
			sentence:    "i want to go to the store",
			word:        "to",
			expectedTop: "to",
		},
		{
			name:        "Too - excessive",
			sentence:    "it is too late to start",
			word:        "too",
			expectedTop: "too",
		},
		{
			name:        "Two - number",
			sentence:    "i have two dogs at home",
			word:        "two",
			expectedTop: "two",
		},
		{
			name:        "Me too",
			sentence:    "i like that me too",
			word:        "too",
			expectedTop: "too",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := []types.Candidate{
				{Word: "to", Confidence: 0.5, EditDistance: 0, Frequency: 10000000},
				{Word: "too", Confidence: 0.5, EditDistance: 1, Frequency: 5000000},
				{Word: "two", Confidence: 0.5, EditDistance: 1, Frequency: 3000000},
			}

			result := ca.AnalyzeContext(tt.word, tt.sentence, candidates)

			if len(result) == 0 {
				t.Fatal("Expected results, got none")
			}

			if result[0].Word != tt.expectedTop {
				t.Errorf("Expected '%s', got '%s'", tt.expectedTop, result[0].Word)
				t.Logf("Full results: %+v", result)
			}

			t.Logf("✓ '%s' → '%s' (confidence: %.2f)",
				tt.sentence, result[0].Word, result[0].Confidence)
		})
	}
}

func TestContextAnalyzer_YourYoure(t *testing.T) {
	ca := NewContextAnalyzer()

	tests := []struct {
		name        string
		sentence    string
		expectedTop string
	}{
		{
			name:        "Your - possession",
			sentence:    "what is your name",
			expectedTop: "your",
		},
		{
			name:        "You're - contraction",
			sentence:    "you're welcome to join",
			expectedTop: "you're",
		},
		{
			name:        "You're right",
			sentence:    "you're absolutely right",
			expectedTop: "you're",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := []types.Candidate{
				{Word: "your", Confidence: 0.5, EditDistance: 1, Frequency: 5000000},
				{Word: "you're", Confidence: 0.5, EditDistance: 1, Frequency: 3000000},
			}

			result := ca.AnalyzeContext("your", tt.sentence, candidates)

			if len(result) == 0 {
				t.Fatal("Expected results, got none")
			}

			if result[0].Word != tt.expectedTop {
				t.Errorf("Expected '%s', got '%s'", tt.expectedTop, result[0].Word)
			}

			t.Logf("✓ '%s' → '%s'", tt.sentence, result[0].Word)
		})
	}
}

func TestContextAnalyzer_ItsIts(t *testing.T) {
	ca := NewContextAnalyzer()

	tests := []struct {
		name        string
		sentence    string
		expectedTop string
	}{
		{
			name:        "Its - possession",
			sentence:    "the dog wagged its tail",
			expectedTop: "its",
		},
		{
			name:        "It's - contraction",
			sentence:    "it's a beautiful day",
			expectedTop: "it's",
		},
		{
			name:        "It's been",
			sentence:    "it's been a long time",
			expectedTop: "it's",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := []types.Candidate{
				{Word: "its", Confidence: 0.5, EditDistance: 0, Frequency: 3000000},
				{Word: "it's", Confidence: 0.5, EditDistance: 1, Frequency: 2000000},
			}

			result := ca.AnalyzeContext("its", tt.sentence, candidates)

			if len(result) == 0 {
				t.Fatal("Expected results, got none")
			}

			if result[0].Word != tt.expectedTop {
				t.Errorf("Expected '%s', got '%s'", tt.expectedTop, result[0].Word)
			}

			t.Logf("✓ '%s' → '%s'", tt.sentence, result[0].Word)
		})
	}
}

func TestExtractWordContext(t *testing.T) {
	tests := []struct {
		name       string
		sentence   string
		word       string
		windowSize int
		wantLen    int // Expected number of words in result
	}{
		{
			name:       "Middle word",
			sentence:   "the quick brown fox jumps over the lazy dog",
			word:       "jumps",
			windowSize: 2,
			wantLen:    5, // brown fox jumps over the
		},
		{
			name:       "Start word",
			sentence:   "hello world this is a test",
			word:       "hello",
			windowSize: 2,
			wantLen:    3, // hello world this
		},
		{
			name:       "End word",
			sentence:   "this is the end",
			word:       "end",
			windowSize: 2,
			wantLen:    3, // is the end
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractWordContext(tt.sentence, tt.word, tt.windowSize)
			wordCount := len(strings.Fields(result))

			if wordCount != tt.wantLen {
				t.Errorf("Expected %d words, got %d: '%s'", tt.wantLen, wordCount, result)
			}

			t.Logf("✓ Extracted: '%s'", result)
		})
	}
}

func TestContextAnalyzer_NoConfusables(t *testing.T) {
	ca := NewContextAnalyzer()

	// Test with a word that has no confusables
	candidates := []types.Candidate{
		{Word: "hello", Confidence: 0.9, EditDistance: 0, Frequency: 1000000},
		{Word: "help", Confidence: 0.6, EditDistance: 1, Frequency: 500000},
	}

	result := ca.AnalyzeContext("hello", "hello world", candidates)

	// Should return candidates unchanged (or in same order)
	if len(result) != len(candidates) {
		t.Errorf("Expected %d candidates, got %d", len(candidates), len(result))
	}

	if result[0].Word != "hello" {
		t.Errorf("Expected 'hello' to remain top candidate")
	}

	t.Logf("✓ Non-confusable word handled correctly")
}
