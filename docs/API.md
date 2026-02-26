# API Reference

Base URL: `http://127.0.0.1:8080`

## Health

### `GET /health`

Returns backend status/version/mode snapshot.

## Correction Endpoints

### `POST /spell`

Fast typo correction.

Request:
```json
{
  "text": "teh",
  "context": "this is teh "
}
```

Response:
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
  "skipped": false
}
```

### `POST /rescore`

Context-aware correction for confusables.

Request:
```json
{
  "text": "there",
  "context": "i went to there house"
}
```

## Settings

### `GET /settings`
Returns runtime settings.

### `PUT /settings`
Partial update.

Request example:
```json
{
  "enabled": true,
  "mode": "aggressive",
  "auto_correct_threshold": 0.7,
  "suggestion_threshold": 0.45,
  "max_suggestions": 7
}
```

## Dictionary and Ignore Rules

### `GET /dictionary`
Returns:
- custom words
- ignored words
- ignored pairs

### `POST /dictionary/words`
Add/update custom word.

Request:
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
Store accepted/rejected feedback for pair ranking adjustments.

Request:
```json
{
  "original": "teh",
  "suggestion": "the",
  "accepted": true
}
```

## Reload State

### `POST /reload`
Reload JSON state files from disk and sync custom dictionary into SymSpell.
