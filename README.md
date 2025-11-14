# Local Context-Aware Autocorrect System

A privacy-first, real-time autocorrect system that runs completely offline using a small local LLM.

## Architecture

### Backend (Go)
- **Fast Layer**: SymSpell-based typo correction (<50ms)
- **Smart Layer**: 1-3B parameter LLM for context-aware disambiguation
- **HTTP API**: `/spell` (instant) and `/rescore` (contextual)

### Frontend (Browser Extension)
- Monitors all text inputs across websites
- Real-time correction as you type
- Works in Gmail, Docs, Notion, Discord, ChatGPT, etc.

## How It Works

1. **On space** → Fast spell check (SymSpell)
2. **On punctuation/pause** → Context-aware LLM rescoring (if needed)
3. **Auto-correct high confidence**, suggest medium confidence
4. **Skip URLs, code blocks, passwords**

## Tech Stack

- **Backend**: Go 1.24
- **Spell Checker**: SymSpell algorithm
- **LLM**: llama.cpp with 1-3B GGUF models
- **Frontend**: TypeScript + Manifest V3
- **Privacy**: 100% offline, zero cloud calls

## Quick Start

```bash
# 1. Start the backend server
cd backend
go run ./cmd/server/main.go

# 2. Load extension in Chrome
# - Open chrome://extensions/
# - Enable Developer Mode
# - Click "Load unpacked"
# - Select the `extension` folder

# 3. Test it!
# Open extension/test.html in your browser
# Type "teh" and press SPACE → watch it become "the"!
```

📖 **Full installation guide**: [docs/INSTALLATION.md](docs/INSTALLATION.md)

## Current Status

✅ **Phase 1-6 Complete**:
- SymSpell spell checker (0.7ms dictionary load, <1ms lookups)
- HTTP API server with /spell and /rescore endpoints
- Browser extension with real-time monitoring
- **Context-aware correction** (their/there/they're, your/you're, etc.)
- **Intelligent guardrails** (protects URLs, code, emails, passwords)
- Auto-correct for high confidence (>90%)
- Suggestions for medium confidence (>50%)
- 11 test suites, 29 tests, 100% passing

🚧 **Coming Next**:
- LLM integration (Ollama) for complex cases
- Expanded confusables database
- User dictionary & custom words
- Statistics dashboard
- Performance optimization & caching

## Performance

- Dictionary load: **~700µs** (0.7ms)
- Spell check: **<1ms** per word
- Auto-correct threshold: **0.9** (90% confidence)
- Suggestion threshold: **0.5** (50% confidence)

## Project Structure

```
├── backend/              # Go server
│   ├── cmd/server/      # Main entry point
│   ├── internal/
│   │   ├── spellcheck/  # SymSpell implementation
│   │   ├── llm/         # Context analyzer & confusables
│   │   ├── guardrails/  # URL/code/email protection
│   │   ├── api/         # HTTP handlers
│   │   └── types/       # Data structures
│   └── data/            # Dictionary files
├── extension/           # Browser extension
│   ├── src/            # Content & background scripts
│   └── public/         # Popup UI & icons
└── docs/               # Documentation
```

## Documentation

- [📚 Learning Guide](docs/LEARNING_GUIDE.md) - Architecture & concepts
- [📦 Installation Guide](docs/INSTALLATION.md) - Setup instructions
- [🧪 Test Page](extension/test.html) - Try it out locally

## Contributing

This is a learning project! Contributions, suggestions, and feedback are welcome.

## License

MIT

---

Built with ❤️ for privacy and performance
