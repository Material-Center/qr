#!/usr/bin/env bash
set -euo pipefail

# Unified deploy entry
# Usage:
#   ./deploy.sh server
#   ./deploy.sh web
#   ./deploy.sh all
#   ./deploy.sh service

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

###############################################################################
# Deploy config (edit here)
###############################################################################
REMOTE_HOST="210.16.170.132"
REMOTE_USER="root"
REMOTE_PORT="22"
# 认证方式二选一：
# 1) 密钥：填写 SSH_KEY，REMOTE_PASSWORD 留空
# 2) 密码：填写 REMOTE_PASSWORD，SSH_KEY 留空
SSH_KEY=""
REMOTE_PASSWORD="Ca9B0VXUhkLoNleF"

# web
REMOTE_WEB_DIR="/var/www/qr-web"
REMOTE_WEB_POST_CMD=""

# server
REMOTE_SERVER_DIR="/opt/qr-server"
REMOTE_UPDATE_SCRIPT="/opt/qr-server/remote-update-server.sh"
SERVICE_NAME="qr-server"
USE_SUDO="1"
REMOTE_SERVER_POST_CMD=""
BINARY_NAME="server"
SERVICE_RUN_USER="root"
SERVICE_RUN_GROUP="root"
SERVICE_DESCRIPTION="QR Server"
RESTART_AFTER_INSTALL="0"
GOOS_TARGET="linux"
GOARCH_TARGET="amd64"
CGO_ENABLED_TARGET="0"
INCLUDE_CONFIG="0"
###############################################################################

ACTION="${1:-}"

if [[ -z "${ACTION}" ]]; then
  echo "Usage: ./deploy.sh [server|web|all|service]"
  exit 1
fi

if [[ "${REMOTE_HOST}" == "127.0.0.1" || "${REMOTE_USER}" == "deploy" ]]; then
  echo "Please edit deploy config in ./deploy.sh first."
  echo "Current REMOTE_HOST=${REMOTE_HOST}, REMOTE_USER=${REMOTE_USER}"
  exit 1
fi

if [[ -z "${SSH_KEY}" && -z "${REMOTE_PASSWORD}" ]]; then
  echo "Please set one auth method in ./deploy.sh: SSH_KEY or REMOTE_PASSWORD"
  exit 1
fi

if [[ -n "${SSH_KEY}" && -n "${REMOTE_PASSWORD}" ]]; then
  echo "Both SSH_KEY and REMOTE_PASSWORD are set, password mode will be used."
fi

export REMOTE_HOST REMOTE_USER REMOTE_PORT SSH_KEY REMOTE_PASSWORD
export REMOTE_WEB_DIR REMOTE_WEB_POST_CMD
export REMOTE_SERVER_DIR REMOTE_SERVER_POST_CMD
export REMOTE_UPDATE_SCRIPT SERVICE_NAME USE_SUDO
export BINARY_NAME GOOS_TARGET GOARCH_TARGET CGO_ENABLED_TARGET INCLUDE_CONFIG
export SERVICE_RUN_USER SERVICE_RUN_GROUP SERVICE_DESCRIPTION RESTART_AFTER_INSTALL

run_web() {
  "${SCRIPT_DIR}/deploy/scripts/deploy-web.sh"
}

run_server() {
  "${SCRIPT_DIR}/deploy/scripts/deploy-server.sh"
}

run_service() {
  "${SCRIPT_DIR}/deploy/scripts/deploy-systemd-service.sh"
}

case "${ACTION}" in
  web)
    run_web
    ;;
  server)
    run_server
    ;;
  service)
    run_service
    ;;
  all)
    run_server
    run_web
    ;;
  *)
    echo "Unknown action: ${ACTION}"
    echo "Usage: ./deploy.sh [server|web|all|service]"
    exit 1
    ;;
esac

