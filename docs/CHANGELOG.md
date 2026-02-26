# Changelog

## 0.3.0

### Added
- Per-domain profile system with APIs:
  - `GET /profiles`
  - `GET/PUT /profiles/default`
  - `GET/PUT/DELETE /profiles/domain/{domain}`
- Correction journal and undo APIs:
  - `POST /corrections/applied`
  - `POST /undo`
- Pain-point insights API:
  - `GET /insights/pain-points`
- Response metadata additions:
  - `correction_id`
  - `explanation`
  - `undo_ttl_ms`
- Undo UX in content script:
  - time-travel undo chip
  - one-click trust controls (keep word / block pair)
  - hotkey undo (`Ctrl/Cmd + Shift + Backspace`)
- Popup domain-profile controls and pain-point insight panel

### Changed
- Correction requests now support `domain`, `session_id`, and `cursor_token`
- Backend decisions now resolve by per-domain profile before fallback to default
- Skip telemetry now tracks skip reasons

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
