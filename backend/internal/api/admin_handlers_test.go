package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
