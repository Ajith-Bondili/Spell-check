# Troubleshooting

## Backend shows offline in popup

1. Confirm backend is running:
```bash
curl http://127.0.0.1:8080/health
```
2. If curl fails, restart backend from `backend/`.
3. Reload extension in `chrome://extensions`.

## Autocorrect not triggering

- Check popup toggle: **Enable Autocorrect**
- Test in regular text inputs (not password fields)
- Try `extension/test.html` first to isolate website-specific issues

## Too many unwanted corrections

- Switch mode to `conservative` or `suggestions_only`
- Increase auto-correct threshold
- Use:
  - **Always Keep Word**
  - **Never Replace Pair**
  - custom domain profile for that site

## Undo chip does not appear

- Undo chip only appears for auto-corrections, not suggestions
- Lower threshold or use aggressive mode to force more auto-corrects for testing

## Domain profile looks ignored

- Make sure active tab URL is a normal web page (not browser internal pages)
- Save profile, then type again on the same domain
- Reset profile and re-save if needed

## State looks stale

- Use **Reload state** from popup
- Restart backend to reload cleanly
