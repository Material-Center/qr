const { DeviceUtils } = require("../../common/device_util");
const { AppUtils } = require("../../common/app_util");
const { createPhoneRegisterApiClient } = require("./phone_register_api");
const { createCacheToolApiClient } = require("./cache_tool_api");
const { createImageVerifyService } = require("./image_verify_service");
const { normalizeDeviceTask } = require("./verify_code_flow");

const REMOTE_REPORT_ACTIONS = {
  enter_waiting_code: true,
  consume_code_success: true,
  register_success: true,
  fail: true,
};

const PROFILE_ALPHA_CHARS = "abcdefghijklmnopqrstuvwxyz";
const PROFILE_DIGIT_CHARS = "0123456789";
const PROFILE_SYMBOL_CHARS = "!@#$%^&*()_+-=[]{}|;:,.<>?";
const DEFAULT_HARD_MODIFY_HOST_IP_FILE = "/sdcard/ip.txt";
const DEFAULT_HARD_MODIFY_BACKUP_IP_FILE = "/sdcard/ip.txt.bak";
const HARD_MODIFY_DEBUG_WIFI_BY_DEVICE_ID = {
  "1546c952": {
    wifiSSID: "306",
    wifiPassword: "AAABBBCCC!@#",
  },
};

function readHardModifyHostIPText(path) {
  return String(files.read(path) || "");
}

function parseHardModifyHostIP(text) {
  const match = String(text || "").match(
    /内网地址:(\d{1,3}(?:\.\d{1,3}){3})/
  );
  return match ? String(match[1] || "") : "";
}

function randomFromCharset(length, charset) {
  const size = Number(length || 0) || 0;
  let result = "";
  for (let i = 0; i < size; i++) {
    const index = Math.floor(Math.random() * charset.length);
    result += charset.charAt(index);
  }
  return result;
}

function generateQQUsername() {
  const length = 5 + Math.floor(Math.random() * 6);
  return randomFromCharset(length, PROFILE_ALPHA_CHARS);
}

function pickRandomItems(items, count) {
  const source = Array.isArray(items) ? items.slice() : [];
  for (let i = source.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    const temp = source[i];
    source[i] = source[j];
    source[j] = temp;
  }
  return source.slice(0, Math.max(0, Number(count || 0) || 0));
}

function shuffleString(text) {
  const chars = String(text || "").split("");
  for (let i = chars.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    const temp = chars[i];
    chars[i] = chars[j];
    chars[j] = temp;
  }
  return chars.join("");
}

function generateQQPassword() {
  const totalLength = 10;
  const charsets = [
    PROFILE_ALPHA_CHARS,
    PROFILE_DIGIT_CHARS,
    PROFILE_SYMBOL_CHARS,
  ];
  const requiredCharsets = pickRandomItems(charsets, 2);
  let password = "";

  for (let i = 0; i < requiredCharsets.length; i++) {
    password += randomFromCharset(1, requiredCharsets[i]);
  }

  const mergedCharset = charsets.join("");
  password += randomFromCharset(totalLength - password.length, mergedCharset);
  return shuffleString(password);
}

function resolveHardModifyConfig(config) {
  const source = (config && config.resetEnvironment && config.resetEnvironment.hardModify) || {};
  return {
    enabled: source.enabled !== false,
    startupGuardEnabled: source.startupGuardEnabled !== false,
    startupRetryIntervalMs:
      Number(source.startupRetryIntervalMs || 10 * 1000) || 10 * 1000,
    startupAllowBackupAfterMs:
      Number(source.startupAllowBackupAfterMs || 60 * 1000) || 60 * 1000,
    enableWifiBeforeTrigger: source.enableWifiBeforeTrigger !== false,
    enableUsbDebugBeforeTrigger: source.enableUsbDebugBeforeTrigger !== false,
    wifiSSID: String(source.wifiSSID || "").trim(),
    wifiPassword: String(source.wifiPassword || "").trim(),
    useDeviceIdAsWifiSSID: source.useDeviceIdAsWifiSSID !== false,
    wifiConnectTimeoutMs:
      Number(source.wifiConnectTimeoutMs || 60 * 1000) || 60 * 1000,
    hostIPFile: String(
      source.hostIPFile || DEFAULT_HARD_MODIFY_HOST_IP_FILE
    ).trim(),
    backupHostIPFile: String(
      source.backupHostIPFile || DEFAULT_HARD_MODIFY_BACKUP_IP_FILE
    ).trim(),
    serverPort: Number(source.serverPort || 8088) || 8088,
    healthPath: String(source.healthPath || "/devices/").trim(),
    triggerPath: String(source.triggerPath || "/硬改自动化/").trim(),
    requestTimeoutMs: Number(source.requestTimeoutMs || 1000 * 60 * 10) || 1000 * 60 * 10,
    responseWarnMs: Number(source.responseWarnMs || 5000) || 5000,
    beforeTriggerSleepMs: Number(source.beforeTriggerSleepMs || 1000) || 1000,
    afterTriggerSleepMs: Number(source.afterTriggerSleepMs || 3000) || 3000,
  };
}

function resolveHardModifyWifiConfig(deviceId, config) {
  const normalizedDeviceId = String(deviceId || "").trim().toLowerCase();
  const debugWifi = HARD_MODIFY_DEBUG_WIFI_BY_DEVICE_ID[normalizedDeviceId];
  if (debugWifi && debugWifi.wifiSSID && debugWifi.wifiPassword) {
    return {
      wifiSSID: String(debugWifi.wifiSSID || "").trim(),
      wifiPassword: String(debugWifi.wifiPassword || "").trim(),
      source: "debug-device-map",
    };
  }

  return {
    wifiSSID: config.useDeviceIdAsWifiSSID ? deviceId : config.wifiSSID,
    wifiPassword: config.wifiPassword,
    source: config.useDeviceIdAsWifiSSID ? "device-id" : "config",
  };
}

function RegisterContext(config) {
  this.config = config;
  this.deviceId = DeviceUtils.getSerialNum();
  this.apiClient = createPhoneRegisterApiClient({
    baseURL: config.serverBaseURL,
  });
  this.cacheToolClient = createCacheToolApiClient({
    baseURL: config.cacheToolBaseURL,
  });
  this.imageVerifyConfig = config.imageVerify || {};
  this.imageVerifyService = createImageVerifyService(this.imageVerifyConfig);
  this.deviceConfig = null;
  this.currentTask = null;
  this.profileDraft = null;
  this.hardModifyTriggered = false;
  this.startupGuardReady = false;
  this.registerSuccessReported = false;
  this.heartbeatThread = null;
  this.heartbeatStopFlag = true;
}

RegisterContext.prototype.log = function (message) {
  const prefix = "[regqq][" + this.deviceId + "]";
  const text = String(message || "");
  log(prefix + " " + text);
  try {
    if (this.apiClient && typeof this.apiClient.log === "function") {
      this.apiClient.log(this.deviceId, this.getTaskId() || 0, text);
    }
  } catch (_error) {
    // 日志上报失败不能影响客户端主流程。
  }
};

RegisterContext.prototype.bindTask = function (task) {
  const normalizedTask = normalizeDeviceTask(task);
  const prevTaskId = this.getTaskId();
  this.currentTask = normalizedTask;
  const nextTaskId = this.getTaskId();
  if (!nextTaskId) {
    this.profileDraft = null;
    this.hardModifyTriggered = false;
    this.registerSuccessReported = false;
    return this.currentTask;
  }
  if (this.profileDraft && this.profileDraft.taskId === nextTaskId) {
    return this.currentTask;
  }
  if (prevTaskId !== nextTaskId) {
    this.hardModifyTriggered = false;
    this.registerSuccessReported = false;
    this.prepareQQProfileDraft(true);
  }
  return this.currentTask;
};

RegisterContext.prototype.getTaskId = function () {
  return this.currentTask && this.currentTask.id;
};

RegisterContext.prototype.getTaskPhone = function () {
  return this.currentTask && this.currentTask.phone
    ? String(this.currentTask.phone)
    : "";
};

RegisterContext.prototype.getSmsReceiveMode = function () {
  return this.currentTask && this.currentTask.smsReceiveMode
    ? String(this.currentTask.smsReceiveMode)
    : "";
};

RegisterContext.prototype.pollTask = function () {
  const data = this.apiClient.pollTask(this.deviceId);
  this.bindTask(data || null);
  return this.currentTask;
};

RegisterContext.prototype.refreshCurrentTask = function () {
  const data = this.apiClient.getTask(this.deviceId);
  this.bindTask(data || null);
  return this.currentTask;
};

RegisterContext.prototype.getDeviceConfig = function (forceRefresh) {
  if (!forceRefresh && this.deviceConfig) {
    return this.deviceConfig;
  }
  const data = this.apiClient.getConfig(this.deviceId) || {};
  this.deviceConfig = data;
  return this.deviceConfig;
};

RegisterContext.prototype.resolveImageVerifyConfig = function (forceRefresh) {
  const localConfig = this.config && this.config.imageVerify
    ? this.config.imageVerify
    : {};
  const remoteConfig = this.getDeviceConfig(forceRefresh);
  const remoteImageVerify = remoteConfig && remoteConfig.imageVerify
    ? remoteConfig.imageVerify
    : {};

  const resolved = {};
  Object.keys(localConfig || {}).forEach((key) => {
    resolved[key] = localConfig[key];
  });
  Object.keys(remoteImageVerify || {}).forEach((key) => {
    if (
      remoteImageVerify[key] !== undefined &&
      remoteImageVerify[key] !== null &&
      remoteImageVerify[key] !== ""
    ) {
      resolved[key] = remoteImageVerify[key];
    }
  });
  if (!resolved.modelName) {
    resolved.modelName = "普通模型";
  }
  if (resolved.requestId === undefined || resolved.requestId === null || resolved.requestId === "") {
    resolved.requestId = "42077360";
  }
  if (resolved.version === undefined || resolved.version === null || resolved.version === "") {
    resolved.version = "3.1.1";
  }
  if (!resolved.question) {
    resolved.question = "框出正确位置";
  }
  if (resolved.system === undefined || resolved.system === null) {
    resolved.system = "";
  }

  this.imageVerifyConfig = resolved;
  this.imageVerifyService = createImageVerifyService(this.imageVerifyConfig);
  return this.imageVerifyConfig;
};

RegisterContext.prototype.refreshTaskRuntimeConfig = function () {
  this.log("刷新任务运行时配置");
  return {
    imageVerify: this.resolveImageVerifyConfig(true),
  };
};

RegisterContext.prototype.heartbeat = function () {
  return this.apiClient.heartbeat(this.deviceId);
};

RegisterContext.prototype.report = function (action, message, statusCode) {
  if (!REMOTE_REPORT_ACTIONS[action]) {
    return {
      skipped: true,
    };
  }
  this.log("上报任务状态 action=" + action + " message=" + message);
  return this.apiClient.report(this.deviceId, action, message, statusCode);
};

RegisterContext.prototype.ensureTask = function () {
  if (!this.currentTask || !this.getTaskId()) {
    throw new Error("当前没有可执行的手机号注册任务");
  }
  return this.currentTask;
};

RegisterContext.prototype.getPendingVerifyCode = function () {
  return this.currentTask && this.currentTask.verifyCode
    ? String(this.currentTask.verifyCode)
    : "";
};

RegisterContext.prototype.prepareQQProfileDraft = function (forceRefresh) {
  const taskId = this.getTaskId();
  if (!taskId) {
    this.profileDraft = null;
    return null;
  }
  if (
    !forceRefresh &&
    this.profileDraft &&
    this.profileDraft.taskId === taskId
  ) {
    return this.profileDraft;
  }
  const username = generateQQUsername();
  this.profileDraft = {
    taskId: taskId,
    nickname: generateQQUsername(),
    username: username,
    password: generateQQPassword(),
  };
  this.log(
    "已预生成 QQ 用户名密码 username=" +
      this.profileDraft.username +
      " passwordLength=" +
      this.profileDraft.password.length
  );
  return this.profileDraft;
};

RegisterContext.prototype.getTaskNickname = function () {
  const draft = this.prepareQQProfileDraft(false);
  return draft && draft.nickname ? String(draft.nickname) : "";
};

RegisterContext.prototype.getTaskUsername = function () {
  return this.getQQUsername();
};

RegisterContext.prototype.getQQUsername = function () {
  const draft = this.prepareQQProfileDraft(false);
  return draft && draft.username ? String(draft.username) : "";
};

RegisterContext.prototype.getQQPassword = function () {
  const draft = this.prepareQQProfileDraft(false);
  return draft && draft.password ? String(draft.password) : "";
};

RegisterContext.prototype.regenerateQQProfileDraft = function () {
  return this.prepareQQProfileDraft(true);
};

RegisterContext.prototype.waitForPendingVerifyCode = function (timeoutMs, options) {
  const opts = options || {};
  const maxWaitMs = Number(timeoutMs || 0) || 0;
  const pollIntervalMs = Number(opts.pollIntervalMs || 3000) || 3000;
  const heartbeatIntervalMs =
    Number(opts.heartbeatIntervalMs || this.config.heartbeatIntervalMs || 30000) ||
    30000;
  const onLoop = typeof opts.onLoop === "function" ? opts.onLoop : null;
  const shouldStop = typeof opts.shouldStop === "function" ? opts.shouldStop : null;
  const startAt = Date.now();
  let lastHeartbeatAt = Date.now();

  while (Date.now() - startAt < maxWaitMs) {
    if (shouldStop && shouldStop()) {
      return "";
    }
    this.refreshCurrentTask();
    const code = this.getPendingVerifyCode();
    if (code) {
      return code;
    }
    if (Date.now() - lastHeartbeatAt >= heartbeatIntervalMs) {
      this.heartbeat();
      lastHeartbeatAt = Date.now();
    }
    if (onLoop) {
      onLoop();
    }
    sleep(Math.min(pollIntervalMs, Math.max(maxWaitMs - (Date.now() - startAt), 0)));
  }
  return "";
};

RegisterContext.prototype.startTaskHeartbeatLoop = function () {
  const intervalMs =
    Number(this.config.heartbeatIntervalMs || 30000) || 30000;
  if (
    this.heartbeatThread &&
    typeof this.heartbeatThread.isAlive === "function" &&
    this.heartbeatThread.isAlive()
  ) {
    return this.heartbeatThread;
  }
  const ctx = this;
  this.heartbeatStopFlag = false;
  this.heartbeatThread = threads.start(function () {
    while (!ctx.heartbeatStopFlag) {
      sleep(intervalMs);
      if (ctx.heartbeatStopFlag) {
        break;
      }
      try {
        if (ctx.getTaskId()) {
          ctx.heartbeat();
        }
      } catch (err) {
        ctx.log(
          "后台心跳失败: " +
            (err && err.message ? err.message : String(err))
        );
      }
    }
  });
  return this.heartbeatThread;
};

RegisterContext.prototype.stopTaskHeartbeatLoop = function () {
  this.heartbeatStopFlag = true;
  if (!this.heartbeatThread) {
    return;
  }
  try {
    this.heartbeatThread.interrupt();
  } catch (err) {
    this.log(
      "停止后台心跳线程失败: " +
        (err && err.message ? err.message : String(err))
    );
  }
  this.heartbeatThread = null;
};

RegisterContext.prototype.ensureQQReady = function () {
  var config = this.config || {};
  var packageName = String(config.qqPackageName || "").trim();
  var appName = String(config.qqAppName || "").trim();
  var shouldClearData = config.clearQQDataBeforeLaunch !== false;
  var clearDelayMs = Number(config.clearQQDataDelayMs || 1500) || 1500;

  this.log("打开 QQ 前关闭 USB 调试");
  this.setUsbDebugEnabledForReset(false);

  if (shouldClearData) {
    if (!packageName) {
      throw new Error("clear qq data requires qqPackageName");
    }
    this.log("清理 QQ 数据");
    AppUtils.forceStopApp(packageName);
    AppUtils.clearAppData(packageName);
    sleep(clearDelayMs);
  }

  this.log("打开 QQ 并保持前台");
  if (packageName) {
    try {
      AppUtils.waitPkgLaunched(packageName);
      return true;
    } catch (launchErr) {
      if (appName) {
        this.log("按包名拉起 QQ 失败，尝试按应用名拉起: " + launchErr.message);
        if (AppUtils.openApp(appName)) {
          return true;
        }
      }
      throw new Error("launch qq failed: " + launchErr.message);
    }
  }

  if (appName) {
    if (AppUtils.openApp(appName)) {
      return true;
    }
    throw new Error("launch qq failed");
  }

  throw new Error("qq launch config is missing");
};

RegisterContext.prototype.uploadCurrentCache = function (qqPwd) {
  // this.ensureTask();
  this.ensureCacheToolReady();
  const resolvedQQPwd = String(qqPwd || this.getQQPassword() || "");
  this.log("准备提交本地缓存到 CacheTool");
  return this.cacheToolClient.pushPhoneRegisterCache({
    deviceId: this.deviceId,
    phone: this.getTaskPhone(),
    qqPwd: resolvedQQPwd,
  });
};

RegisterContext.prototype.reportRegisterSuccessIfNeeded = function (message) {
  if (this.registerSuccessReported) {
    return {
      skipped: true,
    };
  }
  const reportMessage =
    String(message || "").trim() || "注册成功，等待上传缓存";
  const result = this.report("register_success", reportMessage);
  this.registerSuccessReported = true;
  return result;
};

RegisterContext.prototype.ensureCacheToolReady = function () {
  var config = this.config || {};
  var launchTimeoutMs =
    Number(config.cacheToolLaunchTimeoutMs || 15000) || 15000;
  var retryIntervalMs =
    Number(config.cacheToolStatusRetryIntervalMs || 1000) || 1000;
  var packageName = String(config.cacheToolPackageName || "").trim();
  var appName = String(config.cacheToolAppName || "").trim();

  this.log("打开 CacheTool 并保持前台");
  if (packageName) {
    try {
      AppUtils.waitPkgLaunched(packageName);
    } catch (launchErr) {
      if (appName) {
        this.log("按包名拉起失败，尝试按应用名拉起: " + launchErr.message);
        if (!AppUtils.openApp(appName)) {
          throw new Error("launch cache tool failed: " + launchErr.message);
        }
      } else {
        throw new Error("launch cache tool failed: " + launchErr.message);
      }
    }
  } else if (appName) {
    if (!AppUtils.openApp(appName)) {
      throw new Error("launch cache tool failed");
    }
  } else {
    throw new Error("cache tool launch config is missing");
  }

  this.log("检查 CacheTool 服务状态");
  var waitedMs = 0;
  var statusResult = null;
  while (waitedMs <= launchTimeoutMs) {
    statusResult = this.cacheToolClient.getStatusSafe();
    if (statusResult && statusResult.ok !== false && statusResult.running === true) {
      this.log("CacheTool 服务启动成功");
      return statusResult;
    }
    sleep(retryIntervalMs);
    waitedMs += retryIntervalMs;
  }

  var reason =
    statusResult && statusResult.error && statusResult.error.message
      ? statusResult.error.message
      : "status endpoint not ready";
  throw new Error("cache tool service not ready: " + reason);
};

RegisterContext.prototype.enableWifiForReset = function () {
  const config = resolveHardModifyConfig(this.config);
  const wifiConfig = resolveHardModifyWifiConfig(this.deviceId, config);
  const wifiSSID = wifiConfig.wifiSSID;
  const wifiPassword = wifiConfig.wifiPassword;
  if (!wifiSSID) {
    throw new Error("未配置 Wi-Fi 名称");
  }
  if (!wifiPassword) {
    throw new Error("未配置 Wi-Fi 密码");
  }
  this.log("尝试连接 Wi-Fi ssid=" + wifiSSID + " source=" + wifiConfig.source);
  DeviceUtils.ensureWifiConnected(wifiSSID, wifiPassword, {
    timeoutMs: config.wifiConnectTimeoutMs,
    allowUiFallback: true,
  });
  return true;
};

RegisterContext.prototype.setUsbDebugEnabledForReset = function (enabled) {
  const expected = enabled === true ? "1" : "0";
  const current = shell("settings get global adb_enabled", true);
  if (!current || current.code !== 0) {
    const message = current ? String(current.result || "") : "unknown";
    throw new Error("获取 USB 调试状态失败: " + message);
  }
  const currentValue = String(current.result || "").trim();
  this.log("USB 调试状态: " + currentValue);
  if (currentValue === expected) {
    this.log("USB 调试已是目标状态，跳过");
    return false;
  }
  const result = shell(
    "settings put global adb_enabled " + expected,
    true
  );
  if (!result || result.code !== 0) {
    const message = result ? String(result.result || "") : "unknown";
    throw new Error("设置 USB 调试状态失败: " + message);
  }
  this.log("USB 调试状态已更新为: " + expected);
  return true;
};

RegisterContext.prototype.getHardModifyHostIP = function (useBackup) {
  const config = resolveHardModifyConfig(this.config);
  let text = "";
  let usedBackup = false;

  try {
    text = readHardModifyHostIPText(config.hostIPFile);
  } catch (primaryErr) {
    if (!useBackup) {
      throw primaryErr;
    }
    this.log("读取硬改 ip 备份文件 path=" + config.backupHostIPFile);
    text = readHardModifyHostIPText(config.backupHostIPFile);
    usedBackup = true;
  }

  const hostIP = parseHardModifyHostIP(text);
  if (!hostIP) {
    throw new Error("获取中控IP失败");
  }

  this.log(
    "当前中控IP: " + hostIP + (usedBackup ? " (backup)" : " (primary)")
  );

  if (!usedBackup) {
    this.log("写入硬改 ip 备份文件 path=" + config.backupHostIPFile);
    files.write(config.backupHostIPFile, text);
    if (files.exists(config.hostIPFile)) {
      files.remove(config.hostIPFile);
      this.log("删除硬改 ip 文件 path=" + config.hostIPFile);
    }
  }

  return hostIP;
};

RegisterContext.prototype.ensureStartupGuardReady = function () {
  const config = resolveHardModifyConfig(this.config);
  if (!config.enabled || !config.startupGuardEnabled || this.startupGuardReady) {
    return true;
  }

  this.log(
    "开始检查中控IP，" +
      Math.floor(config.startupAllowBackupAfterMs / 1000) +
      "s 后会尝试获取备份IP文件"
  );

  const startedAt = Date.now();
  // eslint-disable-next-line no-constant-condition
  while (true) {
    const elapsedMs = Date.now() - startedAt;
    const useBackup = elapsedMs >= config.startupAllowBackupAfterMs;
    try {
      this.getHardModifyHostIP(useBackup);
      this.startupGuardReady = true;
      this.log("启动门禁检查通过");
      return true;
    } catch (error) {
      const message = error && error.message ? error.message : String(error);
      this.log(
        "获取中控IP失败，等待重试 useBackup=" +
          useBackup +
          " message=" +
          message
      );
      sleep(config.startupRetryIntervalMs);
    }
  }
};

RegisterContext.prototype.checkHardModifyServer = function (hostIP) {
  const config = resolveHardModifyConfig(this.config);
  const url =
    "http://" + hostIP + ":" + config.serverPort + config.healthPath;
  const response = http.get(url, { timeout: config.requestTimeoutMs });
  const body = response && response.body ? response.body.string() : "";
  this.log("硬改自动化探活响应 status=" + response.statusCode + " body=" + body);
  return response.statusCode === 200;
};

RegisterContext.prototype.triggerHardModify = function (hostIP) {
  const config = resolveHardModifyConfig(this.config);
  const url =
    "http://" +
    hostIP +
    ":" +
    config.serverPort +
    config.triggerPath +
    "?device=" +
    encodeURIComponent(this.deviceId);

  this.log("硬改自动化触发: " + url);
  const startedAt = Date.now();
  const response = http.get(url, { timeout: config.requestTimeoutMs });
  const costMs = Date.now() - startedAt;
  const body = response && response.body ? response.body.string() : "";

  this.log("硬改自动化响应耗时: " + costMs);
  if (costMs > config.responseWarnMs) {
    this.log("硬改自动化响应超时告警: " + costMs);
  }
  if (response.statusCode !== 200) {
    throw new Error("硬改自动化响应失败: " + response.statusCode + " body=" + body);
  }
  this.log("硬改自动化响应: " + body);
  this.hardModifyTriggered = true;
  return true;
};

RegisterContext.prototype.resetEnvironment = function () {
  this.log("执行环境重置");
  AppUtils.forceStopApp(this.config.qqPackageName);
  const config = resolveHardModifyConfig(this.config);
  if (!config.enabled) {
    sleep(1000);
    return;
  }
  if (this.hardModifyTriggered) {
    this.log("硬改自动化已触发，跳过");
    sleep(config.afterTriggerSleepMs);
    return;
  }
  if (config.enableWifiBeforeTrigger) {
    this.enableWifiForReset();
    sleep(config.beforeTriggerSleepMs);
  }
  if (config.enableUsbDebugBeforeTrigger) {
    if (this.setUsbDebugEnabledForReset(true)) {
      sleep(1000 * 10);
    }
  }
  const hostIP = this.getHardModifyHostIP(true);
  if (!this.checkHardModifyServer(hostIP)) {
    throw new Error("硬改自动化服务器不可用");
  }
  this.triggerHardModify(hostIP);
  sleep(config.afterTriggerSleepMs);
};

RegisterContext.prototype.solveImageVerification = function (question, options) {
  this.log("准备执行图片验证 question=" + String(question || ""));
  return this.imageVerifyService.capturePredictAndClick(question, options);
};

module.exports = {
  RegisterContext,
};
