package spellcheck

import (
	"testing"
)

func TestSymSpellBasic(t *testing.T) {
	// Create a new SymSpell instance
	ss := NewSymSpell(2)

	// Add some test words
	ss.AddWord("hello", 1000000)
	ss.AddWord("world", 900000)
	ss.AddWord("help", 500000)
	ss.AddWord("held", 300000)

	// Test 1: Exact match should return the word
	t.Run("ExactMatch", func(t *testing.T) {
		results := ss.Lookup("hello")
		if len(results) == 0 {
			t.Fatal("Expected results for 'hello', got none")
		}
		if results[0].Word != "hello" {
			t.Errorf("Expected 'hello', got '%s'", results[0].Word)
		}
		if results[0].Confidence != 1.0 {
			t.Errorf("Expected confidence 1.0 for exact match, got %f", results[0].Confidence)
		}
	})

	// Test 2: Single character typo
	t.Run("SingleTypo", func(t *testing.T) {
		results := ss.Lookup("helo") // missing 'l'
		if len(results) == 0 {
			t.Fatal("Expected suggestions for 'helo', got none")
		}

		// Should suggest "hello" and "help"
		found := false
		for _, r := range results {
			if r.Word == "hello" {
				found = true
				if r.EditDistance != 1 {
					t.Errorf("Expected edit distance 1 for 'helo'->'hello', got %d", r.EditDistance)
				}
			}
		}

		if !found {
			t.Error("Expected 'hello' in suggestions for 'helo'")
		}
	})

	// Test 3: Two character typo
	t.Run("DoubleTypo", func(t *testing.T) {
		results := ss.Lookup("hllo") // missing 'e'
		if len(results) == 0 {
			t.Fatal("Expected suggestions for 'hllo', got none")
		}

		found := false
		for _, r := range results {
			if r.Word == "hello" {
				found = true
			}
		}

		if !found {
			t.Error("Expected 'hello' in suggestions for 'hllo'")
		}
	})
}

func TestEditDistance(t *testing.T) {
	tests := []struct {
		s1   string
		s2   string
		want int
	}{
		{"", "", 0},
		{"hello", "hello", 0},
		{"hello", "helo", 1},  // one deletion
		{"hello", "hallo", 1}, // one substitution
		{"hello", "helloo", 1}, // one insertion
		{"kitten", "sitting", 3}, // classic example
		{"the", "teh", 2}, // two operations needed
	}

	for _, tt := range tests {
		t.Run(tt.s1+"->"+tt.s2, func(t *testing.T) {
			got := editDistance(tt.s1, tt.s2)
			if got != tt.want {
				t.Errorf("editDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, got, tt.want)
			}
		})
	}
}

func TestLoadDictionary(t *testing.T) {
	ss := NewSymSpell(2)

	// Try loading test dictionary
	err := ss.LoadDictionary("../../data/test_dictionary.txt")
	if err != nil {
		t.Fatalf("Failed to load dictionary: %v", err)
	}

	// Verify some words were loaded
	if len(ss.dictionary) == 0 {
		t.Fatal("Dictionary is empty after loading")
	}

	// Test common typos
	t.Run("CommonTypos", func(t *testing.T) {
		typos := map[string]string{
			"teh":        "the",
			"recieve":    "receive",
			"seperate":   "separate",
			"occured":    "occurred",
			"definately": "definitely",
		}

		for typo, correct := range typos {
			results := ss.Lookup(typo)
			if len(results) == 0 {
				t.Errorf("No suggestions for typo '%s'", typo)
				continue
			}

			// Check if the correct word is in suggestions
			found := false
			for _, r := range results {
				if r.Word == correct {
					found = true
					t.Logf("✓ '%s' -> '%s' (confidence: %.2f)", typo, r.Word, r.Confidence)
					break
				}
			}

			if !found {
				t.Errorf("Expected '%s' in suggestions for '%s', got: %v",
					correct, typo, results[:min3(3, len(results))])
			}
		}
	})
}

func TestRemoveWord(t *testing.T) {
	ss := NewSymSpell(2)
	ss.AddWord("localbrandname", 1500000)

	results := ss.Lookup("localbrandname")
	if len(results) == 0 || results[0].Word != "localbrandname" {
		t.Fatalf("expected added word to appear in lookup, got %+v", results)
	}

	ss.RemoveWord("localbrandname")
	results = ss.Lookup("localbrandname")
	for _, candidate := range results {
		if candidate.Word == "localbrandname" {
			t.Fatalf("word should have been removed, got candidates %+v", results)
		}
	}
}

// Helper to get first N elements
func min3(a, b int) int {
	if a < b {
		return a
	}
	return b
}
