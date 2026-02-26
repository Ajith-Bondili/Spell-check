package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func defaultTestSettings() Settings {
	return Settings{
		Enabled:              true,
		Mode:                 ModeConservative,
		AutoCorrectThreshold: 0.75,
		SuggestionThreshold:  0.50,
		MaxSuggestions:       5,
	}
}

func TestNewStoreInitializesFilesAndDefaults(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir, defaultTestSettings())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	settings := store.GetSettings()
	if !settings.Enabled {
		t.Fatal("expected enabled by default")
	}
	if settings.Mode != ModeConservative {
		t.Fatalf("unexpected mode: %s", settings.Mode)
	}

	files := []string{
		"settings.json",
		"profiles.json",
		"user_dictionary.json",
		"ignored.json",
		"stats.json",
		"feedback.json",
		"correction_journal.json",
	}
	for _, filename := range files {
		if _, err := os.Stat(filepath.Join(dir, filename)); err != nil {
			t.Fatalf("expected %s to exist: %v", filename, err)
		}
	}
}

func TestDomainProfilesResolve(t *testing.T) {
	store, err := NewStore(t.TempDir(), defaultTestSettings())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	profile, err := store.SetDomainProfile("my.chat.app", Settings{
		Enabled:              true,
		Mode:                 ModeSuggestions,
		AutoCorrectThreshold: 0.8,
		SuggestionThreshold:  0.35,
		MaxSuggestions:       6,
		RespectSlang:         true,
	})
	if err != nil {
		t.Fatalf("SetDomainProfile failed: %v", err)
	}
	if !profile.RespectSlang {
		t.Fatal("expected respect_slang=true")
	}

	effective, matched := store.ResolveSettings("room.my.chat.app")
	if matched != "my.chat.app" {
		t.Fatalf("expected matched domain my.chat.app, got %s", matched)
	}
	if effective.Mode != ModeSuggestions {
		t.Fatalf("expected suggestions mode, got %s", effective.Mode)
	}
}

func TestCorrectionJournalUndoAndInsights(t *testing.T) {
	store, err := NewStore(t.TempDir(), defaultTestSettings())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	record, err := store.RecordAppliedCorrection(CorrectionRecord{
		Original:    "teh",
		Suggestion:  "the",
		Domain:      "docs.google.com",
		Source:      "spell",
		Mode:        ModeConservative,
		Reason:      "auto_correct",
		Explanation: "Fixed likely typo with high confidence.",
		Confidence:  0.88,
	})
	if err != nil {
		t.Fatalf("RecordAppliedCorrection failed: %v", err)
	}
	if record.CorrectionID == "" {
		t.Fatal("expected correction_id to be set")
	}

	undone, found, err := store.UndoCorrection(record.CorrectionID)
	if err != nil {
		t.Fatalf("UndoCorrection failed: %v", err)
	}
	if !found || !undone.Undone {
		t.Fatalf("expected correction to be marked undone, got %+v", undone)
	}

	insights := store.GetPainPointInsights(5)
	if len(insights.TopUndonePairs) == 0 {
		t.Fatal("expected undone pair insight")
	}
	if insights.TopUndonePairs[0].Key != "teh|the" {
		t.Fatalf("expected top undone pair teh|the, got %s", insights.TopUndonePairs[0].Key)
	}
	if len(insights.DomainCorrectionVolume) == 0 {
		t.Fatal("expected domain correction volume insight")
	}
}

func TestAddAndRemoveCustomWord(t *testing.T) {
	store, err := NewStore(t.TempDir(), defaultTestSettings())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	entry, err := store.AddCustomWord("AjithName", 0)
	if err != nil {
		t.Fatalf("AddCustomWord failed: %v", err)
	}
	if entry.Word != "ajithname" {
		t.Fatalf("expected normalized word, got: %s", entry.Word)
	}
	if entry.Frequency <= 0 {
		t.Fatalf("expected frequency to be set, got: %d", entry.Frequency)
	}

	freq, ok := store.GetCustomWordFrequency("ajithname")
	if !ok {
		t.Fatal("expected custom word frequency to exist")
	}
	if freq != entry.Frequency {
		t.Fatalf("expected frequency %d, got %d", entry.Frequency, freq)
	}

	removed, err := store.RemoveCustomWord("ajithname")
	if err != nil {
		t.Fatalf("RemoveCustomWord failed: %v", err)
	}
	if !removed {
		t.Fatal("expected word removal to succeed")
	}
	if _, ok := store.GetCustomWordFrequency("ajithname"); ok {
		t.Fatal("word should no longer exist after removal")
	}
}

func TestIgnoreRules(t *testing.T) {
	store, err := NewStore(t.TempDir(), defaultTestSettings())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if err := store.AddIgnoredWord("MyBrandWord"); err != nil {
		t.Fatalf("AddIgnoredWord failed: %v", err)
	}
	if !store.IsWordIgnored("mybrandword") {
		t.Fatal("expected ignored word to be found")
	}

	if err := store.AddIgnoredPair("teh", "the"); err != nil {
		t.Fatalf("AddIgnoredPair failed: %v", err)
	}
	if !store.IsPairIgnored("teh", "the") {
		t.Fatal("expected ignored pair to be found")
	}
}

func TestFeedbackAdjustment(t *testing.T) {
	store, err := NewStore(t.TempDir(), defaultTestSettings())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		if err := store.RecordFeedback("teh", "the", true); err != nil {
			t.Fatalf("RecordFeedback accepted failed: %v", err)
		}
	}
	boost := store.FeedbackAdjustment("teh", "the")
	if boost <= 0 {
		t.Fatalf("expected positive boost, got: %f", boost)
	}

	for i := 0; i < 8; i++ {
		if err := store.RecordFeedback("wierd", "weird", false); err != nil {
			t.Fatalf("RecordFeedback rejected failed: %v", err)
		}
	}
	penalty := store.FeedbackAdjustment("wierd", "weird")
	if penalty >= 0 {
		t.Fatalf("expected negative penalty, got: %f", penalty)
	}
}

func TestStatsRecordingAndReset(t *testing.T) {
	store, err := NewStore(t.TempDir(), defaultTestSettings())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	_ = store.RecordSpellRequest()
	_ = store.RecordRescoreRequest()
	_ = store.RecordAutoCorrect()
	_ = store.RecordSuggestion()
	_ = store.RecordSkip()
	_ = store.RecordError()

	stats := store.GetStats()
	if stats.TotalRequests != 2 {
		t.Fatalf("expected total_requests=2, got %d", stats.TotalRequests)
	}
	if stats.AutoCorrected != 1 || stats.Suggestions != 1 || stats.Skipped != 1 || stats.Errors != 1 {
		t.Fatalf("unexpected stats snapshot: %+v", stats)
	}

	if err := store.ResetStats(); err != nil {
		t.Fatalf("ResetStats failed: %v", err)
	}
	reset := store.GetStats()
	if reset.TotalRequests != 0 || reset.AutoCorrected != 0 {
		t.Fatalf("expected stats reset, got %+v", reset)
	}
}

func TestSetSettingsValidation(t *testing.T) {
	store, err := NewStore(t.TempDir(), defaultTestSettings())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	bad := Settings{
		Enabled:              true,
		Mode:                 "not_a_mode",
		AutoCorrectThreshold: 0.75,
		SuggestionThreshold:  0.50,
		MaxSuggestions:       5,
	}
	if err := store.SetSettings(bad); err == nil {
		t.Fatal("expected invalid settings to fail")
	}

	good := Settings{
		Enabled:              false,
		Mode:                 ModeSuggestions,
		AutoCorrectThreshold: 0.9,
		SuggestionThreshold:  0.4,
		MaxSuggestions:       7,
	}
	if err := store.SetSettings(good); err != nil {
		t.Fatalf("expected settings update to succeed: %v", err)
	}
	snapshot := store.GetSettings()
	if snapshot.Enabled != good.Enabled || snapshot.Mode != good.Mode || snapshot.MaxSuggestions != good.MaxSuggestions {
		t.Fatalf("unexpected settings snapshot: %+v", snapshot)
	}
}
