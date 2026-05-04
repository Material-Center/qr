/**
 * @file app_util.js
 * @description 应用启动、强制停止、系统「无响应」弹窗处理，以及 Activity 轮询等待。
 * @see 依赖 Auto.js 全局：launchApp、back、sleep、text、shell、launch、waitForPackage、currentActivity
 */

const AppUtils = {
  /**
   * 按应用名称启动应用（与 Auto.js launchApp 行为一致）。
   * @param {string} name 应用名称（显示名）。
   * @returns {boolean} 是否成功打开（取决于 Auto.js 实现）。
   */
  openApp(name) {
    return launchApp(name);
  },

  /**
   * 多次返回键尝试回到桌面或退出当前界面（粗略「关应用」流程，非 force-stop）。
   * @returns {void}
   */
  closeApp() {
    var i = 0;
    while (i < 5) {
      back();
      sleep(1000);
      i++;
    }
    back();
    sleep(1000);
    back();
  },

  /**
   * 先执行 closeApp 再重新 openApp。
   * @param {string} name 应用名称。
   * @returns {void}
   */
  backHomeReloadApp(name) {
    AppUtils.closeApp();
    sleep(1000);
    AppUtils.openApp(name);
  },

  /**
   * 使用 am force-stop 强制停止指定包名应用。
   * @param {string} packageName Android 包名，如 com.tencent.mobileqq。
   * @returns {void} 异常时仅打印日志，不向外抛出。
   */
  forceStopApp(packageName) {
    try {
      shell("am force-stop " + packageName, true);
    } catch (e) {
      console.log(e);
    }
  },

  /**
   * 使用 pm clear 清理指定应用数据。
   * @param {string} packageName Android 包名，如 com.tencent.mobileqq。
   * @returns {boolean} 清理成功返回 true。
   * @throws {Error} 包名为空或 shell 返回失败时抛出。
   */
  clearAppData(packageName) {
    const pkg = String(packageName || "").trim();
    if (!pkg) {
      throw new Error("clear app data packageName is required");
    }
    const result = shell("pm clear " + pkg, true);
    if (!result || result.code !== 0) {
      throw new Error("pm clear failed: " + (result ? result.result : ""));
    }
    return true;
  },

  /**
   * 若出现「等待 / 关闭应用」类无响应对话框，则点击关闭应用。
   * @returns {boolean} 找到并处理了弹窗为 true，否则 false。
   */
  noResponse() {
    var e;
    if (text("等待").findOnce() && (e = text("关闭应用").findOnce())) {
      e.click();
      sleep(1000);
      return true;
    }
    return false;
  },

  /**
   * 启动包并等待其出现在前台（waitForPackage）。
   * @param {string} pkg 包名。
   * @returns {void}
   * @throws {Error} launch 失败时抛出。
   */
  waitPkgLaunched(pkg) {
    if (!launch(pkg)) {
      throw new Error("Launch " + pkg + " failed.");
    }
    waitForPackage(pkg, 200);
  },

  /**
   * 轮询直到当前 Activity 与给定 prev 不同，或超时。
   * @param {string} prev 上一 Activity 标识（与 currentActivity() 返回值比较）。
   * @param {number} maxWaitMs 最长等待毫秒数。
   * @returns {boolean} 已变化为 true，超时仍未变化为 false。
   */
  waitForActivityChanged(prev, maxWaitMs) {
    var n = 0;
    while (n < maxWaitMs) {
      sleep(50);
      var cur = currentActivity();
      if (cur != prev) {
        return true;
      }
      n += 50;
    }
    return false;
  },

  /**
   * 轮询直到当前 Activity 与初始值不同，返回新的 Activity；超时返回 null。
   * @param {string} [initial] 初始 Activity，缺省时取当前 currentActivity()。
   * @param {number} maxWaitMs 最长等待毫秒数。
   * @returns {string|null} 变化后的 Activity 名；超时为 null。
   */
  waitForCurrentActivityChanged(initial, maxWaitMs) {
    initial = initial || currentActivity();
    var n = 0;
    while (n < maxWaitMs) {
      sleep(50);
      var cur = currentActivity();
      if (cur != initial) {
        return cur;
      }
      n += 50;
    }
    return null;
  },

  /**
   * 轮询直到当前 Activity 等于目标 activityId，或超时。
   * @param {string} activityId 目标 Activity 标识（与 currentActivity() 一致比较）。
   * @param {number} maxWaitMs 最长等待毫秒数。
   * @returns {boolean} 已到达目标为 true，超时为 false。
   */
  waitForActivity(activityId, maxWaitMs) {
    var n = 0;
    while (n < maxWaitMs) {
      sleep(50);
      var cur = currentActivity();
      if (cur == activityId) {
        return true;
      }
      n += 50;
    }
    return false;
  },
};

module.exports = { AppUtils };
