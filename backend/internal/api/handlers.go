package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Ajith-Bondili/spell-check/internal/guardrails"
	"github.com/Ajith-Bondili/spell-check/internal/llm"
	"github.com/Ajith-Bondili/spell-check/internal/spellcheck"
	"github.com/Ajith-Bondili/spell-check/internal/storage"
	"github.com/Ajith-Bondili/spell-check/internal/types"
)

const apiVersion = "0.3.0"

// Server holds API dependencies.
type Server struct {
	spellChecker    *spellcheck.SymSpell
	contextAnalyzer *llm.ContextAnalyzer
	guardrails      *guardrails.Guardrails
	store           *storage.Store
	config          *types.Config

	spellMu     sync.RWMutex
	customWords map[string]int64
}

// NewServer creates a new API server.
func NewServer(spellChecker *spellcheck.SymSpell, config *types.Config, store *storage.Store) *Server {
	server := &Server{
		spellChecker:    spellChecker,
		contextAnalyzer: llm.NewContextAnalyzer(),
		guardrails:      guardrails.NewGuardrails(),
		store:           store,
		config:          config,
		customWords:     make(map[string]int64),
	}
	server.syncCustomDictionaryFromStore()
	return server
}

// SpellHandler handles fast spell checks.
func (s *Server) SpellHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req types.CorrectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = s.store.RecordError()
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Text = normalizeWord(req.Text)
	req.Context = strings.TrimSpace(req.Context)
	req.Domain = normalizeDomain(req.Domain)
	req.SessionID = strings.TrimSpace(req.SessionID)

	if req.Text == "" {
		_ = s.store.RecordError()
		writeError(w, http.StatusBadRequest, "text field is required")
		return
	}

	_ = s.store.RecordSpellRequest()
	settings, _ := s.store.ResolveSettings(req.Domain)

	if !settings.Enabled {
		writeJSON(w, http.StatusOK, s.skipResponse(req.Text, "spell", settings.Mode, "disabled", start))
		return
	}
	if s.store.IsWordIgnored(req.Text) {
		writeJSON(w, http.StatusOK, s.skipResponse(req.Text, "spell", settings.Mode, "ignored_word", start))
		return
	}
	if skip, reason := s.guardrails.ShouldSkipWord(req.Text, req.Context); skip {
		writeJSON(w, http.StatusOK, s.skipResponse(req.Text, "spell", settings.Mode, reason, start))
		return
	}

	candidates := s.lookup(req.Text)
	candidates = s.applyStoreSignals(req.Text, candidates, settings)

	resp := s.decide(req.Text, candidates, settings, "spell")
	if resp.BestCandidate != nil {
		resp.CorrectionID = s.store.NewCorrectionID()
	}
	if resp.ShouldAutoCorrect {
		resp.UndoTTLms = s.store.GetUndoTTL()
	}
	resp.ProcessingTimeMs = time.Since(start).Milliseconds()
	writeJSON(w, http.StatusOK, resp)
}

// RescoreHandler handles context-aware checks.
func (s *Server) RescoreHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req types.CorrectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = s.store.RecordError()
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Text = normalizeWord(req.Text)
	req.Context = strings.TrimSpace(req.Context)
	req.Domain = normalizeDomain(req.Domain)
	req.SessionID = strings.TrimSpace(req.SessionID)
	if req.Text == "" || req.Context == "" {
		_ = s.store.RecordError()
		writeError(w, http.StatusBadRequest, "text and context fields are required")
		return
	}

	_ = s.store.RecordRescoreRequest()
	settings, _ := s.store.ResolveSettings(req.Domain)

	if !settings.Enabled {
		writeJSON(w, http.StatusOK, s.skipResponse(req.Text, "rescore", settings.Mode, "disabled", start))
		return
	}
	if s.store.IsWordIgnored(req.Text) {
		writeJSON(w, http.StatusOK, s.skipResponse(req.Text, "rescore", settings.Mode, "ignored_word", start))
		return
	}
	if skip, reason := s.guardrails.ShouldSkipContext(req.Context); skip {
		writeJSON(w, http.StatusOK, s.skipResponse(req.Text, "rescore", settings.Mode, reason, start))
		return
	}
	if skip, reason := s.guardrails.ShouldSkipWord(req.Text, req.Context); skip {
		writeJSON(w, http.StatusOK, s.skipResponse(req.Text, "rescore", settings.Mode, reason, start))
		return
	}

	base := s.lookup(req.Text)
	withConfusables := s.contextAnalyzer.AddConfusableCandidates(req.Text, base)
	rescored := s.contextAnalyzer.AnalyzeContext(req.Text, req.Context, withConfusables)
	rescored = s.applyStoreSignals(req.Text, rescored, settings)

	resp := s.decide(req.Text, rescored, settings, "rescore")
	if resp.BestCandidate != nil {
		resp.CorrectionID = s.store.NewCorrectionID()
	}
	if resp.ShouldAutoCorrect {
		resp.UndoTTLms = s.store.GetUndoTTL()
	}
	resp.ProcessingTimeMs = time.Since(start).Milliseconds()
	writeJSON(w, http.StatusOK, resp)
}

// HealthHandler checks backend health.
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	settings := s.store.GetSettings()
	profiles := s.store.GetDomainProfiles()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":               "healthy",
		"version":              apiVersion,
		"mode":                 settings.Mode,
		"enabled":              settings.Enabled,
		"state_dir":            s.config.StateDir,
		"domain_profile_count": len(profiles),
	})
}

func (s *Server) lookup(word string) []types.Candidate {
	s.spellMu.RLock()
	defer s.spellMu.RUnlock()
	return s.spellChecker.Lookup(word)
}

func (s *Server) applyStoreSignals(original string, candidates []types.Candidate, settings storage.Settings) []types.Candidate {
	normalizedOriginal := normalizeWord(original)
	filtered := make([]types.Candidate, 0, len(candidates)+1)

	hasExact := false
	for _, candidate := range candidates {
		if candidate.Word == normalizedOriginal {
			hasExact = true
		}
		if s.store.IsPairIgnored(normalizedOriginal, candidate.Word) {
			continue
		}

		adjusted := candidate
		if freq, ok := s.store.GetCustomWordFrequency(candidate.Word); ok {
			if adjusted.Frequency < freq {
				adjusted.Frequency = freq
			}
			adjusted.Confidence = clampConfidence(adjusted.Confidence + 0.08)
		}

		if feedbackDelta := s.store.FeedbackAdjustment(normalizedOriginal, candidate.Word); feedbackDelta != 0 {
			adjusted.Confidence = clampConfidence(adjusted.Confidence + feedbackDelta)
		}

		filtered = append(filtered, adjusted)
	}

	if !hasExact {
		if freq, ok := s.store.GetCustomWordFrequency(normalizedOriginal); ok {
			filtered = append(filtered, types.Candidate{
				Word:         normalizedOriginal,
				Confidence:   1.0,
				EditDistance: 0,
				Frequency:    freq,
			})
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Confidence == filtered[j].Confidence {
			if filtered[i].Frequency == filtered[j].Frequency {
				return filtered[i].EditDistance < filtered[j].EditDistance
			}
			return filtered[i].Frequency > filtered[j].Frequency
		}
		return filtered[i].Confidence > filtered[j].Confidence
	})

	if settings.MaxSuggestions > 0 && len(filtered) > settings.MaxSuggestions {
		filtered = filtered[:settings.MaxSuggestions]
	}
	return filtered
}

func (s *Server) skipResponse(original, source, mode, reason string, start time.Time) types.CorrectionResponse {
	_ = s.store.RecordSkipReason(reason)
	return types.CorrectionResponse{
		Original:         original,
		Candidates:       []types.Candidate{},
		ProcessingTimeMs: time.Since(start).Milliseconds(),
		Source:           source,
		Reason:           reason,
		DecisionMode:     mode,
		Skipped:          true,
		Explanation:      "Skipped to avoid changing protected text.",
	}
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func normalizeWord(word string) string {
	word = strings.TrimSpace(strings.ToLower(word))
	word = strings.Trim(word, " \t\n\r.,!?;:\"()[]{}")
	return word
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "www.")
	if idx := strings.IndexRune(domain, '/'); idx >= 0 {
		domain = domain[:idx]
	}
	return strings.Trim(domain, ".")
}

func (s *Server) syncCustomDictionaryFromStore() {
	words := s.store.ListCustomWords()
	desired := make(map[string]int64, len(words))
	for _, entry := range words {
		desired[entry.Word] = entry.Frequency
	}

	s.spellMu.Lock()
	defer s.spellMu.Unlock()

	for word := range s.customWords {
		if _, ok := desired[word]; !ok {
			s.spellChecker.RemoveWord(word)
			delete(s.customWords, word)
		}
	}
	for word, freq := range desired {
		if existing, ok := s.customWords[word]; ok && existing == freq {
			continue
		}
		s.spellChecker.AddWord(word, freq)
		s.customWords[word] = freq
	}
}

func (s *Server) addCustomWordToSpell(word string, frequency int64) {
	s.spellMu.Lock()
	defer s.spellMu.Unlock()
	s.spellChecker.AddWord(word, frequency)
	s.customWords[word] = frequency
}

func (s *Server) removeCustomWordFromSpell(word string) {
	s.spellMu.Lock()
	defer s.spellMu.Unlock()
	s.spellChecker.RemoveWord(word)
	delete(s.customWords, word)
}

// CORSMiddleware handles CORS preflight requests.
func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			setCORSHeaders(w)
			w.WriteHeader(http.StatusOK)
			return
		}
		setCORSHeaders(w)
		next(w, r)
	}
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
