# Changelog

## 0.2.0

### Added
- JSON-backed store for settings, custom dictionary, ignore rules, stats, and feedback
- New APIs:
  - `GET/PUT /settings`
  - `GET /dictionary`
  - `POST /dictionary/words`
  - `DELETE /dictionary/words/{word}`
  - `POST /dictionary/ignore`
  - `GET /stats`
  - `POST /stats/reset`
  - `POST /feedback`
  - `POST /reload`
- Decision modes:
  - `conservative`
  - `aggressive`
  - `suggestions_only`
- Feedback-based confidence adjustments
- Custom-word synchronization into SymSpell
- Extension popup controls for settings/dictionary/ignore/stats
- Storage and decision unit tests

### Changed
- Correction responses now include decision metadata (`source`, `reason`, `decision_mode`, `skipped`)
- Background script now acts as control plane for backend API calls
- Content script now requests corrections via background and reports feedback

### Fixed
- Context matching false positives from substring checks in flexible pattern matching
- Baseline backend build/test issues
