# Roadmap

## Completed in v0.2

- JSON-backed persistent runtime state
- Settings API and decision modes
- User dictionary add/remove flow
- Ignore word/pair rules
- Stats and feedback endpoints
- Extension popup controls wired to backend

## Next Priorities

1. Better phrase-level correction
- stuck-words (`havea` -> `have a`)
- repeated-word cleanup (`the the`)

2. Stronger context scoring
- richer pattern groups
- weighted neighboring-token windows
- per-domain correction profiles

3. UX improvements
- in-page inline suggestion chips
- undo after auto-correct
- keyboard shortcut customization

4. Performance and reliability
- request coalescing/debounce tuning
- candidate cache for hot typo paths
- race-detector and integration test suite

5. Packaging
- signed extension packaging
- optional tray app to auto-start backend
