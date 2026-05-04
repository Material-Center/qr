/**
 * @file device_util.js
 * @description 设备与网络相关工具：飞行模式、联网状态、Root/UI 下 Wi‑Fi 连接、序列号读取等。
 * @see 依赖 Auto.js 全局：shell、sleep、log、context、device、importClass、launch、id、text、className、click
 */

const { NodeUtils } = require("./node_util");
const { GestureUtils } = require("./gesture_util");

const DeviceUtils = {
  /**
   * 读取系统飞行模式是否开启。
   * @returns {boolean} true 表示飞行模式已开启；读取异常时返回 false。
   */
  getAirplaneModeEnabled() {
    try {
      const result = shell("settings get global airplane_mode_on", true).result;
      return result.trim() === "1";
    } catch (e) {
      return false;
    }
  },

  /**
   * 切换飞行模式一次后恢复为调用前的状态（用于刷新网络等场景）。
   * @param {number} [delay=1000] 关闭飞行模式后休眠毫秒数，再执行后续步骤。
   * @returns {void}
   */
  toggleAirplaneMode(delay = 1000) {
    const enabled = this.getAirplaneModeEnabled();
    console.log("当前飞行模式状态:", enabled ? "开启" : "关闭");

    // 关闭飞行模式
    console.log("关闭飞行模式");
    this.setAirplaneModeEnabled(false);
    sleep(delay);

    // 重新开启飞行模式
    console.log("重新开启飞行模式");
    this.setAirplaneModeEnabled(true);
    sleep(500);

    // 最终恢复到原始状态
    if (enabled) {
      console.log("恢复到原始状态: 开启飞行模式");
      this.setAirplaneModeEnabled(true);
    } else {
      console.log("恢复到原始状态: 关闭飞行模式");
      this.setAirplaneModeEnabled(false);
    }
  },

  /**
   * 设置飞行模式开关（写入 settings 并发送广播）。
   * @param {boolean} enabled true 开启，false 关闭。
   * @returns {void}
   */
  setAirplaneModeEnabled(enabled) {
    shell(
      "settings put global airplane_mode_on " + (enabled ? "1" : "0"),
      true,
    );
    shell(
      "am broadcast -a android.intent.action.AIRPLANE_MODE --ez state " +
        enabled,
      true,
    );
    shell(
      "am broadcast -a android.intent.action.AIRPLANE_MODE_CHANGED --ez state " +
        enabled,
      true,
    );
  },

  /**
   * 判断当前是否有可用的活动网络（会 toast 提示）。
   * @returns {boolean} 有可用网络为 true，否则 false。
   */
  getNetworkAvailable() {
    importClass(android.net.ConnectivityManager);
    var cm = context.getSystemService(context.CONNECTIVITY_SERVICE);
    var net = cm.getActiveNetworkInfo();

    if (net == null || !net.isAvailable()) {
      toastLog("网络连接不可用!");
      return false;
    } else {
      toastLog("网络连接可用!");
      return true;
    }
  },

  /**
   * 获取当前活动网络的类型名称（小写）。
   * @returns {string} 如 wifi、mobile；无活动网络时返回空字符串 ""。
   */
  getNetworkType() {
    importClass(android.net.ConnectivityManager);
    var cm = context.getSystemService(context.CONNECTIVITY_SERVICE);
    var net = cm.getActiveNetworkInfo();
    if (!net) {
      return "";
    }
    return (net.getTypeName() || "").toLowerCase();
  },

  /**
   * 使用 Root 执行系统命令连接指定 WPA2 Wi‑Fi（需已 Root 且系统支持 cmd wifi）。
   * @param {string} ssid Wi‑Fi 名称。
   * @param {string} password 密码。
   * @param {number} [timeoutMs=60000] 等待连接成功的最长毫秒数。
   * @returns {void} 当前已是 Wi‑Fi 且已连接时直接返回；成功连接后正常结束。
   * @throws {Error} 超时未连上或 shell 执行异常时抛出。
   */
  connectWifiWithRoot(ssid, password, timeoutMs = 60 * 1000) {
    log("尝试使用 root 命令连接Wi-Fi");

    if (this.getNetworkType() === "wifi") {
      log("当前WiFi已连接");
      return;
    }

    try {
      // 确保 WiFi 已启用
      log("启用 WiFi");
      shell(`svc wifi enable`, true);
      sleep(2000);

      log(`连接 WiFi: ${ssid}`);
      const securityType = "wpa2";

      const command = `cmd wifi connect-network "${ssid}" ${securityType} "${password}"`;
      log(`执行命令: ${command}`);

      const result = shell(command, true);
      log(`连接结果: ${result}`);

      log("等待 WiFi 连接 " + timeoutMs / 1000 + " 秒");

      const start = Date.now();
      while (!this.getNetworkAvailable() || this.getNetworkType() !== "wifi") {
        sleep(1000);

        if (Date.now() - start > timeoutMs) {
          throw new Error("Wi-Fi连接超时");
        }
      }
    } catch (e) {
      log(`连接WiFi时发生错误: ${e.message}`);
      throw e;
    }
  },

  /**
   * 使用系统设置页 UI 流程连接 Wi‑Fi，适配当前项目已有的 MI 8 / P10 行为。
   * @param {string} ssid Wi‑Fi 名称。
   * @param {string} password Wi‑Fi 密码。
   * @param {number} [timeoutMs=60000] 最长等待连接成功时间。
   * @returns {boolean} 成功连接返回 true。
   * @throws {Error} 连接失败或超时抛错。
   */
  connectWifiWithSettings(ssid, password, timeoutMs = 60 * 1000) {
    const networkName = String(ssid || "").trim();
    const networkPassword = String(password || "").trim();
    if (!networkName) {
      throw new Error("wifi ssid is required");
    }
    if (!networkPassword) {
      throw new Error("wifi password is required");
    }
    if (this.getNetworkAvailable() && this.getNetworkType() === "wifi") {
      log("当前 Wi-Fi 已连接，跳过设置页连接");
      return true;
    }

    const startedAt = Date.now();
    while (Date.now() - startedAt < timeoutMs) {
      if (device.model === "MI 8") {
        this._connectWifiMI8(networkName, networkPassword);
      } else {
        this._connectWifiP10(networkName, networkPassword);
      }
      sleep(2000);
      if (this.getNetworkAvailable() && this.getNetworkType() === "wifi") {
        log("设置页连接 Wi-Fi 成功");
        return true;
      }
    }
    throw new Error("Wi-Fi 设置页连接超时");
  },

  /**
   * 优先走 root 连接，失败后回退到系统设置页 UI 连接。
   * @param {string} ssid Wi‑Fi 名称。
   * @param {string} password Wi‑Fi 密码。
   * @param {{timeoutMs?: number, allowUiFallback?: boolean}} [options]
   * @returns {boolean} 连接成功返回 true。
   * @throws {Error} 所有方式失败时抛错。
   */
  ensureWifiConnected(ssid, password, options = {}) {
    const timeoutMs = Number(options.timeoutMs || 60 * 1000) || 60 * 1000;
    const allowUiFallback = options.allowUiFallback !== false;
    if (this.getNetworkAvailable() && this.getNetworkType() === "wifi") {
      log("当前 Wi-Fi 已连接");
      return true;
    }
    try {
      this.connectWifiWithRoot(ssid, password, timeoutMs);
      return true;
    } catch (rootErr) {
      log("root 连接 Wi-Fi 失败: " + rootErr.message);
      if (!allowUiFallback) {
        throw rootErr;
      }
    }
    this.connectWifiWithSettings(ssid, password, timeoutMs);
    return true;
  },

  _connectWifiP10(ssid, password) {
    log("开始通过设置页连接 Wi-Fi(P10)");
    launch("com.android.settings");
    sleep(500);
    NodeUtils.clickTextContains("无线和网络");
    sleep(500);
    NodeUtils.clickTextContains("WLAN");
    sleep(500);

    const switchNode = className("android.widget.Switch").findOne(1000);
    if (!switchNode) {
      throw new Error("未找到 Wi-Fi 开关");
    }
    if (!switchNode.checked()) {
      log("打开 Wi-Fi");
      switchNode.click();
      sleep(1000);
    }

    while (!NodeUtils.waitNodeMatchExists("text", "添加其他网络…", 500)) {
      if (this.getNetworkAvailable() && this.getNetworkType() === "wifi") {
        log("当前 Wi-Fi 已连接");
        return true;
      }
      GestureUtils.swipeToBottom();
    }

    const addWifiBtn = text("添加其他网络…").findOne(1000);
    if (!addWifiBtn || !addWifiBtn.parent()) {
      throw new Error("未找到添加其他网络按钮");
    }
    addWifiBtn.parent().click();

    const ssidInput = id("ssid").findOne(1000);
    const passwordInput = id("password").findOne(1000);
    if (!ssidInput || !passwordInput) {
      throw new Error("未找到 Wi-Fi 输入框");
    }
    ssidInput.setText(ssid);
    NodeUtils.clickTextContains("安全性");
    sleep(200);
    NodeUtils.clickTextContains("WPA/WPA2/FT PSK");
    passwordInput.setText(password);
    const connectBtn = id("btn_wifi_connect").findOne(1000);
    if (!connectBtn) {
      throw new Error("未找到 Wi-Fi 连接按钮");
    }
    connectBtn.click();
    return true;
  },

  _connectWifiMI8(ssid, password) {
    log("开始通过设置页连接 Wi-Fi(MI 8)");
    launch("com.android.settings");
    sleep(500);
    NodeUtils.clickTextContains("WLAN");

    let switchReady = false;
    while (!switchReady) {
      const switchNode = className("android.widget.CheckBox").findOne(1000);
      if (!switchNode) {
        throw new Error("未找到 Wi-Fi 复选框");
      }
      switchReady = switchNode.checked();
      if (!switchReady) {
        const rect = switchNode.bounds();
        if (!click(rect.centerX(), rect.centerY())) {
          throw new Error("开启 Wi-Fi 失败");
        }
        sleep(1000);
      }
    }

    if (this.getNetworkAvailable() && this.getNetworkType() === "wifi") {
      log("当前 Wi-Fi 已连接");
      return true;
    }

    while (!NodeUtils.waitNodeMatchExists("text", "添加网络", 500)) {
      GestureUtils.swipeToBottom();
      sleep(1000);
    }

    NodeUtils.clickTextContains("添加网络");
    sleep(1000);

    const ssidInput = id("ssid").findOne(1000);
    const passwordInput = id("password").findOne(1000);
    if (!ssidInput || !passwordInput) {
      throw new Error("未找到 Wi-Fi 输入框");
    }
    ssidInput.setText(ssid);
    sleep(500);
    click(620, 788);
    sleep(500);
    NodeUtils.clickTextContains("WPA/WPA2-Personal");
    sleep(500);
    passwordInput.setText(password);
    sleep(500);
    click(970, 166);
    return true;
  },

  /**
   * 读取设备序列号：优先 device.serial，若为 unknown 则尝试 getprop ro.serialno。
   * @returns {string} 序列号字符串。
   * @throws {Error} 读取过程异常时抛出。
   */
  getSerialNum() {
    try {
      let deviceId = device.serial;
      if (deviceId == "unknown") {
        const res = shell("getprop ro.serialno", true);
        if (res.code == 0) {
          deviceId = res.result.replace("\n", "");
        }
      }
      return deviceId;
    } catch (error) {
      console.error("get serial number error: ", error);
      throw error;
    }
  },

  /**
   * 请求截图权限
   * @returns {boolean} true 表示请求成功，false 表示请求失败
   */
  requestScreenAuthPermission() {
    let Thread = threads.start(function () {
      if (auto.service != null) {
        let Allow = textMatches(/(允许|立即开始|统一)/).findOne(10 * 1000);
        if (Allow) {
          Allow.click();
        }
      }
    });
    if (!requestScreenCapture()) {
      return false;
    } else {
      Thread.interrupt();
      return true;
    }
  },
};

module.exports = { DeviceUtils };
