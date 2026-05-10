#!/usr/bin/env bash
set -euo pipefail

# Install/update extra on Ubuntu.
#
# Optional env:
#   INSTALL_DIR=/opt/extra
#   SERVICE_NAME=extra
#   SERVICE_USER=root
#   SERVICE_GROUP=root
#   SERVICE_PORT=19091
#   JAR_SOURCE=/tmp/extra.jar
#   INSTALL_JAVA=1
#   JAVA_PACKAGE=openjdk-17-jre-headless
#   USE_SUDO=1

INSTALL_DIR="${INSTALL_DIR:-/opt/extra}"
SERVICE_NAME="${SERVICE_NAME:-extra}"
SERVICE_USER="${SERVICE_USER:-root}"
SERVICE_GROUP="${SERVICE_GROUP:-root}"
SERVICE_PORT="${SERVICE_PORT:-19091}"
JAR_SOURCE="${JAR_SOURCE:-}"
INSTALL_JAVA="${INSTALL_JAVA:-1}"
JAVA_PACKAGE="${JAVA_PACKAGE:-openjdk-17-jre-headless}"
USE_SUDO="${USE_SUDO:-1}"

SUDO=""
if [[ "${USE_SUDO}" == "1" && "$(id -u)" != "0" ]]; then
  SUDO="sudo"
fi

if [[ ! -f /etc/os-release ]]; then
  echo "Cannot detect OS: /etc/os-release not found"
  exit 1
fi

. /etc/os-release
if [[ "${ID:-}" != "ubuntu" && "${ID_LIKE:-}" != *"ubuntu"* && "${ID_LIKE:-}" != *"debian"* ]]; then
  echo "This installer is intended for Ubuntu/Debian. Current ID=${ID:-unknown}"
  exit 1
fi

if [[ "${INSTALL_JAVA}" == "1" ]] && ! command -v java >/dev/null 2>&1; then
  echo "==> Installing Java runtime: ${JAVA_PACKAGE}"
  ${SUDO} apt-get update
  ${SUDO} env DEBIAN_FRONTEND=noninteractive apt-get install -y "${JAVA_PACKAGE}"
fi

if ! command -v java >/dev/null 2>&1; then
  echo "java command not found. Install Java first or set INSTALL_JAVA=1."
  exit 1
fi

echo "==> Preparing install dir: ${INSTALL_DIR}"
${SUDO} mkdir -p "${INSTALL_DIR}"

if [[ -n "${JAR_SOURCE}" ]]; then
  if [[ ! -f "${JAR_SOURCE}" ]]; then
    echo "JAR_SOURCE not found: ${JAR_SOURCE}"
    exit 1
  fi
  echo "==> Installing jar from ${JAR_SOURCE}"
  ${SUDO} cp -f "${JAR_SOURCE}" "${INSTALL_DIR}/app.jar"
fi

if [[ ! -f "${INSTALL_DIR}/app.jar" ]]; then
  echo "Missing ${INSTALL_DIR}/app.jar. Provide JAR_SOURCE or copy app.jar manually."
  exit 1
fi

${SUDO} chown -R "${SERVICE_USER}:${SERVICE_GROUP}" "${INSTALL_DIR}" || true
${SUDO} chmod 755 "${INSTALL_DIR}"
${SUDO} chmod 644 "${INSTALL_DIR}/app.jar"

SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
echo "==> Writing systemd service: ${SERVICE_FILE}"
${SUDO} tee "${SERVICE_FILE}" >/dev/null <<EOF
[Unit]
Description=extra
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
WorkingDirectory=${INSTALL_DIR}
ExecStart=/usr/bin/java -jar ${INSTALL_DIR}/app.jar --port=${SERVICE_PORT}
Restart=always
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF

echo "==> Reloading and restarting service: ${SERVICE_NAME}"
${SUDO} systemctl daemon-reload
${SUDO} systemctl enable "${SERVICE_NAME}"
${SUDO} systemctl restart "${SERVICE_NAME}"
${SUDO} systemctl --no-pager --full status "${SERVICE_NAME}" || true

echo "extra installed."
echo "Health check: curl http://127.0.0.1:${SERVICE_PORT}/health"
