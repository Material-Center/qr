#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT="${OUT:-"${ROOT_DIR}/dist/miserver-windows-amd64.exe"}"

mkdir -p "$(dirname "${OUT}")"

cd "${ROOT_DIR}"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o "${OUT}" .

echo "Windows binary built: ${OUT}"
