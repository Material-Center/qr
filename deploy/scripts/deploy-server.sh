#!/usr/bin/env bash
set -euo pipefail

# Local build + upload server artifact to remote host.
#
# Required env:
#   REMOTE_HOST         e.g. 192.168.1.10
#   REMOTE_USER         e.g. deploy
#   REMOTE_SERVER_DIR   e.g. /opt/qr-server
#
# Optional env:
#   REMOTE_PORT=22
#   SSH_KEY=~/.ssh/id_rsa
#   REMOTE_PASSWORD=your_password
#   SERVER_DIR=<repo>/server
#   BINARY_NAME=server
#   GOOS_TARGET=linux
#   GOARCH_TARGET=amd64
#   CGO_ENABLED_TARGET=0
#   INCLUDE_CONFIG=0      # 1 to include config.yaml into package
#   REMOTE_UPDATE_SCRIPT=/opt/qr-server/remote-update-server.sh
#   SERVICE_NAME=qr-server
#   USE_SUDO=1
#   REMOTE_SERVER_POST_CMD=""  # e.g. "sudo systemctl status qr-server"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

SERVER_DIR="${SERVER_DIR:-${REPO_ROOT}/server}"

REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_USER="${REMOTE_USER:-}"
REMOTE_SERVER_DIR="${REMOTE_SERVER_DIR:-}"
REMOTE_PORT="${REMOTE_PORT:-22}"
SSH_KEY="${SSH_KEY:-}"
REMOTE_PASSWORD="${REMOTE_PASSWORD:-}"
REMOTE_SERVER_POST_CMD="${REMOTE_SERVER_POST_CMD:-}"
REMOTE_UPDATE_SCRIPT="${REMOTE_UPDATE_SCRIPT:-${REMOTE_SERVER_DIR}/remote-update-server.sh}"
SERVICE_NAME="${SERVICE_NAME:-}"
USE_SUDO="${USE_SUDO:-1}"

BINARY_NAME="${BINARY_NAME:-server}"
GOOS_TARGET="${GOOS_TARGET:-linux}"
GOARCH_TARGET="${GOARCH_TARGET:-amd64}"
CGO_ENABLED_TARGET="${CGO_ENABLED_TARGET:-0}"
INCLUDE_CONFIG="${INCLUDE_CONFIG:-0}"

if [[ -z "${REMOTE_HOST}" || -z "${REMOTE_USER}" || -z "${REMOTE_SERVER_DIR}" ]]; then
  echo "Missing required env. Need: REMOTE_HOST, REMOTE_USER, REMOTE_SERVER_DIR"
  exit 1
fi

for cmd in go tar scp ssh; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Missing required command: ${cmd}"
    exit 1
  fi
done

SSH_OPTS=(-p "${REMOTE_PORT}" -o StrictHostKeyChecking=accept-new)
SCP_OPTS=(-P "${REMOTE_PORT}" -o StrictHostKeyChecking=accept-new)
if [[ -n "${REMOTE_PASSWORD}" ]]; then
  if ! command -v "sshpass" >/dev/null 2>&1; then
    echo "Password mode requires sshpass. Please install sshpass first."
    exit 1
  fi
  export SSHPASS="${REMOTE_PASSWORD}"
elif [[ -n "${SSH_KEY}" ]]; then
  SSH_OPTS+=(-i "${SSH_KEY}")
  SCP_OPTS+=(-i "${SSH_KEY}")
fi
if [[ -n "${REMOTE_PASSWORD}" ]]; then
  SSH_CMD=(sshpass -e ssh "${SSH_OPTS[@]}")
  SCP_CMD=(sshpass -e scp "${SCP_OPTS[@]}")
else
  SSH_CMD=(ssh "${SSH_OPTS[@]}")
  SCP_CMD=(scp "${SCP_OPTS[@]}")
fi

WORK_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${WORK_DIR}"
}
trap cleanup EXIT

PKG_DIR="${WORK_DIR}/package"
mkdir -p "${PKG_DIR}"

echo "==> Building server on local machine"
pushd "${SERVER_DIR}" >/dev/null
CGO_ENABLED="${CGO_ENABLED_TARGET}" GOOS="${GOOS_TARGET}" GOARCH="${GOARCH_TARGET}" \
  go build -buildvcs=false -trimpath -ldflags="-s -w -buildid=" \
  -o "${PKG_DIR}/${BINARY_NAME}" ./main.go
popd >/dev/null

if [[ "${INCLUDE_CONFIG}" == "1" ]]; then
  cp "${SERVER_DIR}/config.yaml" "${PKG_DIR}/config.yaml"
fi

ARCHIVE_NAME="${BINARY_NAME}_$(date +%Y%m%d%H%M%S).tar.gz"
ARCHIVE_PATH="${WORK_DIR}/${ARCHIVE_NAME}"
tar -C "${PKG_DIR}" -czf "${ARCHIVE_PATH}" .

REMOTE_TMP="/tmp/${ARCHIVE_NAME}"
REMOTE_UPDATE_TMP="/tmp/remote-update-server.sh"
REMOTE_HELPER="${SCRIPT_DIR}/remote-update-server.sh"

echo "==> Uploading package to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TMP}"
"${SCP_CMD[@]}" "${ARCHIVE_PATH}" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_TMP}"

echo "==> Uploading remote update script"
"${SCP_CMD[@]}" "${REMOTE_HELPER}" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_UPDATE_TMP}"

echo "==> Installing remote update script"
"${SSH_CMD[@]}" "${REMOTE_USER}@${REMOTE_HOST}" \
  "mkdir -p '$(dirname "${REMOTE_UPDATE_SCRIPT}")' && cp -f '${REMOTE_UPDATE_TMP}' '${REMOTE_UPDATE_SCRIPT}' && chmod +x '${REMOTE_UPDATE_SCRIPT}' && rm -f '${REMOTE_UPDATE_TMP}'"

echo "==> Executing remote update script"
"${SSH_CMD[@]}" "${REMOTE_USER}@${REMOTE_HOST}" \
  "REMOTE_SERVER_DIR='${REMOTE_SERVER_DIR}' BINARY_NAME='${BINARY_NAME}' UPLOAD_ARCHIVE='${REMOTE_TMP}' SERVICE_NAME='${SERVICE_NAME}' USE_SUDO='${USE_SUDO}' REMOTE_SERVER_POST_CMD='${REMOTE_SERVER_POST_CMD}' '${REMOTE_UPDATE_SCRIPT}'"

echo "Server deploy finished."
