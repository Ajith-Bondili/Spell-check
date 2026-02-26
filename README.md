# Local Autocorrect (Offline, Private, Fast)

Local Autocorrect is a browser-extension + localhost backend setup that fixes typos in real time without sending your text to the cloud.

## Why this exists

Most autocorrect tools either:
- feel too aggressive and annoying, or
- require cloud processing for “smart” behavior.

This project is built to feel safer:
- local-first by default
- fast enough to run while typing
- transparent controls (undo, keep word, block replacement)

## What you get

- Fast typo correction while typing
- Context-aware suggestions for common confusables
- Per-domain behavior (docs vs chat can act differently)
- Time-travel undo chip for auto-corrections
- Friendly popup controls for tuning, custom words, and insights

## 3-minute start

1. Start backend:
```bash
cd backend
go run ./cmd/server/main.go
```

2. Load extension:
- Open `chrome://extensions`
- Enable Developer mode
- Click **Load unpacked**
- Select `extension/`

3. Test:
- Open `extension/test.html`
- Type `teh ` and similar typos

## Where to go next

- Quick setup help: [docs/INSTALLATION.md](docs/INSTALLATION.md)
- API details: [docs/API.md](docs/API.md)
- Architecture notes: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- What changed recently: [docs/CHANGELOG.md](docs/CHANGELOG.md)

## Tech note

Current implementation is algorithm-first (no external LLM runtime required).

## License

MIT
