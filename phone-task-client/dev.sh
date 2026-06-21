#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WAILS_BIN="${WAILS_BIN:-wails}"

if ! command -v "${WAILS_BIN}" >/dev/null 2>&1; then
  if [[ -x "${HOME}/go/bin/wails" ]]; then
    WAILS_BIN="${HOME}/go/bin/wails"
  else
    echo "wails not found; install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest" >&2
    exit 1
  fi
fi

cd "${ROOT_DIR}"
export GOCACHE="${GOCACHE:-/private/tmp/qr-go-build-cache}"
export PHONE_TASK_CLIENT_DEV_DIR="${PHONE_TASK_CLIENT_DEV_DIR:-${ROOT_DIR}/.dev}"

echo "starting Wails dev server..."
echo "root: ${ROOT_DIR}"
echo "wails: ${WAILS_BIN}"
echo "gocache: ${GOCACHE}"
echo "dev data: ${PHONE_TASK_CLIENT_DEV_DIR}/data"
echo "dev logs: ${PHONE_TASK_CLIENT_DEV_DIR}/logs"

exec "${WAILS_BIN}" dev "$@"
