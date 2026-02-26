package api

import (
	"strings"

	"github.com/Ajith-Bondili/spell-check/internal/storage"
	"github.com/Ajith-Bondili/spell-check/internal/types"
)

func (s *Server) decide(original string, candidates []types.Candidate, settings storage.Settings, source string) types.CorrectionResponse {
	resp := types.CorrectionResponse{
		Original:     original,
		Candidates:   candidates,
		Source:       source,
		DecisionMode: settings.Mode,
	}

	if len(candidates) == 0 {
		resp.Reason = "no_candidates"
		return resp
	}

	top := candidates[0]
	if strings.EqualFold(top.Word, normalizeWord(original)) {
		resp.Reason = "already_correct"
		return resp
	}

	autoThreshold := settings.AutoCorrectThreshold
	switch settings.Mode {
	case storage.ModeAggressive:
		autoThreshold -= 0.12
		if autoThreshold < 0.55 {
			autoThreshold = 0.55
		}
	case storage.ModeSuggestions:
		// No-op: this mode never auto-corrects.
	default:
		// Conservative mode gets an extra safety gate:
		// we avoid auto-correcting distance-2 suggestions.
		if top.EditDistance > 1 && top.Confidence >= settings.SuggestionThreshold {
			resp.BestCandidate = &top
			resp.ShouldAutoCorrect = false
			resp.Reason = "conservative_distance_gate"
			_ = s.store.RecordSuggestion()
			return resp
		}
	}

	if settings.Mode != storage.ModeSuggestions && top.Confidence >= autoThreshold {
		resp.BestCandidate = &top
		resp.ShouldAutoCorrect = true
		resp.Reason = "auto_correct"
		_ = s.store.RecordAutoCorrect()
		return resp
	}

	if top.Confidence >= settings.SuggestionThreshold {
		resp.BestCandidate = &top
		resp.ShouldAutoCorrect = false
		if settings.Mode == storage.ModeSuggestions {
			resp.Reason = "suggestions_only_mode"
		} else {
			resp.Reason = "suggestion"
		}
		_ = s.store.RecordSuggestion()
		return resp
	}

	resp.Reason = "low_confidence"
	return resp
}
