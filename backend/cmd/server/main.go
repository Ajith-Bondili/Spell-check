package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Ajith-Bondili/spell-check/internal/api"
	"github.com/Ajith-Bondili/spell-check/internal/spellcheck"
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

	// Create API server
	server := api.NewServer(spellChecker, config)

	// Setup routes
	http.HandleFunc("/health", api.CORSMiddleware(server.HealthHandler))
	http.HandleFunc("/spell", api.CORSMiddleware(server.SpellHandler))
	http.HandleFunc("/rescore", api.CORSMiddleware(server.RescoreHandler))

	// Start server
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	fmt.Printf("\n✨ Server running on http://%s\n", addr)
	fmt.Println("📡 Endpoints:")
	fmt.Println("   GET  /health  - Health check")
	fmt.Println("   POST /spell   - Fast spell check (on space)")
	fmt.Println("   POST /rescore - Context-aware correction (on punctuation)")
	fmt.Println("\n💡 Press Ctrl+C to stop\n")

	// Listen and serve
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
