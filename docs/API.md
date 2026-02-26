# API Reference (v0.3)

Base URL: `http://127.0.0.1:8080`

## Health

### `GET /health`
Returns backend status/version/default-mode metadata.

## Correction Endpoints

### `POST /spell`
Fast typo correction.

Request:
```json
{
  "text": "teh",
  "context": "this is teh ",
  "domain": "docs.google.com",
  "session_id": "sess_123",
  "cursor_token": "INPUT:14"
}
```

Response (auto-correct example):
```json
{
  "original": "teh",
  "candidates": [{ "word": "the", "confidence": 0.81, "edit_distance": 1, "frequency": 23135851162 }],
  "best_candidate": { "word": "the", "confidence": 0.81, "edit_distance": 1, "frequency": 23135851162 },
  "should_auto_correct": true,
  "processing_time_ms": 1,
  "source": "spell",
  "reason": "auto_correct",
  "decision_mode": "conservative",
  "explanation": "Fixed likely typo with high confidence.",
  "correction_id": "corr_...",
  "undo_ttl_ms": 6000,
  "skipped": false
}
```

### `POST /rescore`
Context-aware correction for confusables.

Request:
```json
{
  "text": "there",
  "context": "i went to there house",
  "domain": "mail.google.com",
  "session_id": "sess_123"
}
```

## Settings

### `GET /settings`
Returns default profile settings.

### `PUT /settings`
Updates default profile settings.

Request:
```json
{
  "enabled": true,
  "mode": "aggressive",
  "auto_correct_threshold": 0.7,
  "suggestion_threshold": 0.45,
  "max_suggestions": 7,
  "respect_slang": false
}
```

## Profiles

### `GET /profiles`
Returns:
- default profile
- all explicit domain profiles

### `GET /profiles/default`
Returns default profile.

### `PUT /profiles/default`
Updates default profile (same shape as `PUT /settings`).

### `GET /profiles/domain/{domain}`
Returns domain-specific profile if it exists.

### `PUT /profiles/domain/{domain}`
Creates/updates domain-specific profile.

### `DELETE /profiles/domain/{domain}`
Removes domain-specific profile.

## Dictionary and Ignore Rules

### `GET /dictionary`
Returns custom words, ignored words, ignored pairs.

### `POST /dictionary/words`
Add/update custom word.

```json
{
  "word": "notionworkspace",
  "frequency": 1800000
}
```

### `DELETE /dictionary/words/{word}`
Remove custom word.

### `POST /dictionary/ignore`
Add ignore rule.

Ignore word:
```json
{ "word": "mybrandword" }
```

Ignore pair:
```json
{ "original": "teh", "suggestion": "the" }
```

## Stats and Feedback

### `GET /stats`
Returns runtime counters and dictionary/ignore counts.

### `POST /stats/reset`
Resets runtime counters.

### `POST /feedback`
Stores accepted/rejected correction feedback.

```json
{
  "original": "teh",
  "suggestion": "the",
  "accepted": true
}
```

## Correction Journal + Undo

### `POST /corrections/applied`
Records a correction the client actually applied.

```json
{
  "correction_id": "corr_123",
  "original": "teh",
  "suggestion": "the",
  "domain": "docs.google.com",
  "source": "spell",
  "mode": "conservative",
  "reason": "auto_correct",
  "explanation": "Fixed likely typo with high confidence.",
  "confidence": 0.88,
  "session_id": "sess_123",
  "before_text": "this is teh ",
  "after_text": "this is the "
}
```

### `POST /undo`
Marks a correction as undone in backend journal.

```json
{
  "correction_id": "corr_123"
}
```

## Insights

### `GET /insights/pain-points`
Returns:
- top undone correction pairs
- top ignored pairs
- top ignored words
- per-domain correction volume
- top skip reasons

## Reload State

### `POST /reload`
Reloads JSON state files from disk and re-syncs custom dictionary.
