#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_OUT_DIR="${ROOT_DIR}/dist"
OUT="${OUT:-${DEFAULT_OUT_DIR}/phone-task-client-windows-amd64.exe}"
OUT_DIR="$(dirname "${OUT}")"

mkdir -p "${OUT_DIR}"

cd "${ROOT_DIR}"
GOOS=windows GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags="-s -w -buildid=" -o "${OUT}" ./cmd/phone-task-client
cp "${ROOT_DIR}/run_phone_task_client.bat" "${OUT_DIR}/run_phone_task_client.bat"

echo "built ${OUT}"
echo "copied ${OUT_DIR}/run_phone_task_client.bat"
