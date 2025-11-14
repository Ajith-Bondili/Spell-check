package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Ajith-Bondili/spell-check/internal/spellcheck"
	"github.com/Ajith-Bondili/spell-check/internal/types"
)

// Server holds our API dependencies
type Server struct {
	spellChecker *spellcheck.SymSpell
	config       *types.Config
}

// NewServer creates a new API server
func NewServer(spellChecker *spellcheck.SymSpell, config *types.Config) *Server {
	return &Server{
		spellChecker: spellChecker,
		config:       config,
	}
}

// SpellHandler handles the /spell endpoint (fast layer)
// This is called on SPACE - must be blazingly fast (<50ms)
func (s *Server) SpellHandler(w http.ResponseWriter, r *http.Request) {
	// Start timing for performance monitoring
	startTime := time.Now()

	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req types.CorrectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Text == "" {
		http.Error(w, "Text field is required", http.StatusBadRequest)
		return
	}

	// Lookup suggestions
	candidates := s.spellChecker.Lookup(req.Text)

	// Build response
	response := types.CorrectionResponse{
		Original:   req.Text,
		Candidates: candidates,
	}

	// If we have a high-confidence candidate, mark for auto-correct
	if len(candidates) > 0 {
		topCandidate := candidates[0]

		// Only auto-correct if:
		// 1. Confidence is high enough
		// 2. It's not the original word (no change needed)
		if topCandidate.Confidence >= s.config.AutoCorrectThreshold &&
			topCandidate.Word != req.Text {
			response.BestCandidate = &topCandidate
			response.ShouldAutoCorrect = true
		} else if topCandidate.Confidence >= s.config.SuggestionThreshold &&
			topCandidate.Word != req.Text {
			// Show as suggestion (user can tab to accept)
			response.BestCandidate = &topCandidate
			response.ShouldAutoCorrect = false
		}
	}

	// Calculate processing time
	response.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	// Set CORS headers (so browser extension can call us)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Send response
	json.NewEncoder(w).Encode(response)
}

// RescoreHandler handles the /rescore endpoint (smart layer with LLM)
// This is called on PUNCTUATION or PAUSE - can be slower (~200-300ms)
func (s *Server) RescoreHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req types.CorrectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Text == "" || req.Context == "" {
		http.Error(w, "Text and context fields are required", http.StatusBadRequest)
		return
	}

	// Step 1: Get fast suggestions from SymSpell
	candidates := s.spellChecker.Lookup(req.Text)

	// Step 2: Use LLM to rescore based on context
	// TODO: This will call the LLM layer when we implement it
	// For now, just return the fast layer results

	// Build response
	response := types.CorrectionResponse{
		Original:   req.Text,
		Candidates: candidates,
	}

	if len(candidates) > 0 {
		topCandidate := candidates[0]

		// For now, use same logic as fast layer
		// Later, we'll use LLM confidence scores
		if topCandidate.Confidence >= s.config.AutoCorrectThreshold &&
			topCandidate.Word != req.Text {
			response.BestCandidate = &topCandidate
			response.ShouldAutoCorrect = true
		} else if topCandidate.Confidence >= s.config.SuggestionThreshold &&
			topCandidate.Word != req.Text {
			response.BestCandidate = &topCandidate
			response.ShouldAutoCorrect = false
		}
	}

	response.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

// HealthHandler checks if the server is running
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"version": "0.1.0",
	})
}

// CORSMiddleware handles CORS preflight requests
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
