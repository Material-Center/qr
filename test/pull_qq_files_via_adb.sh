#!/usr/bin/env bash

set -euo pipefail

# 用法:
#   bash test/pull_qq_files_via_adb.sh
#   bash test/pull_qq_files_via_adb.sh <serial>

SERIAL="${1:-}"
ADB_BIN="${ADB_BIN:-adb}"

if [[ -n "${SERIAL}" ]]; then
  ADB_CMD=("${ADB_BIN}" -s "${SERIAL}")
else
  ADB_CMD=("${ADB_BIN}")
fi

QQ_DATA_DIR="/data/data/com.tencent.mobileqq"
REMOTE_STAGE_DIR="/data/local/tmp/qq_pull_stage"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
LOCAL_OUT_DIR="$(cd "$(dirname "$0")" && pwd)/qq_pull_${TIMESTAMP}"

echo "[1/6] 检查 adb 连接..."
"${ADB_CMD[@]}" get-state >/dev/null

echo "[2/6] 清理并创建远端暂存目录..."
"${ADB_CMD[@]}" shell "su -c 'rm -rf ${REMOTE_STAGE_DIR} && mkdir -p ${REMOTE_STAGE_DIR}/files'"

echo "[3/6] 从 QQ 私有目录复制文件到远端暂存目录..."
"${ADB_CMD[@]}" shell "su -c '
  cp -f ${QQ_DATA_DIR}/files/wlogin_device.dat ${REMOTE_STAGE_DIR}/files/wlogin_device.dat 2>/dev/null || true
  cp -f ${QQ_DATA_DIR}/databases/tk_file ${REMOTE_STAGE_DIR}/files/tk_file 2>/dev/null || true
  cp -f ${QQ_DATA_DIR}/shared_prefs/mobileQQ.xml ${REMOTE_STAGE_DIR}/files/mobileQQ.xml 2>/dev/null || true
  cp -rf ${QQ_DATA_DIR}/files/uid ${REMOTE_STAGE_DIR}/files/uid 2>/dev/null || true
  cp -rf ${QQ_DATA_DIR}/files/user ${REMOTE_STAGE_DIR}/files/user 2>/dev/null || true
  cp -rf ${QQ_DATA_DIR}/files/mmkv ${REMOTE_STAGE_DIR}/files/mmkv 2>/dev/null || true
  chmod -R 777 ${REMOTE_STAGE_DIR}
'"

echo "[4/6] 拉取到本地 test 目录..."
mkdir -p "${LOCAL_OUT_DIR}"
"${ADB_CMD[@]}" pull "${REMOTE_STAGE_DIR}/files" "${LOCAL_OUT_DIR}/"

echo "[5/6] 清理远端暂存目录..."
"${ADB_CMD[@]}" shell "su -c 'rm -rf ${REMOTE_STAGE_DIR}'"

echo "[6/6] 完成"
echo "本地导出目录: ${LOCAL_OUT_DIR}"
echo "下一步可执行: bash test/report_latest_qq_pull.sh"
