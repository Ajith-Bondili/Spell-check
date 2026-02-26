# Installation Guide

## Prerequisites

- Go 1.24+
- Chrome/Arc/Chromium browser

## 1) Start Backend

```bash
cd backend
go run ./cmd/server/main.go
```

You should see:
- dictionary loaded
- state store initialized (`data/state`)
- server listening on `http://127.0.0.1:8080`

## 2) Load Extension

1. Open `chrome://extensions/`
2. Enable **Developer mode**
3. Click **Load unpacked**
4. Select the repo’s `extension/` folder

## 3) Verify Health

- Click extension icon
- Popup should show backend online
- If offline, confirm server is running and `127.0.0.1:8080` is reachable

## 4) Try Autocorrect

1. Open `extension/test.html`
2. Type typo + space (example: `teh `)
3. Try context corrections with punctuation

## 5) Use New Controls

From popup:
- toggle enable/disable
- switch mode (`conservative`, `aggressive`, `suggestions_only`)
- adjust thresholds
- configure per-domain profile overrides
- add/remove custom words
- add ignore word or ignore pair
- inspect/reset stats
- inspect pain-point insights
- reload backend state

In-page while typing:
- auto-correct undo chip appears after auto-fixes
- quick actions: Undo, Always Keep Word, Never Replace Pair
- keyboard undo: `Ctrl/Cmd + Shift + Backspace`

## Troubleshooting

### Backend offline in popup

- Ensure backend is running
- Check `curl http://127.0.0.1:8080/health`
- Reload extension once

### Setting updates fail

- Confirm backend health first
- Validate threshold values:
  - `0 <= suggestion_threshold <= auto_correct_threshold <= 1` (except suggestions-only mode flexibility)
- Keep `max_suggestions` between 1 and 20

### Extension appears loaded but no corrections

- Check popup `Enable Autocorrect`
- Make sure target field is text/textarea/contenteditable (not password field)
- Inspect page console for content script logs

## Build/Test

```bash
cd backend
go test ./...
```
