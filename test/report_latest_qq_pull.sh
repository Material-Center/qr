#!/usr/bin/env bash

set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")" && pwd)"

LATEST_DIR="$(ls -dt "${BASE_DIR}"/qq_pull_* 2>/dev/null | head -n 1 || true)"
if [[ -z "${LATEST_DIR}" ]]; then
  echo "未找到 qq_pull_* 目录，请先执行: bash test/pull_qq_files_via_adb.sh"
  exit 1
fi

echo "使用目录: ${LATEST_DIR}"
echo ""
echo "文件清单(相对路径 | 大小字节):"

python3 - <<'PY' "${LATEST_DIR}"
import os
import sys

root = sys.argv[1]
entries = []
for dp, dns, fns in os.walk(root):
    for fn in fns:
        p = os.path.join(dp, fn)
        rel = os.path.relpath(p, root)
        size = os.path.getsize(p)
        entries.append((rel.replace("\\", "/"), size))

if not entries:
    print("  (空目录)")
else:
    for rel, size in sorted(entries):
        print(f"  {rel} | {size}")
PY

echo ""
echo "关键文件存在性:"
for path in \
  "files/wlogin_device.dat" \
  "files/tk_file" \
  "files/mobileQQ.xml" \
  "files/uid" \
  "files/user" \
  "files/mmkv"
do
  if [[ -e "${LATEST_DIR}/${path}" ]]; then
    echo "  [OK] ${path}"
  else
    echo "  [MISS] ${path}"
  fi
done
