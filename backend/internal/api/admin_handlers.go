package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Ajith-Bondili/spell-check/internal/storage"
)

type settingsUpdateRequest struct {
	Enabled              *bool    `json:"enabled,omitempty"`
	Mode                 string   `json:"mode,omitempty"`
	AutoCorrectThreshold *float64 `json:"auto_correct_threshold,omitempty"`
	SuggestionThreshold  *float64 `json:"suggestion_threshold,omitempty"`
	MaxSuggestions       *int     `json:"max_suggestions,omitempty"`
	RespectSlang         *bool    `json:"respect_slang,omitempty"`
}

type addWordRequest struct {
	Word      string `json:"word"`
	Frequency int64  `json:"frequency,omitempty"`
}

type ignoreRequest struct {
	Word       string `json:"word,omitempty"`
	Original   string `json:"original,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

type feedbackRequest struct {
	Original   string `json:"original"`
	Suggestion string `json:"suggestion"`
	Accepted   bool   `json:"accepted"`
}

type appliedCorrectionRequest struct {
	CorrectionID string  `json:"correction_id,omitempty"`
	Original     string  `json:"original"`
	Suggestion   string  `json:"suggestion"`
	Domain       string  `json:"domain,omitempty"`
	Source       string  `json:"source,omitempty"`
	Mode         string  `json:"mode,omitempty"`
	Reason       string  `json:"reason,omitempty"`
	Explanation  string  `json:"explanation,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`
	SessionID    string  `json:"session_id,omitempty"`
	BeforeText   string  `json:"before_text,omitempty"`
	AfterText    string  `json:"after_text,omitempty"`
}

type undoRequest struct {
	CorrectionID string `json:"correction_id"`
}

// SettingsHandler manages runtime settings.
func (s *Server) SettingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.store.GetSettings())
		return
	case http.MethodPut:
		current := s.store.GetSettings()
		var req settingsUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid settings payload")
			return
		}

		if req.Enabled != nil {
			current.Enabled = *req.Enabled
		}
		if req.Mode != "" {
			current.Mode = req.Mode
		}
		if req.AutoCorrectThreshold != nil {
			current.AutoCorrectThreshold = *req.AutoCorrectThreshold
		}
		if req.SuggestionThreshold != nil {
			current.SuggestionThreshold = *req.SuggestionThreshold
		}
		if req.MaxSuggestions != nil {
			current.MaxSuggestions = *req.MaxSuggestions
		}
		if req.RespectSlang != nil {
			current.RespectSlang = *req.RespectSlang
		}

		if err := s.store.SetSettings(current); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, current)
		return
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// DictionaryHandler returns dictionary and ignore state.
func (s *Server) DictionaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"words":         s.store.ListCustomWords(),
		"ignored_words": s.store.ListIgnoredWords(),
		"ignored_pairs": s.store.ListIgnoredPairs(),
	})
}

// DictionaryWordsHandler handles add/remove custom words.
func (s *Server) DictionaryWordsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req addWordRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid word payload")
			return
		}

		entry, err := s.store.AddCustomWord(req.Word, req.Frequency)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.addCustomWordToSpell(entry.Word, entry.Frequency)
		writeJSON(w, http.StatusCreated, entry)
		return

	case http.MethodDelete:
		word := strings.TrimPrefix(r.URL.Path, "/dictionary/words/")
		word = strings.TrimSpace(word)
		if word == "" {
			writeError(w, http.StatusBadRequest, "word path parameter is required")
			return
		}

		removed, err := s.store.RemoveCustomWord(word)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !removed {
			writeError(w, http.StatusNotFound, "word not found")
			return
		}
		s.removeCustomWordFromSpell(normalizeWord(word))
		writeJSON(w, http.StatusOK, map[string]string{"status": "removed", "word": normalizeWord(word)})
		return

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// DictionaryIgnoreHandler stores ignore rules.
func (s *Server) DictionaryIgnoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req ignoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid ignore payload")
		return
	}

	if req.Word != "" {
		if err := s.store.AddIgnoredWord(req.Word); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored_word_added", "word": normalizeWord(req.Word)})
		return
	}

	if req.Original != "" && req.Suggestion != "" {
		if err := s.store.AddIgnoredPair(req.Original, req.Suggestion); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"status":     "ignored_pair_added",
			"original":   normalizeWord(req.Original),
			"suggestion": normalizeWord(req.Suggestion),
		})
		return
	}

	writeError(w, http.StatusBadRequest, "provide either word or original+suggestion")
}

// StatsHandler returns current runtime statistics.
func (s *Server) StatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	stats := s.store.GetStats()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stats":              stats,
		"custom_word_count":  len(s.store.ListCustomWords()),
		"ignored_word_count": len(s.store.ListIgnoredWords()),
		"ignored_pair_count": len(s.store.ListIgnoredPairs()),
	})
}

// StatsResetHandler resets statistics.
func (s *Server) StatsResetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := s.store.ResetStats(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset stats")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stats_reset"})
}

// FeedbackHandler stores user correction feedback.
func (s *Server) FeedbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req feedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid feedback payload")
		return
	}

	if err := s.store.RecordFeedback(req.Original, req.Suggestion, req.Accepted); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "feedback_recorded",
		"original":   normalizeWord(req.Original),
		"suggestion": normalizeWord(req.Suggestion),
		"accepted":   req.Accepted,
	})
}

// ReloadHandler reloads persisted state from disk.
func (s *Server) ReloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := s.store.Reload(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload state")
		return
	}
	s.syncCustomDictionaryFromStore()
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}

// ProfilesHandler returns all profile data.
func (s *Server) ProfilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"default": s.store.GetSettings(),
		"domains": s.store.GetDomainProfiles(),
	})
}

// ProfilesDefaultHandler gets/updates the default profile.
func (s *Server) ProfilesDefaultHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.store.GetSettings())
		return
	case http.MethodPut:
		current := s.store.GetSettings()
		var req settingsUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid profile payload")
			return
		}
		if req.Enabled != nil {
			current.Enabled = *req.Enabled
		}
		if req.Mode != "" {
			current.Mode = req.Mode
		}
		if req.AutoCorrectThreshold != nil {
			current.AutoCorrectThreshold = *req.AutoCorrectThreshold
		}
		if req.SuggestionThreshold != nil {
			current.SuggestionThreshold = *req.SuggestionThreshold
		}
		if req.MaxSuggestions != nil {
			current.MaxSuggestions = *req.MaxSuggestions
		}
		if req.RespectSlang != nil {
			current.RespectSlang = *req.RespectSlang
		}

		if err := s.store.SetSettings(current); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, current)
		return
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ProfilesDomainHandler gets/updates/deletes domain-specific profiles.
func (s *Server) ProfilesDomainHandler(w http.ResponseWriter, r *http.Request) {
	domain := strings.TrimPrefix(r.URL.Path, "/profiles/domain/")
	domain = strings.TrimSpace(domain)
	if domain == "" {
		writeError(w, http.StatusBadRequest, "domain path parameter is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		profile, found := s.store.GetDomainProfile(domain)
		if !found {
			writeError(w, http.StatusNotFound, "domain profile not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"domain":   normalizeDomain(domain),
			"profile":  profile,
			"resolved": true,
		})
		return

	case http.MethodPut:
		base := s.store.GetSettings()
		var req settingsUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid domain profile payload")
			return
		}
		if req.Enabled != nil {
			base.Enabled = *req.Enabled
		}
		if req.Mode != "" {
			base.Mode = req.Mode
		}
		if req.AutoCorrectThreshold != nil {
			base.AutoCorrectThreshold = *req.AutoCorrectThreshold
		}
		if req.SuggestionThreshold != nil {
			base.SuggestionThreshold = *req.SuggestionThreshold
		}
		if req.MaxSuggestions != nil {
			base.MaxSuggestions = *req.MaxSuggestions
		}
		if req.RespectSlang != nil {
			base.RespectSlang = *req.RespectSlang
		}

		profile, err := s.store.SetDomainProfile(domain, base)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"domain":  normalizeDomain(domain),
			"profile": profile,
			"status":  "saved",
		})
		return

	case http.MethodDelete:
		removed, err := s.store.DeleteDomainProfile(domain)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if !removed {
			writeError(w, http.StatusNotFound, "domain profile not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"domain": normalizeDomain(domain),
			"status": "deleted",
		})
		return
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// CorrectionAppliedHandler records an applied correction into the journal.
func (s *Server) CorrectionAppliedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req appliedCorrectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid correction payload")
		return
	}

	record, err := s.store.RecordAppliedCorrection(storage.CorrectionRecord{
		CorrectionID: req.CorrectionID,
		Original:     req.Original,
		Suggestion:   req.Suggestion,
		Domain:       req.Domain,
		Source:       req.Source,
		Mode:         req.Mode,
		Reason:       req.Reason,
		Explanation:  req.Explanation,
		Confidence:   req.Confidence,
		SessionID:    req.SessionID,
		BeforeText:   req.BeforeText,
		AfterText:    req.AfterText,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "recorded",
		"record": record,
	})
}

// UndoHandler marks a correction as undone in backend journal.
func (s *Server) UndoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req undoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid undo payload")
		return
	}

	record, found, err := s.store.UndoCorrection(req.CorrectionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "correction_id not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "undone",
		"record": record,
	})
}

// PainPointsHandler exposes top friction trends.
func (s *Server) PainPointsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	limit := 5
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"insights": s.store.GetPainPointInsights(limit),
		"limit":    limit,
	})
}

// DefaultSettingsFromConfig builds settings defaults from static config.
func DefaultSettingsFromConfig(configAuto, configSuggest float64, mode string, maxSuggestions int) storage.Settings {
	if mode == "" {
		mode = storage.ModeConservative
	}
	if maxSuggestions <= 0 {
		maxSuggestions = 5
	}
	return storage.Settings{
		Enabled:              true,
		Mode:                 mode,
		AutoCorrectThreshold: configAuto,
		SuggestionThreshold:  configSuggest,
		MaxSuggestions:       maxSuggestions,
		RespectSlang:         false,
	}
}
