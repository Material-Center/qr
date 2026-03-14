#!/usr/bin/env bash
set -euo pipefail

# Local build + upload web dist to remote server.
#
# Required env:
#   REMOTE_HOST       e.g. 192.168.1.10
#   REMOTE_USER       e.g. deploy
#   REMOTE_WEB_DIR    e.g. /var/www/qr-web
#
# Optional env:
#   REMOTE_PORT=22
#   SSH_KEY=~/.ssh/id_rsa
#   REMOTE_PASSWORD=your_password
#   WEB_DIR=<repo>/web
#   REMOTE_WEB_POST_CMD=""   # e.g. "sudo systemctl reload nginx"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

WEB_DIR="${WEB_DIR:-${REPO_ROOT}/web}"
WEB_DIST_DIR="${WEB_DIR}/dist"

REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_USER="${REMOTE_USER:-}"
REMOTE_WEB_DIR="${REMOTE_WEB_DIR:-}"
REMOTE_PORT="${REMOTE_PORT:-22}"
SSH_KEY="${SSH_KEY:-}"
REMOTE_PASSWORD="${REMOTE_PASSWORD:-}"
REMOTE_WEB_POST_CMD="${REMOTE_WEB_POST_CMD:-}"

if [[ -z "${REMOTE_HOST}" || -z "${REMOTE_USER}" || -z "${REMOTE_WEB_DIR}" ]]; then
  echo "Missing required env. Need: REMOTE_HOST, REMOTE_USER, REMOTE_WEB_DIR"
  exit 1
fi

for cmd in npm rsync ssh; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Missing required command: ${cmd}"
    exit 1
  fi
done

SSH_OPTS=(-p "${REMOTE_PORT}" -o StrictHostKeyChecking=accept-new)
if [[ -n "${REMOTE_PASSWORD}" ]]; then
  if ! command -v "sshpass" >/dev/null 2>&1; then
    echo "Password mode requires sshpass. Please install sshpass first."
    exit 1
  fi
  export SSHPASS="${REMOTE_PASSWORD}"
elif [[ -n "${SSH_KEY}" ]]; then
  SSH_OPTS+=(-i "${SSH_KEY}")
fi
if [[ -n "${REMOTE_PASSWORD}" ]]; then
  SSH_CMD=(sshpass -e ssh "${SSH_OPTS[@]}")
  RSYNC_RSH="sshpass -e ssh ${SSH_OPTS[*]}"
else
  SSH_CMD=(ssh "${SSH_OPTS[@]}")
  RSYNC_RSH="ssh ${SSH_OPTS[*]}"
fi

echo "==> Building web on local machine"
pushd "${WEB_DIR}" >/dev/null
if [[ -f "package-lock.json" ]]; then
  npm ci
else
  npm install
fi
npm run build
popd >/dev/null

if [[ ! -d "${WEB_DIST_DIR}" ]]; then
  echo "Build failed: dist directory not found: ${WEB_DIST_DIR}"
  exit 1
fi

# Ensure deploy output has no source map debug artifacts.
find "${WEB_DIST_DIR}" -type f -name "*.map" -delete || true

echo "==> Uploading dist to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_WEB_DIR}"
rsync -az --delete -e "${RSYNC_RSH}" "${WEB_DIST_DIR}/" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_WEB_DIR}/"

if [[ -n "${REMOTE_WEB_POST_CMD}" ]]; then
  echo "==> Running remote post command"
  "${SSH_CMD[@]}" "${REMOTE_USER}@${REMOTE_HOST}" "${REMOTE_WEB_POST_CMD}"
fi

echo "Web deploy finished."
