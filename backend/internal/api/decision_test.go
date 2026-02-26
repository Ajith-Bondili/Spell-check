package api

import (
	"testing"

	"github.com/Ajith-Bondili/spell-check/internal/storage"
	"github.com/Ajith-Bondili/spell-check/internal/types"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	store, err := storage.NewStore(t.TempDir(), storage.Settings{
		Enabled:              true,
		Mode:                 storage.ModeConservative,
		AutoCorrectThreshold: 0.75,
		SuggestionThreshold:  0.5,
		MaxSuggestions:       5,
	})
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return &Server{store: store}
}

func TestDecideConservativeDistanceGate(t *testing.T) {
	server := newTestServer(t)
	settings := storage.Settings{
		Enabled:              true,
		Mode:                 storage.ModeConservative,
		AutoCorrectThreshold: 0.75,
		SuggestionThreshold:  0.5,
		MaxSuggestions:       5,
	}

	resp := server.decide("wierd", []types.Candidate{
		{Word: "weird", Confidence: 0.91, EditDistance: 2, Frequency: 1000},
	}, settings, "spell")

	if resp.ShouldAutoCorrect {
		t.Fatal("expected conservative mode to avoid auto-correct at distance 2")
	}
	if resp.BestCandidate == nil || resp.BestCandidate.Word != "weird" {
		t.Fatalf("expected weird suggestion, got %+v", resp.BestCandidate)
	}
	if resp.Reason != "conservative_distance_gate" {
		t.Fatalf("unexpected reason: %s", resp.Reason)
	}
}

func TestDecideAggressiveLowersAutoThreshold(t *testing.T) {
	server := newTestServer(t)
	settings := storage.Settings{
		Enabled:              true,
		Mode:                 storage.ModeAggressive,
		AutoCorrectThreshold: 0.78,
		SuggestionThreshold:  0.5,
		MaxSuggestions:       5,
	}

	resp := server.decide("seperate", []types.Candidate{
		{Word: "separate", Confidence: 0.69, EditDistance: 1, Frequency: 10000},
	}, settings, "spell")

	if !resp.ShouldAutoCorrect {
		t.Fatalf("expected aggressive mode to auto-correct, got %+v", resp)
	}
	if resp.Reason != "auto_correct" {
		t.Fatalf("unexpected reason: %s", resp.Reason)
	}
}

func TestDecideSuggestionsOnlyNeverAutocorrects(t *testing.T) {
	server := newTestServer(t)
	settings := storage.Settings{
		Enabled:              true,
		Mode:                 storage.ModeSuggestions,
		AutoCorrectThreshold: 0.75,
		SuggestionThreshold:  0.4,
		MaxSuggestions:       5,
	}

	resp := server.decide("teh", []types.Candidate{
		{Word: "the", Confidence: 0.95, EditDistance: 1, Frequency: 100000},
	}, settings, "spell")

	if resp.ShouldAutoCorrect {
		t.Fatal("suggestions_only mode should never auto-correct")
	}
	if resp.BestCandidate == nil || resp.BestCandidate.Word != "the" {
		t.Fatalf("expected suggestion candidate, got %+v", resp.BestCandidate)
	}
	if resp.Reason != "suggestions_only_mode" {
		t.Fatalf("unexpected reason: %s", resp.Reason)
	}
}
