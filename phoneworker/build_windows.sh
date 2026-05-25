#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="${OUT:-${ROOT_DIR}/dist/phoneworker-windows-amd64.exe}"
OUT_DIR="$(dirname "${OUT}")"

mkdir -p "${OUT_DIR}"
cd "${ROOT_DIR}"

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o "${OUT}" .
cp "${ROOT_DIR}/run_phoneworker.bat" "${OUT_DIR}/run_phoneworker.bat"
echo "built ${OUT}"
echo "copied ${OUT_DIR}/run_phoneworker.bat"
