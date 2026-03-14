#!/usr/bin/env bash
set -euo pipefail

# Reinstall MySQL + install Redis, then initialize database users.
# Recommended for Debian/Ubuntu.
#
# Usage:
#   sudo bash install-mysql-redis.sh
#
# Optional env:
#   FORCE_REINSTALL_MYSQL=1
#   INSTALL_REDIS=1
#   MYSQL_ROOT_PASSWORD=NewRootPass123!
#   APP_DB_NAME=qr
#   APP_DB_USER=qr
#   APP_DB_PASSWORD=QrPass123!
#   REDIS_BIND=127.0.0.1
#   REDIS_PORT=6379

FORCE_REINSTALL_MYSQL="${FORCE_REINSTALL_MYSQL:-1}"
INSTALL_REDIS="${INSTALL_REDIS:-1}"

MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-123456}"
APP_DB_NAME="${APP_DB_NAME:-qr}"
APP_DB_USER="${APP_DB_USER:-qr}"
APP_DB_PASSWORD="${APP_DB_PASSWORD:-123456}"

REDIS_BIND="${REDIS_BIND:-127.0.0.1}"
REDIS_PORT="${REDIS_PORT:-6379}"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Please run as root: sudo bash $0"
  exit 1
fi

if ! command -v apt-get >/dev/null 2>&1; then
  echo "This script currently targets Debian/Ubuntu (apt-get)."
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive

echo "==> Installing base packages"
apt-get update
apt-get install -y ca-certificates lsb-release gnupg

if [[ "${FORCE_REINSTALL_MYSQL}" == "1" ]]; then
  echo "==> Reinstalling MySQL"
  systemctl stop mysql || true
  apt-get purge -y mysql-server mysql-client mysql-common mysql-server-core-* mysql-client-core-* || true
  apt-get autoremove -y || true
  rm -rf /etc/mysql /var/lib/mysql /var/log/mysql
fi

apt-get update
apt-get install -y mysql-server

if [[ "${INSTALL_REDIS}" == "1" ]]; then
  apt-get install -y redis-server
fi

echo "==> Fixing MySQL runtime directory"
mkdir -p /var/run/mysqld
chown mysql:mysql /var/run/mysqld
chmod 755 /var/run/mysqld

echo "==> Starting MySQL"
systemctl enable mysql
systemctl restart mysql

echo "==> Initializing MySQL users and database"
mysql <<EOF
ALTER USER 'root'@'localhost' IDENTIFIED BY '${MYSQL_ROOT_PASSWORD}';
CREATE USER IF NOT EXISTS 'root'@'127.0.0.1' IDENTIFIED BY '${MYSQL_ROOT_PASSWORD}';
ALTER USER 'root'@'127.0.0.1' IDENTIFIED BY '${MYSQL_ROOT_PASSWORD}';
GRANT ALL PRIVILEGES ON *.* TO 'root'@'127.0.0.1' WITH GRANT OPTION;

CREATE DATABASE IF NOT EXISTS \`${APP_DB_NAME}\` DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;
CREATE USER IF NOT EXISTS '${APP_DB_USER}'@'localhost' IDENTIFIED BY '${APP_DB_PASSWORD}';
CREATE USER IF NOT EXISTS '${APP_DB_USER}'@'127.0.0.1' IDENTIFIED BY '${APP_DB_PASSWORD}';
ALTER USER '${APP_DB_USER}'@'localhost' IDENTIFIED BY '${APP_DB_PASSWORD}';
ALTER USER '${APP_DB_USER}'@'127.0.0.1' IDENTIFIED BY '${APP_DB_PASSWORD}';
GRANT ALL PRIVILEGES ON \`${APP_DB_NAME}\`.* TO '${APP_DB_USER}'@'localhost';
GRANT ALL PRIVILEGES ON \`${APP_DB_NAME}\`.* TO '${APP_DB_USER}'@'127.0.0.1';
FLUSH PRIVILEGES;
EOF

if [[ "${INSTALL_REDIS}" == "1" ]]; then
  echo "==> Configuring Redis"
  if [[ -f "/etc/redis/redis.conf" ]]; then
    sed -i.bak -E "s/^bind .*/bind ${REDIS_BIND}/" /etc/redis/redis.conf || true
    sed -i.bak -E "s/^port .*/port ${REDIS_PORT}/" /etc/redis/redis.conf || true
  fi
  systemctl enable redis-server
  systemctl restart redis-server
fi

echo "Install finished."
echo "MySQL root password: ${MYSQL_ROOT_PASSWORD}"
echo "App DB: ${APP_DB_NAME}"
echo "App DB user: ${APP_DB_USER}"
if [[ "${INSTALL_REDIS}" == "1" ]]; then
  echo "Redis bind: ${REDIS_BIND}, port: ${REDIS_PORT}"
fi

echo "Verify MySQL:"
echo "  mysql --protocol=TCP -h127.0.0.1 -P3306 -u${APP_DB_USER} -p -e \"SELECT 1;\""
