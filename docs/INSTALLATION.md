# Installation Guide

## Prerequisites

- Go 1.20+ installed
- Chrome or Arc browser
- Basic command line knowledge

## Step 1: Start the Backend Server

The backend server must be running for the extension to work.

```bash
# Navigate to the project root
cd /path/to/Spell-check

# Navigate to backend
cd backend

# Run the server
go run ./cmd/server/main.go
```

You should see:
```
🚀 Starting Local Autocorrect Server...
📝 Configuration loaded (port: 8080)
📚 Loading spell checker...
✅ Dictionary loaded in 716.826µs

✨ Server running on http://127.0.0.1:8080
```

**Keep this terminal window open!** The server needs to stay running.

### Alternative: Build and Run

For production use, build a binary:

```bash
# Build the server
go build -o autocorrect-server ./cmd/server

# Run the binary
./autocorrect-server
```

## Step 2: Load the Browser Extension

### For Chrome/Arc:

1. Open Chrome and navigate to `chrome://extensions/`
2. Enable **Developer mode** (toggle in top right)
3. Click **Load unpacked**
4. Select the `extension` folder from this project
5. The extension should now appear in your extensions list

### Verify Installation:

1. Click the extension icon in your toolbar
2. You should see "Backend running (v0.1.0)" status
3. If it shows "Backend offline", make sure step 1 is complete

## Step 3: Test It!

### Quick Test:

1. Open `extension/test.html` in your browser
2. Type a typo like "teh" and press SPACE
3. Watch it auto-correct to "the"!

### Real-World Test:

1. Go to Gmail, Google Docs, or any website
2. Start typing in a text field
3. Make intentional typos:
   - "seperate" → "separate"
   - "definately" → "definitely"
   - "occured" → "occurred"

## Troubleshooting

### "Backend offline" error

**Problem**: Extension can't reach the backend server

**Solution**:
1. Make sure the Go server is running (Step 1)
2. Check that it's listening on port 8080
3. Test manually: `curl http://127.0.0.1:8080/health`
4. Check browser console for CORS errors

### Extension not loading

**Problem**: Chrome rejects the extension

**Solution**:
1. Make sure you selected the `extension` folder, not the root
2. Check for errors in `chrome://extensions/`
3. Verify all files exist (manifest.json, src/, public/)

### Corrections not happening

**Problem**: Typing but nothing gets corrected

**Solution**:
1. Open browser DevTools (F12)
2. Look for console messages starting with "🎯 Local Autocorrect"
3. Check Network tab for failed requests to localhost:8080
4. Verify the extension is enabled in chrome://extensions

### Permission errors

**Problem**: Extension can't access certain sites

**Solution**:
1. Some sites (like chrome:// pages) can't run extensions
2. Try a regular website like gmail.com or docs.google.com
3. Check if the site blocks content scripts

## Advanced Configuration

### Custom Dictionary

To use your own dictionary:

1. Create a frequency dictionary file:
   ```
   word1 frequency1
   word2 frequency2
   ```

2. Update `backend/internal/types/config.go`:
   ```go
   DictionaryPath: "data/my_custom_dictionary.txt",
   ```

3. Restart the backend server

### Change Server Port

Edit `backend/internal/types/config.go`:

```go
Port: 9090,  // Change from 8080
```

Also update `extension/src/content.js`:

```javascript
const CONFIG = {
    backendUrl: 'http://127.0.0.1:9090',  // Match your port
    // ...
};
```

### Disable Auto-Correct (Suggestions Only)

In `backend/internal/types/config.go`:

```go
AutoCorrectThreshold: 1.1,  // Never auto-correct (max is 1.0)
SuggestionThreshold:  0.5,  // Show suggestions only
```

## Development Mode

### Hot Reload Backend

Install `air` for hot reloading:

```bash
go install github.com/cosmtrek/air@latest
cd backend
air
```

### Debug Extension

1. Open DevTools on any page (F12)
2. Look for messages starting with 🎯 or other emoji
3. Use `console.log()` to add your own debugging

### Run Tests

```bash
cd backend
go test ./...
```

## Next Steps

- See `docs/LEARNING_GUIDE.md` for architecture details
- See `README.md` for project overview
- Read `docs/ROADMAP.md` for future features

## Getting Help

If you're stuck:

1. Check the console logs (both backend and browser)
2. Test the backend directly: `curl -X POST http://127.0.0.1:8080/spell -H "Content-Type: application/json" -d '{"text":"teh"}'`
3. Verify the extension loaded: chrome://extensions
4. Open an issue with:
   - Error messages
   - Browser version
   - Go version
   - What you expected vs what happened
