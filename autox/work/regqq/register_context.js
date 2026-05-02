const { DeviceUtils } = require("../../common/device_util");
const { AppUtils } = require("../../common/app_util");
const { createPhoneRegisterApiClient } = require("./phone_register_api");
const { createCacheToolApiClient } = require("./cache_tool_api");
const { createImageVerifyService } = require("./image_verify_service");

function RegisterContext(config) {
  this.config = config;
  this.deviceId = DeviceUtils.getSerialNum();
  this.apiClient = createPhoneRegisterApiClient({
    baseURL: config.serverBaseURL,
  });
  this.cacheToolClient = createCacheToolApiClient({
    baseURL: config.cacheToolBaseURL,
  });
  this.imageVerifyService = createImageVerifyService(config.imageVerify || {});
  this.currentTask = null;
}

RegisterContext.prototype.log = function (message) {
  const prefix = "[regqq][" + this.deviceId + "]";
  log(prefix + " " + message);
};

RegisterContext.prototype.bindTask = function (task) {
  this.currentTask = task || null;
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

RegisterContext.prototype.heartbeat = function () {
  return this.apiClient.heartbeat(this.deviceId);
};

RegisterContext.prototype.report = function (action, message, statusCode) {
  this.log("上报任务状态 action=" + action + " message=" + message);
  return this.apiClient.report(this.deviceId, action, message, statusCode);
};

RegisterContext.prototype.ensureTask = function () {
  if (!this.currentTask || !this.currentTask.id) {
    throw new Error("当前没有可执行的手机号注册任务");
  }
  return this.currentTask;
};

RegisterContext.prototype.uploadCurrentCache = function (qqPwd) {
  // this.ensureTask();
  this.ensureCacheToolReady();
  this.log("准备提交本地缓存到 CacheTool");
  return this.cacheToolClient.pushPhoneRegisterCache({
    deviceId: this.deviceId,
    phone: this.getTaskPhone(),
    qqPwd: qqPwd || "",
  });
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

RegisterContext.prototype.resetEnvironment = function () {
  this.log("执行环境重置");
  AppUtils.forceStopApp(this.config.qqPackageName);
  sleep(1000);
};

RegisterContext.prototype.solveImageVerification = function (question, options) {
  this.log("准备执行图片验证 question=" + String(question || ""));
  return this.imageVerifyService.capturePredictAndClick(question, options);
};

module.exports = {
  RegisterContext,
};
