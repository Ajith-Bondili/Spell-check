package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Ajith-Bondili/spell-check/internal/api"
	"github.com/Ajith-Bondili/spell-check/internal/spellcheck"
	"github.com/Ajith-Bondili/spell-check/internal/storage"
	"github.com/Ajith-Bondili/spell-check/internal/types"
)

func main() {
	fmt.Println("🚀 Starting Local Autocorrect Server...")

	// Load configuration
	config := types.DefaultConfig()
	fmt.Printf("📝 Configuration loaded (port: %d)\n", config.Port)

	// Initialize spell checker
	fmt.Println("📚 Loading spell checker...")
	spellChecker := spellcheck.NewSymSpell(config.MaxEditDistance)

	// Load dictionary
	startTime := time.Now()
	err := spellChecker.LoadDictionary(config.DictionaryPath)
	if err != nil {
		log.Fatalf("❌ Failed to load dictionary: %v", err)
	}
	fmt.Printf("✅ Dictionary loaded in %v\n", time.Since(startTime))

	// Initialize persistent runtime state
	defaultSettings := storage.Settings{
		Enabled:              true,
		Mode:                 config.DefaultMode,
		AutoCorrectThreshold: config.AutoCorrectThreshold,
		SuggestionThreshold:  config.SuggestionThreshold,
		MaxSuggestions:       config.MaxSuggestions,
	}
	store, err := storage.NewStore(config.StateDir, defaultSettings)
	if err != nil {
		log.Fatalf("❌ Failed to initialize state store: %v", err)
	}
	fmt.Printf("✅ State store initialized (%s)\n", config.StateDir)

	// Create API server
	server := api.NewServer(spellChecker, config, store)

	// Setup routes
	http.HandleFunc("/health", api.CORSMiddleware(server.HealthHandler))
	http.HandleFunc("/spell", api.CORSMiddleware(server.SpellHandler))
	http.HandleFunc("/rescore", api.CORSMiddleware(server.RescoreHandler))
	http.HandleFunc("/settings", api.CORSMiddleware(server.SettingsHandler))
	http.HandleFunc("/dictionary", api.CORSMiddleware(server.DictionaryHandler))
	http.HandleFunc("/dictionary/words", api.CORSMiddleware(server.DictionaryWordsHandler))
	http.HandleFunc("/dictionary/words/", api.CORSMiddleware(server.DictionaryWordsHandler))
	http.HandleFunc("/dictionary/ignore", api.CORSMiddleware(server.DictionaryIgnoreHandler))
	http.HandleFunc("/stats", api.CORSMiddleware(server.StatsHandler))
	http.HandleFunc("/stats/reset", api.CORSMiddleware(server.StatsResetHandler))
	http.HandleFunc("/feedback", api.CORSMiddleware(server.FeedbackHandler))
	http.HandleFunc("/profiles", api.CORSMiddleware(server.ProfilesHandler))
	http.HandleFunc("/profiles/default", api.CORSMiddleware(server.ProfilesDefaultHandler))
	http.HandleFunc("/profiles/domain/", api.CORSMiddleware(server.ProfilesDomainHandler))
	http.HandleFunc("/corrections/applied", api.CORSMiddleware(server.CorrectionAppliedHandler))
	http.HandleFunc("/undo", api.CORSMiddleware(server.UndoHandler))
	http.HandleFunc("/insights/pain-points", api.CORSMiddleware(server.PainPointsHandler))
	http.HandleFunc("/reload", api.CORSMiddleware(server.ReloadHandler))

	// Start server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	fmt.Printf("\n✨ Server running on http://%s\n", addr)
	fmt.Println("📡 Endpoints:")
	fmt.Println("   GET  /health  - Health check")
	fmt.Println("   POST /spell   - Fast spell check (on space)")
	fmt.Println("   POST /rescore - Context-aware correction (on punctuation)")
	fmt.Println("   GET  /settings / PUT /settings")
	fmt.Println("   GET  /dictionary / POST /dictionary/words / DELETE /dictionary/words/{word}")
	fmt.Println("   POST /dictionary/ignore / GET /stats / POST /stats/reset")
	fmt.Println("   POST /feedback / POST /reload")
	fmt.Println("   GET /profiles / GET+PUT /profiles/default / GET+PUT+DELETE /profiles/domain/{domain}")
	fmt.Println("   POST /corrections/applied / POST /undo / GET /insights/pain-points")
	fmt.Println("")
	fmt.Println("💡 Press Ctrl+C to stop")
	fmt.Println("")

	// Listen and serve
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
