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
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
if [ "${GIT_COMMIT}" != "unknown" ] && [ -n "$(git status --short 2>/dev/null)" ]; then
  GIT_COMMIT="${GIT_COMMIT}-dirty"
fi
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

"${WAILS_BIN}" build \
  -clean \
  -platform windows/amd64 \
  -nopackage \
  -trimpath \
  -ldflags "-s -w -buildid= -X main.version=phone-task-client-ui -X main.gitCommit=${GIT_COMMIT} -X main.buildTime=${BUILD_TIME}" \
  -o phone-task-client-ui-windows-amd64.exe

echo "built ${ROOT_DIR}/build/bin/phone-task-client-ui-windows-amd64.exe gitCommit=${GIT_COMMIT} buildTime=${BUILD_TIME}"
