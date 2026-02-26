#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"

echo "== Local Autocorrect Smoke Test =="
echo "Base URL: ${BASE_URL}"

check() {
  local name="$1"
  local method="$2"
  local path="$3"
  local body="${4:-}"

  echo "-- ${name}"
  if [[ -n "${body}" ]]; then
    curl -sS -X "${method}" \
      -H "Content-Type: application/json" \
      -d "${body}" \
      "${BASE_URL}${path}" >/dev/null
  else
    curl -sS -X "${method}" "${BASE_URL}${path}" >/dev/null
  fi
  echo "   ok"
}

check "health" GET "/health"
check "spell typo path" POST "/spell" '{"text":"teh","context":"this is teh ","domain":"example.com","session_id":"smoke"}'
check "rescore context path" POST "/rescore" '{"text":"their","context":"i went to their house","domain":"example.com","session_id":"smoke"}'
check "settings read path" GET "/settings"
check "insights read path" GET "/insights/pain-points"

echo "Smoke test passed."
