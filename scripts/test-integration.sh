#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="$ROOT/.env"

if [ -f "$ENV_FILE" ]; then
  set -a
  source "$ENV_FILE"
  set +a
else
  echo "[test-integration] .env not found, create one from .env.example and fill in your credentials"
  echo "[test-integration]   cp .env.example .env"
fi

if [ -z "${GITHUB_TOKEN:-}" ]; then
  echo "[test-integration] GITHUB_TOKEN is not set — tests that require a real token will be skipped"
fi

PACKAGES="${1:-./...}"

echo "[test-integration] running integration tests: $PACKAGES"
cd "$ROOT"
go test -v -tags=integration -count=1 -timeout 5m "$PACKAGES"
