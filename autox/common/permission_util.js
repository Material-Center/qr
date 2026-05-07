/**
 * @file permission_util.js
 * @description Android 权限工具：外部文件读写权限检测和申请。
 * @see 依赖 Auto.js/AutoX 全局：context、runtime、sleep、toastLog、importClass
 */

const READ_EXTERNAL_STORAGE = "android.permission.READ_EXTERNAL_STORAGE";
const WRITE_EXTERNAL_STORAGE = "android.permission.WRITE_EXTERNAL_STORAGE";

function getSdkInt() {
  importClass(android.os.Build);
  return Build.VERSION.SDK_INT;
}

function hasRuntimePermission(permission) {
  if (getSdkInt() < 23) {
    return true;
  }
  if (
    typeof context === "undefined" ||
    !context ||
    typeof context.checkSelfPermission !== "function"
  ) {
    return false;
  }
  importClass(android.content.pm.PackageManager);
  return context.checkSelfPermission(permission) === PackageManager.PERMISSION_GRANTED;
}

function waitUntil(predicate, timeoutMs, intervalMs) {
  const maxWaitMs = Number(timeoutMs || 0) || 0;
  const interval = Number(intervalMs || 300) || 300;
  const startedAt = Date.now();
  while (Date.now() - startedAt < maxWaitMs) {
    if (predicate()) {
      return true;
    }
    sleep(interval);
  }
  return predicate();
}

function startPermissionAllowWatcher(timeoutMs, allowTexts) {
  if (typeof threads === "undefined" || !threads) {
    return null;
  }
  if (typeof textMatches !== "function") {
    return null;
  }

  const maxWaitMs = Number(timeoutMs || 8000) || 8000;
  const pattern = allowTexts || /允许|同意|继续|始终允许|仅在使用中允许|确定|开启|打开/;

  return threads.start(function () {
    const deadline = Date.now() + maxWaitMs;
    while (Date.now() < deadline) {
      const btn = textMatches(pattern).findOne(300);
      if (btn) {
        if (typeof btn.click === "function" && btn.click()) {
          return;
        }
        if (typeof click === "function") {
          const rect = btn.bounds();
          click(rect.centerX(), rect.centerY());
          return;
        }
      }
      sleep(200);
    }
  });
}

function stopWatcher(watcher) {
  if (watcher && typeof watcher.isAlive === "function" && watcher.isAlive()) {
    watcher.interrupt();
  }
}

const PermissionUtils = {
  /**
   * 判断是否具备旧版外部存储读写权限。
   * Android 6 以下默认返回 true；Android 6+ 检查 READ/WRITE_EXTERNAL_STORAGE。
   * @returns {boolean} 已具备权限为 true。
   */
  hasLegacyExternalStoragePermission() {
    return (
      hasRuntimePermission(READ_EXTERNAL_STORAGE) &&
      hasRuntimePermission(WRITE_EXTERNAL_STORAGE)
    );
  },

  /**
   * 申请旧版外部存储读写权限。
   * 注意：打包应用仍需在 Manifest/project 配置中声明对应权限，否则系统不会授权。
   * @param {number} [timeoutMs=8000] 申请后等待权限生效的最长毫秒数。
   * @returns {boolean} 已具备权限为 true。
   */
  requestLegacyExternalStoragePermission(timeoutMs = 8000) {
    if (this.hasLegacyExternalStoragePermission()) {
      return true;
    }
    if (
      typeof runtime === "undefined" ||
      !runtime ||
      typeof runtime.requestPermissions !== "function"
    ) {
      return false;
    }
    const watcher = startPermissionAllowWatcher(timeoutMs);
    runtime.requestPermissions(["read_external_storage", "write_external_storage"]);
    const granted = waitUntil(
      () => this.hasLegacyExternalStoragePermission(),
      timeoutMs,
      300,
    );
    stopWatcher(watcher);
    return granted;
  },

  /**
   * 确保外部文件读写权限可用。
   * 仅申请 READ/WRITE_EXTERNAL_STORAGE，不跳转“管理所有文件”设置页。
   * @param {{timeoutMs?: number, toast?: boolean}} [options]
   * @returns {boolean} 权限已具备为 true，否则 false。
   */
  ensureExternalStoragePermission(options = {}) {
    const timeoutMs = Number(options.timeoutMs || 15000) || 15000;
    const showToast = options.toast !== false;

    if (showToast && !this.hasLegacyExternalStoragePermission()) {
      this.toast("正在申请外部文件读写权限");
    }
    return this.requestLegacyExternalStoragePermission(timeoutMs);
  },

  toast(message) {
    if (typeof toastLog === "function") {
      toastLog(message);
      return;
    }
    if (typeof log === "function") {
      log(message);
    }
  },
};

module.exports = { PermissionUtils };
