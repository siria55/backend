#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="$ROOT_DIR/migrations"

if ! command -v migrate >/dev/null 2>&1; then
  echo "error: migrate CLI 未安装。请先安装 https://github.com/golang-migrate/migrate/tree/master/cmd/migrate" >&2
  exit 1
fi

export DATABASE_URL=${DATABASE_URL:-postgres://postgres:w1XEPKbf24egWv8bgdJP@localhost:5432/mars?sslmode=disable}

CMD=${1:-up}
shift || true

migrate -path "$MIGRATIONS_DIR" -database "$DATABASE_URL" "$CMD" "$@"
