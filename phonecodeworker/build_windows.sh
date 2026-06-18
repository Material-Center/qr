#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="${OUT:-${ROOT_DIR}/dist/phonecodeworker-windows-amd64.exe}"
OUT_DIR="$(dirname "${OUT}")"

mkdir -p "${OUT_DIR}"
cd "${ROOT_DIR}"

GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
if [ "${GIT_COMMIT}" != "unknown" ] && [ -n "$(git status --short 2>/dev/null)" ]; then
  GIT_COMMIT="${GIT_COMMIT}-dirty"
fi
BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -buildvcs=false -trimpath \
  -ldflags="-s -w -buildid= -X main.version=phonecodeworker -X main.gitCommit=${GIT_COMMIT} -X main.buildTime=${BUILD_TIME}" \
  -o "${OUT}" .
cp "${ROOT_DIR}/run_phonecodeworker.bat" "${OUT_DIR}/run_phonecodeworker.bat"
cp "${ROOT_DIR}/pause_phonecodeworker.bat" "${OUT_DIR}/pause_phonecodeworker.bat"
cp "${ROOT_DIR}/start_phonecodeworker.bat" "${OUT_DIR}/start_phonecodeworker.bat"
echo "built ${OUT} gitCommit=${GIT_COMMIT} buildTime=${BUILD_TIME}"
echo "copied ${OUT_DIR}/run_phonecodeworker.bat"
