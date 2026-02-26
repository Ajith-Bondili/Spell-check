# Local Autocorrect (Offline, Context-Aware)

A privacy-first autocorrect system that runs fully on localhost.

No cloud calls. No remote model dependency. Fast typo correction + rule-based context disambiguation.

## v0.3 Highlight Features

- Time-travel undo chip for auto-corrections (`Undo`, `Always Keep Word`, `Never Replace Pair`)
- Undo hotkey: `Ctrl/Cmd + Shift + Backspace`
- Per-domain profiles (different behavior for docs/chat/mail)
- Correction journal + explicit undo API
- Pain-point insights API (undone pairs, skip reasons, per-domain volume)

## What’s Implemented

### Backend (Go)
- SymSpell fast correction engine (`/spell`)
- Context-aware rescoring for confusables (`/rescore`)
- Guardrails for URLs, code-like text, emails, numbers, hashtags, mentions
- Runtime settings API (`/settings`)
- Profile APIs (`/profiles`, `/profiles/default`, `/profiles/domain/{domain}`)
- User dictionary + ignore rules API (`/dictionary`, `/dictionary/words`, `/dictionary/ignore`)
- Stats + feedback APIs (`/stats`, `/stats/reset`, `/feedback`)
- Applied-correction + undo APIs (`/corrections/applied`, `/undo`)
- Insight API (`/insights/pain-points`)
- Persistent JSON state store (`backend/data/state`)

### Extension (Manifest V3)
- Real-time correction on text fields + contenteditable elements
- Background service worker as API control plane
- Live settings sync from popup
- Domain profile editor in popup
- Custom-word management from popup
- Ignore rules from popup
- Live stats + reset + backend reload controls from popup
- Undo chip and trust controls in-page (keep word / block pair)

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
- `GET /profiles`
- `GET /profiles/default`
- `PUT /profiles/default`
- `GET /profiles/domain/{domain}`
- `PUT /profiles/domain/{domain}`
- `DELETE /profiles/domain/{domain}`
- `POST /corrections/applied`
- `POST /undo`
- `GET /insights/pain-points`
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
