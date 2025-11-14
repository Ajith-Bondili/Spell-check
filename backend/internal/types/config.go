package types

import "time"

// DefaultConfig returns sensible defaults for the application
func DefaultConfig() *Config {
	return &Config{
		// Server listens on localhost only (security!)
		Port: 8080,
		Host: "127.0.0.1",

		// SymSpell settings
		// Edit distance 2 catches most typos while staying fast
		MaxEditDistance: 2,
		DictionaryPath:  "data/test_dictionary.txt",

		// LLM settings (we'll add the model later)
		ModelPath:     "data/models/phi-2-Q4_K_M.gguf",
		ContextLength: 512, // Keep it small for speed
		UseGPU:        false, // CPU-only for now

		// Thresholds based on testing
		// 0.9 = very confident, auto-correct immediately
		// 0.5 = somewhat confident, show as suggestion
		AutoCorrectThreshold: 0.9,
		SuggestionThreshold:  0.5,
	}
}

// Timeouts for different operations
const (
	// Fast layer must respond instantly
	FastCorrectionTimeout = 50 * time.Millisecond

	// LLM layer can take a bit longer
	LLMCorrectionTimeout = 300 * time.Millisecond

	// Overall request timeout
	RequestTimeout = 500 * time.Millisecond
)
