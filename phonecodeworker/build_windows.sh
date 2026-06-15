#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="${OUT:-${ROOT_DIR}/dist/phonecodeworker-windows-amd64.exe}"
OUT_DIR="$(dirname "${OUT}")"

mkdir -p "${OUT_DIR}"
cd "${ROOT_DIR}"

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
  go build -buildvcs=false -trimpath -ldflags="-s -w -buildid=" -o "${OUT}" .
cp "${ROOT_DIR}/run_phonecodeworker.bat" "${OUT_DIR}/run_phonecodeworker.bat"
cp "${ROOT_DIR}/pause_phonecodeworker.bat" "${OUT_DIR}/pause_phonecodeworker.bat"
cp "${ROOT_DIR}/start_phonecodeworker.bat" "${OUT_DIR}/start_phonecodeworker.bat"
echo "built ${OUT}"
echo "copied ${OUT_DIR}/run_phonecodeworker.bat"
