package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ajith-Bondili/spell-check/internal/spellcheck"
	"github.com/Ajith-Bondili/spell-check/internal/storage"
	"github.com/Ajith-Bondili/spell-check/internal/types"
)

func newAPITestServer(t *testing.T) *Server {
	t.Helper()

	ss := spellcheck.NewSymSpell(2)
	ss.AddWord("the", 100000)
	ss.AddWord("hello", 50000)

	store, err := storage.NewStore(t.TempDir(), storage.Settings{
		Enabled:              true,
		Mode:                 storage.ModeConservative,
		AutoCorrectThreshold: 0.75,
		SuggestionThreshold:  0.5,
		MaxSuggestions:       5,
	})
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	cfg := &types.Config{
		StateDir: t.TempDir(),
	}
	return NewServer(ss, cfg, store)
}

func TestSettingsHandlerUpdate(t *testing.T) {
	server := newAPITestServer(t)

	body := map[string]interface{}{
		"mode":                   "aggressive",
		"auto_correct_threshold": 0.71,
		"suggestion_threshold":   0.42,
		"max_suggestions":        7,
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	server.SettingsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	settings := server.store.GetSettings()
	if settings.Mode != storage.ModeAggressive {
		t.Fatalf("expected mode=aggressive, got %s", settings.Mode)
	}
	if settings.MaxSuggestions != 7 {
		t.Fatalf("expected max_suggestions=7, got %d", settings.MaxSuggestions)
	}
}

func TestDictionaryWordLifecycle(t *testing.T) {
	server := newAPITestServer(t)

	addPayload := []byte(`{"word":"AcmeLexicon","frequency":1900000}`)
	addReq := httptest.NewRequest(http.MethodPost, "/dictionary/words", bytes.NewReader(addPayload))
	addRec := httptest.NewRecorder()
	server.DictionaryWordsHandler(addRec, addReq)

	if addRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on add, got %d: %s", addRec.Code, addRec.Body.String())
	}
	if _, ok := server.store.GetCustomWordFrequency("acmelexicon"); !ok {
		t.Fatal("expected custom word frequency to be present")
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/dictionary/words/acmelexicon", nil)
	delRec := httptest.NewRecorder()
	server.DictionaryWordsHandler(delRec, delReq)

	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d: %s", delRec.Code, delRec.Body.String())
	}
	if _, ok := server.store.GetCustomWordFrequency("acmelexicon"); ok {
		t.Fatal("word should be removed from store")
	}
}

func TestProfilesDomainLifecycle(t *testing.T) {
	server := newAPITestServer(t)

	payload := []byte(`{
		"mode": "suggestions_only",
		"auto_correct_threshold": 0.9,
		"suggestion_threshold": 0.4,
		"max_suggestions": 6,
		"respect_slang": true
	}`)
	putReq := httptest.NewRequest(http.MethodPut, "/profiles/domain/chat.openai.com", bytes.NewReader(payload))
	putRec := httptest.NewRecorder()
	server.ProfilesDomainHandler(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on put profile, got %d: %s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/profiles/domain/chat.openai.com", nil)
	getRec := httptest.NewRecorder()
	server.ProfilesDomainHandler(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on get profile, got %d: %s", getRec.Code, getRec.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/profiles/domain/chat.openai.com", nil)
	delRec := httptest.NewRecorder()
	server.ProfilesDomainHandler(delRec, delReq)
	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete profile, got %d: %s", delRec.Code, delRec.Body.String())
	}
}

func TestCorrectionAppliedUndoAndInsightsHandlers(t *testing.T) {
	server := newAPITestServer(t)

	applyPayload := []byte(`{
		"correction_id":"corr_test_1",
		"original":"teh",
		"suggestion":"the",
		"domain":"docs.google.com",
		"source":"spell",
		"mode":"conservative",
		"reason":"auto_correct",
		"explanation":"Fixed likely typo with high confidence.",
		"confidence":0.88,
		"session_id":"sess_test"
	}`)
	applyReq := httptest.NewRequest(http.MethodPost, "/corrections/applied", bytes.NewReader(applyPayload))
	applyRec := httptest.NewRecorder()
	server.CorrectionAppliedHandler(applyRec, applyReq)
	if applyRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on applied correction, got %d: %s", applyRec.Code, applyRec.Body.String())
	}

	undoReq := httptest.NewRequest(http.MethodPost, "/undo", bytes.NewReader([]byte(`{"correction_id":"corr_test_1"}`)))
	undoRec := httptest.NewRecorder()
	server.UndoHandler(undoRec, undoReq)
	if undoRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on undo, got %d: %s", undoRec.Code, undoRec.Body.String())
	}

	insightsReq := httptest.NewRequest(http.MethodGet, "/insights/pain-points", nil)
	insightsRec := httptest.NewRecorder()
	server.PainPointsHandler(insightsRec, insightsReq)
	if insightsRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on insights, got %d: %s", insightsRec.Code, insightsRec.Body.String())
	}
}

func TestPainPointsHandlerLimitQuery(t *testing.T) {
	server := newAPITestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/insights/pain-points?limit=3", nil)
	rec := httptest.NewRecorder()
	server.PainPointsHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"limit":3`) {
		t.Fatalf("expected response to include limit=3, got: %s", rec.Body.String())
	}
}
