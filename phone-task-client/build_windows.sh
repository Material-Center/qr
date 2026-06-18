#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_OUT_DIR="${ROOT_DIR}/dist"
OUT="${OUT:-${DEFAULT_OUT_DIR}/phone-task-client-windows-amd64.exe}"
OUT_DIR="$(dirname "${OUT}")"

mkdir -p "${OUT_DIR}"

cd "${ROOT_DIR}"
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
if [ "${GIT_COMMIT}" != "unknown" ] && [ -n "$(git status --short 2>/dev/null)" ]; then
  GIT_COMMIT="${GIT_COMMIT}-dirty"
fi
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

GOOS=windows GOARCH=amd64 go build -buildvcs=false -trimpath \
  -ldflags="-s -w -buildid= -X main.version=phone-task-client -X main.gitCommit=${GIT_COMMIT} -X main.buildTime=${BUILD_TIME}" \
  -o "${OUT}" ./cmd/phone-task-client
cp "${ROOT_DIR}/run_phone_task_client.bat" "${OUT_DIR}/run_phone_task_client.bat"

echo "built ${OUT} gitCommit=${GIT_COMMIT} buildTime=${BUILD_TIME}"
echo "copied ${OUT_DIR}/run_phone_task_client.bat"
