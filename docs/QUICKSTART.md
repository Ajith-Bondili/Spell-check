# Quickstart (First 10 Minutes)

This guide is for getting the system running fast, not learning every internal detail.

## 1) Start backend

```bash
cd backend
go run ./cmd/server/main.go
```

Keep this terminal open.

## 2) Load extension

1. Open `chrome://extensions`
2. Enable **Developer mode**
3. Click **Load unpacked**
4. Select `extension/`

## 3) Verify health

- Click the extension icon
- It should show backend online

If it does not:
- run `curl http://127.0.0.1:8080/health`
- confirm the backend terminal has no startup errors

## 4) Try real typing

Open `extension/test.html` and try:
- `teh ` -> likely auto-correct
- context sentence with confusables

## 5) Try controls users care about

- Undo chip after an auto-correct
- Add a custom word
- Block a correction pair
- Set a domain profile for current site
