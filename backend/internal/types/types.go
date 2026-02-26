package types

// CorrectionRequest represents a request from the browser extension
// to check and potentially correct text
type CorrectionRequest struct {
	// The word or phrase to check
	Text string `json:"text"`

	// Surrounding context for better corrections
	// e.g., "I want to meet you" helps choose "meet" over "meat"
	Context string `json:"context,omitempty"`

	// Position of the word in the context (for future use)
	Position int `json:"position,omitempty"`

	// Current website domain (used for per-domain profiles)
	Domain string `json:"domain,omitempty"`

	// Session identifier from extension
	SessionID string `json:"session_id,omitempty"`

	// Optional cursor token for future reconciliation
	CursorToken string `json:"cursor_token,omitempty"`
}

// Candidate represents a potential correction for a word
type Candidate struct {
	// The suggested word
	Word string `json:"word"`

	// Confidence score (0.0 to 1.0)
	// 0.9+ = auto-correct
	// 0.5-0.9 = show suggestion
	// <0.5 = ignore
	Confidence float64 `json:"confidence"`

	// Edit distance from original (for SymSpell)
	EditDistance int `json:"edit_distance"`

	// Word frequency in our dictionary (higher = more common)
	Frequency int64 `json:"frequency"`
}

// CorrectionResponse is what we send back to the extension
type CorrectionResponse struct {
	// Original text that was checked
	Original string `json:"original"`

	// List of possible corrections, sorted by confidence
	Candidates []Candidate `json:"candidates"`

	// The top suggestion (highest confidence)
	// nil if no good correction found
	BestCandidate *Candidate `json:"best_candidate,omitempty"`

	// Whether we should auto-apply this correction
	ShouldAutoCorrect bool `json:"should_auto_correct"`

	// Processing time in milliseconds (for debugging)
	ProcessingTimeMs int64 `json:"processing_time_ms"`

	// Which correction source produced this response (spell, rescore, skip)
	Source string `json:"source,omitempty"`

	// High-level reason for the final decision
	Reason string `json:"reason,omitempty"`

	// Decision mode used for this request
	DecisionMode string `json:"decision_mode,omitempty"`

	// True if the request was intentionally skipped
	Skipped bool `json:"skipped"`

	// Unique correction identifier (set when we provide a best candidate)
	CorrectionID string `json:"correction_id,omitempty"`

	// User-facing explanation for why this decision happened
	Explanation string `json:"explanation,omitempty"`

	// Time window where undo is shown for auto-corrections
	UndoTTLms int `json:"undo_ttl_ms,omitempty"`
}

// Config holds application configuration
type Config struct {
	// Server settings
	Port int    `json:"port"`
	Host string `json:"host"`

	// Spell checker settings
	MaxEditDistance int    `json:"max_edit_distance"`
	DictionaryPath  string `json:"dictionary_path"`

	// LLM settings
	ModelPath     string `json:"model_path"`
	ContextLength int    `json:"context_length"`
	UseGPU        bool   `json:"use_gpu"`

	// Confidence thresholds
	AutoCorrectThreshold float64 `json:"auto_correct_threshold"`
	SuggestionThreshold  float64 `json:"suggestion_threshold"`

	// Runtime state storage
	StateDir string `json:"state_dir"`

	// Decision behavior
	DefaultMode    string `json:"default_mode"`
	MaxSuggestions int    `json:"max_suggestions"`
}
