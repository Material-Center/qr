#!/usr/bin/env bash

set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")" && pwd)"
DECRYPT_DIR="${BASE_DIR}/decrypt_cache"
LATEST_DIR="$(ls -dt "${BASE_DIR}"/qq_pull_* 2>/dev/null | head -n 1 || true)"

if [[ -z "${LATEST_DIR}" ]]; then
  echo "未找到 qq_pull_* 目录，请先执行: bash test/pull_qq_files_via_adb.sh"
  exit 1
fi

TK_FILE="${LATEST_DIR}/files/tk_file"
WLOGIN_FILE="${LATEST_DIR}/files/wlogin_device.dat"

if [[ ! -f "${TK_FILE}" ]]; then
  echo "缺少文件: ${TK_FILE}"
  exit 1
fi
if [[ ! -f "${WLOGIN_FILE}" ]]; then
  echo "缺少文件: ${WLOGIN_FILE}"
  exit 1
fi
if [[ ! -f "${DECRYPT_DIR}/main.go" ]]; then
  echo "缺少解密工具目录: ${DECRYPT_DIR}"
  exit 1
fi
if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "未安装 sqlite3，请先安装（macOS: brew install sqlite）"
  exit 1
fi
if ! command -v go >/dev/null 2>&1; then
  echo "未安装 go，请先安装 Go 环境"
  exit 1
fi

echo "使用提取目录: ${LATEST_DIR}"
echo "开始解密 tk_file ..."

pushd "${DECRYPT_DIR}" >/dev/null
go run . -tk "${TK_FILE}" -wlogin "${WLOGIN_FILE}"

if [[ ! -f "out.dat" ]]; then
  popd >/dev/null
  echo "解密未生成 out.dat"
  exit 1
fi

OUT_COPY="${LATEST_DIR}/decode_out.dat"
cp -f "out.dat" "${OUT_COPY}"

if [[ -f "SerializationDumper.jar" ]]; then
  java -jar SerializationDumper.jar -r "${OUT_COPY}" > "${LATEST_DIR}/decode_dump.txt" || true
  echo "已输出: ${LATEST_DIR}/decode_dump.txt"
fi
popd >/dev/null

echo "解密完成:"
echo "  - ${OUT_COPY}"
if [[ -f "${LATEST_DIR}/decode_dump.txt" ]]; then
  echo "  - ${LATEST_DIR}/decode_dump.txt"
fi
