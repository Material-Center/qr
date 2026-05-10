#!/usr/bin/env bash
set -euo pipefail

# Build extra jar locally, upload it, and install/restart the systemd service
# on the remote Ubuntu host.
#
# Required env:
#   REMOTE_HOST
#   REMOTE_USER
#
# Optional env:
#   REMOTE_PORT=22
#   SSH_KEY=~/.ssh/id_rsa
#   REMOTE_PASSWORD=your_password
#   QQ_CACHE_EXTRACTOR_DIR=<repo>/qq-cache-extractor-service
#   REMOTE_QQ_CACHE_EXTRACTOR_DIR=/opt/extra
#   QQ_CACHE_EXTRACTOR_SERVICE_NAME=extra
#   QQ_CACHE_EXTRACTOR_PORT=19091
#   QQ_CACHE_EXTRACTOR_RUN_USER=root
#   QQ_CACHE_EXTRACTOR_RUN_GROUP=root
#   QQ_CACHE_EXTRACTOR_INSTALL_JAVA=1
#   USE_SUDO=1
#   REMOTE_QQ_CACHE_EXTRACTOR_POST_CMD=""

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

QQ_CACHE_EXTRACTOR_DIR="${QQ_CACHE_EXTRACTOR_DIR:-${REPO_ROOT}/qq-cache-extractor-service}"
REMOTE_HOST="${REMOTE_HOST:-}"
REMOTE_USER="${REMOTE_USER:-}"
REMOTE_PORT="${REMOTE_PORT:-22}"
SSH_KEY="${SSH_KEY:-}"
REMOTE_PASSWORD="${REMOTE_PASSWORD:-}"
REMOTE_QQ_CACHE_EXTRACTOR_DIR="${REMOTE_QQ_CACHE_EXTRACTOR_DIR:-/opt/extra}"
QQ_CACHE_EXTRACTOR_SERVICE_NAME="${QQ_CACHE_EXTRACTOR_SERVICE_NAME:-extra}"
QQ_CACHE_EXTRACTOR_PORT="${QQ_CACHE_EXTRACTOR_PORT:-19091}"
QQ_CACHE_EXTRACTOR_RUN_USER="${QQ_CACHE_EXTRACTOR_RUN_USER:-root}"
QQ_CACHE_EXTRACTOR_RUN_GROUP="${QQ_CACHE_EXTRACTOR_RUN_GROUP:-root}"
QQ_CACHE_EXTRACTOR_INSTALL_JAVA="${QQ_CACHE_EXTRACTOR_INSTALL_JAVA:-1}"
USE_SUDO="${USE_SUDO:-1}"
REMOTE_QQ_CACHE_EXTRACTOR_POST_CMD="${REMOTE_QQ_CACHE_EXTRACTOR_POST_CMD:-}"

if [[ -z "${REMOTE_HOST}" || -z "${REMOTE_USER}" ]]; then
  echo "Missing required env. Need: REMOTE_HOST, REMOTE_USER"
  exit 1
fi

for cmd in scp ssh; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Missing required command: ${cmd}"
    exit 1
  fi
done

if [[ ! -x "${QQ_CACHE_EXTRACTOR_DIR}/gradlew" ]]; then
  echo "Missing gradlew: ${QQ_CACHE_EXTRACTOR_DIR}/gradlew"
  exit 1
fi

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

echo "==> Building extra on local machine"
pushd "${QQ_CACHE_EXTRACTOR_DIR}" >/dev/null
./gradlew clean fatJar
popd >/dev/null

JAR_PATH="${QQ_CACHE_EXTRACTOR_DIR}/build/libs/extra-1.0.0.jar"
INSTALLER_PATH="${QQ_CACHE_EXTRACTOR_DIR}/deploy/install-ubuntu.sh"
if [[ ! -f "${JAR_PATH}" ]]; then
  echo "Build failed: jar not found: ${JAR_PATH}"
  exit 1
fi
if [[ ! -f "${INSTALLER_PATH}" ]]; then
  echo "Installer not found: ${INSTALLER_PATH}"
  exit 1
fi

REMOTE_JAR="/tmp/extra-$(date +%Y%m%d%H%M%S).jar"
REMOTE_INSTALLER="/tmp/install-extra.sh"

echo "==> Uploading jar to ${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_JAR}"
"${SCP_CMD[@]}" "${JAR_PATH}" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_JAR}"

echo "==> Uploading installer"
"${SCP_CMD[@]}" "${INSTALLER_PATH}" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_INSTALLER}"

echo "==> Installing/restarting remote service"
"${SSH_CMD[@]}" "${REMOTE_USER}@${REMOTE_HOST}" \
  "chmod +x '${REMOTE_INSTALLER}' && INSTALL_DIR='${REMOTE_QQ_CACHE_EXTRACTOR_DIR}' SERVICE_NAME='${QQ_CACHE_EXTRACTOR_SERVICE_NAME}' SERVICE_PORT='${QQ_CACHE_EXTRACTOR_PORT}' SERVICE_USER='${QQ_CACHE_EXTRACTOR_RUN_USER}' SERVICE_GROUP='${QQ_CACHE_EXTRACTOR_RUN_GROUP}' JAR_SOURCE='${REMOTE_JAR}' INSTALL_JAVA='${QQ_CACHE_EXTRACTOR_INSTALL_JAVA}' USE_SUDO='${USE_SUDO}' '${REMOTE_INSTALLER}' && rm -f '${REMOTE_INSTALLER}' '${REMOTE_JAR}'"

if [[ -n "${REMOTE_QQ_CACHE_EXTRACTOR_POST_CMD}" ]]; then
  echo "==> Running remote post command"
  "${SSH_CMD[@]}" "${REMOTE_USER}@${REMOTE_HOST}" "${REMOTE_QQ_CACHE_EXTRACTOR_POST_CMD}"
fi

echo "extra deploy finished."
