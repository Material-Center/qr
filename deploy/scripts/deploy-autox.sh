#!/usr/bin/env bash
set -euo pipefail

# 本地构建 autox + rsync 上传 dist（排除 main.js、project.json）。
# 不使用 --delete，避免远端被排除的文件被误删。
#
# Required env（由根目录 deploy.sh 注入）:
#   REMOTE_HOST, REMOTE_USER, REMOTE_AUTOX_DIR
#
# Optional:
#   REMOTE_PORT, SSH_KEY, REMOTE_PASSWORD
#   AUTOX_DIR=<repo>/autox
#   REMOTE_AUTOX_POST_CMD
#   AUTOX_RSYNC_DELETE=1  启用 rsync --delete（慎用）

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

AUTOX_DIR="${AUTOX_DIR:-${REPO_ROOT}/autox}"
AUTOX_DIST_DIR="${AUTOX_DIR}/dist"

REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_USER="${REMOTE_USER:-}"
REMOTE_AUTOX_DIR="${REMOTE_AUTOX_DIR:-}"
REMOTE_PORT="${REMOTE_PORT:-22}"
SSH_KEY="${SSH_KEY:-}"
REMOTE_PASSWORD="${REMOTE_PASSWORD:-}"
REMOTE_AUTOX_POST_CMD="${REMOTE_AUTOX_POST_CMD:-}"

if [[ -z "${REMOTE_HOST}" || -z "${REMOTE_USER}" || -z "${REMOTE_AUTOX_DIR}" ]]; then
  echo "Missing required env. Need: REMOTE_HOST, REMOTE_USER, REMOTE_AUTOX_DIR"
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

echo "==> Building autox on local machine"
pushd "${AUTOX_DIR}" >/dev/null
if [[ -f "package-lock.json" ]]; then
  npm ci
else
  npm install
fi
npm run build
popd >/dev/null

if [[ ! -d "${AUTOX_DIST_DIR}" ]]; then
  echo "Build failed: dist directory not found: ${AUTOX_DIST_DIR}"
  exit 1
fi

RSYNC=(rsync -avz -e "${RSYNC_RSH}")
RSYNC+=(--exclude 'update.js' --exclude 'project.json')
if [[ "${AUTOX_RSYNC_DELETE:-}" == "1" ]]; then
  RSYNC+=(--delete)
fi
RSYNC+=("${AUTOX_DIST_DIR}/" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_AUTOX_DIR%/}/")

echo "==> Uploading autox dist to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_AUTOX_DIR}"
"${RSYNC[@]}"

if [[ -n "${REMOTE_AUTOX_POST_CMD}" ]]; then
  echo "==> Running remote post command"
  "${SSH_CMD[@]}" "${REMOTE_USER}@${REMOTE_HOST}" "${REMOTE_AUTOX_POST_CMD}"
fi

echo "Autox deploy finished."
