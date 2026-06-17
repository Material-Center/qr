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
"${WAILS_BIN}" build \
  -clean \
  -platform windows/amd64 \
  -nopackage \
  -trimpath \
  -ldflags "-s -w -buildid=" \
  -o phone-task-client-ui-windows-amd64.exe

echo "built ${ROOT_DIR}/build/bin/phone-task-client-ui-windows-amd64.exe"
