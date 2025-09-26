#!/usr/bin/env bash
set -eo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

export APP_ENV=${APP_ENV:-development}
export HTTP_HOST=${HTTP_HOST:-0.0.0.0}
export HTTP_PORT=${HTTP_PORT:-8080}
export DATABASE_URL=${DATABASE_URL:-postgres://postgres:z13547842355@localhost:5432/mars?sslmode=disable}

exec go run ./cmd/server
