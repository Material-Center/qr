#!/usr/bin/env bash
set -euo pipefail

# Run on remote host.
# Required env:
#   REMOTE_SERVER_DIR
#   BINARY_NAME
#   UPLOAD_ARCHIVE
#
# Optional env:
#   SERVICE_NAME=""            # systemd service name
#   USE_SUDO=1                 # use sudo for systemctl
#   REMOTE_SERVER_POST_CMD=""  # extra command after restart

REMOTE_SERVER_DIR="${REMOTE_SERVER_DIR:-}"
BINARY_NAME="${BINARY_NAME:-server}"
UPLOAD_ARCHIVE="${UPLOAD_ARCHIVE:-}"
SERVICE_NAME="${SERVICE_NAME:-}"
USE_SUDO="${USE_SUDO:-1}"
REMOTE_SERVER_POST_CMD="${REMOTE_SERVER_POST_CMD:-}"

if [[ -z "${REMOTE_SERVER_DIR}" || -z "${UPLOAD_ARCHIVE}" ]]; then
  echo "Missing required env. Need: REMOTE_SERVER_DIR, UPLOAD_ARCHIVE"
  exit 1
fi

for cmd in tar mkdir cp rm chmod; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Missing required command on remote host: ${cmd}"
    exit 1
  fi
done

SYSTEMCTL_CMD="systemctl"
if [[ "${USE_SUDO}" == "1" ]]; then
  SYSTEMCTL_CMD="sudo systemctl"
fi

stop_service() {
  if [[ -n "${SERVICE_NAME}" ]]; then
    ${SYSTEMCTL_CMD} stop "${SERVICE_NAME}"
  fi
}

start_service() {
  if [[ -n "${SERVICE_NAME}" ]]; then
    ${SYSTEMCTL_CMD} start "${SERVICE_NAME}"
  fi
}

mkdir -p "${REMOTE_SERVER_DIR}"

BACKUP_DIR="${REMOTE_SERVER_DIR}/backup"
TARGET_BIN="${REMOTE_SERVER_DIR}/${BINARY_NAME}"
BACKUP_BIN=""

if [[ -f "${TARGET_BIN}" ]]; then
  mkdir -p "${BACKUP_DIR}"
  BACKUP_BIN="${BACKUP_DIR}/${BINARY_NAME}.$(date +%Y%m%d%H%M%S)"
  cp -f "${TARGET_BIN}" "${BACKUP_BIN}"
fi

restore_if_needed() {
  if [[ -n "${BACKUP_BIN}" && -f "${BACKUP_BIN}" ]]; then
    cp -f "${BACKUP_BIN}" "${TARGET_BIN}"
  fi
}

stop_service

if ! tar -xzf "${UPLOAD_ARCHIVE}" -C "${REMOTE_SERVER_DIR}"; then
  echo "Extract failed, restoring previous binary."
  restore_if_needed
  start_service
  exit 1
fi

chmod +x "${TARGET_BIN}"
rm -f "${UPLOAD_ARCHIVE}"

if ! start_service; then
  echo "Service start failed, rolling back."
  restore_if_needed
  start_service || true
  exit 1
fi

if [[ -n "${REMOTE_SERVER_POST_CMD}" ]]; then
  eval "${REMOTE_SERVER_POST_CMD}"
fi

echo "Remote server update finished."
