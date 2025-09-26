#!/usr/bin/env bash
set -eo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export APP_ENV=${APP_ENV:-development}
export HTTP_HOST=${HTTP_HOST:-0.0.0.0}
export HTTP_PORT=${HTTP_PORT:-8080}

exec go run ./cmd/server
