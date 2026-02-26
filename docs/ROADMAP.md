# Roadmap

## Completed in v0.3

- JSON-backed persistent runtime state
- Settings API and decision modes
- User dictionary add/remove flow
- Ignore word/pair rules
- Stats and feedback endpoints
- Extension popup controls wired to backend
- Per-domain profile overrides
- Undo journal and undo endpoint
- In-page undo chip + trust controls
- Pain-point insights endpoint and popup view

## Next Priorities

1. Better phrase-level correction
- stuck-words (`havea` -> `have a`)
- repeated-word cleanup (`the the`)

2. Stronger context scoring
- richer pattern groups
- weighted neighboring-token windows
- phrase-level context windows

3. UX improvements
- richer inline suggestion chips
- keyboard shortcut customization
- domain profile templates

4. Performance and reliability
- request coalescing/debounce tuning
- candidate cache for hot typo paths
- race-detector and integration test suite

5. Packaging
- signed extension packaging
- optional tray app to auto-start backend
