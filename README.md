# Local Autocorrect (Offline, Context-Aware)

A privacy-first autocorrect system that runs fully on localhost.

No cloud calls. No remote model dependency. Fast typo correction + rule-based context disambiguation.

## What’s Implemented

### Backend (Go)
- SymSpell fast correction engine (`/spell`)
- Context-aware rescoring for confusables (`/rescore`)
- Guardrails for URLs, code-like text, emails, numbers, hashtags, mentions
- Runtime settings API (`/settings`)
- User dictionary + ignore rules API (`/dictionary`, `/dictionary/words`, `/dictionary/ignore`)
- Stats + feedback APIs (`/stats`, `/stats/reset`, `/feedback`)
- Persistent JSON state store (`backend/data/state`)

### Extension (Manifest V3)
- Real-time correction on text fields + contenteditable elements
- Background service worker as API control plane
- Live settings sync from popup
- Custom-word management from popup
- Ignore rules from popup
- Live stats + reset + backend reload controls from popup

## Quick Start

```bash
# 1) Start backend
cd backend
go run ./cmd/server/main.go

# 2) Load extension
# chrome://extensions -> Developer mode -> Load unpacked -> extension/

# 3) Test
# Open extension/test.html and type typos like "teh "
```

## API Overview

Core:
- `GET /health`
- `POST /spell`
- `POST /rescore`

Runtime control:
- `GET /settings`
- `PUT /settings`
- `GET /dictionary`
- `POST /dictionary/words`
- `DELETE /dictionary/words/{word}`
- `POST /dictionary/ignore`
- `GET /stats`
- `POST /stats/reset`
- `POST /feedback`
- `POST /reload`

See full examples in [docs/API.md](docs/API.md).

## Current Decision Modes

- `conservative` (default): safer auto-correct behavior
- `aggressive`: lower auto-correct threshold for speed
- `suggestions_only`: never auto-apply replacements

## Project Structure

```text
Spell-check/
├── backend/
│   ├── cmd/server/
│   ├── internal/
│   │   ├── api/
│   │   ├── guardrails/
│   │   ├── llm/          # rule-based context logic (no external LLM dependency)
│   │   ├── spellcheck/
│   │   ├── storage/      # JSON-backed runtime state
│   │   └── types/
│   └── data/
├── extension/
│   ├── src/
│   ├── public/
│   └── test.html
└── docs/
```

## Testing

```bash
cd backend
go test ./...
```

## Docs

- [Installation](docs/INSTALLATION.md)
- [Architecture](docs/ARCHITECTURE.md)
- [API Reference](docs/API.md)
- [Roadmap](docs/ROADMAP.md)
- [Changelog](docs/CHANGELOG.md)

## License

MIT
