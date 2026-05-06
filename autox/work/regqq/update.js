const { DeviceUtils } = require("../../common/device_util");
const { NodeUtils } = require("../../common/node_util");
const { GestureUtils } = require("../../common/gesture_util");

const deviceId = DeviceUtils.getSerialNum();

const wifiName = deviceId;
// const wifiName = "2505";
const pwd = "12345678";

main();

function main() {
  try {
    const model = device.model;
    toastLog("当前型号：" + model);
    if (model === "MI 8") {
      connectWifiMI8();
    } else {
      connectWifiP10();
    }

    log("开始获取脚本");
    const rsp = http.get("https://www.qq123qq.com/remote/scripts/regqq/main.js");

    const code = rsp.body.string();

    const filepath = files.cwd() + "/main.js";
    log("写入到" + filepath);

    files.write(filepath, code);

    log("开始执行");
    global.require(filepath);
  } catch (err) {
    toastLog("执行失败 " + err.message);
    setTimeout(() => {
      main();
    }, 1000);
  }
}

function connectWifiP10() {
  log("开始连接Wi-Fi");

  if (getNetworkState()) {
    log("当前wifi已链接");
    return;
  }

  launch("com.android.settings");
  sleep(500);
  NodeUtils.clickTextContains("无线和网络");
  sleep(500);
  NodeUtils.clickTextContains("WLAN");
  sleep(500);

  const btn = className("android.widget.Switch").findOne();

  if (!btn.checked()) {
    log("打开Wi-Fi");
    btn.click();
    sleep(1000);
  }

  log("开始向下滑动");
  while (!NodeUtils.waitNodeMatchExists("text", "添加其他网络…", 500)) {
    // 避免成功连接是走来这里
    if (getNetworkState()) {
      log("当前wifi已链接");
      return;
    }

    GestureUtils.swipeToBottom();
  }

  log("开始添加网络");
  const addWifiBtn = text("添加其他网络…").findOne().parent();
  if (addWifiBtn) {
    addWifiBtn.click();
  }

  id("ssid").findOne().setText(wifiName);
  NodeUtils.clickTextContains("安全性");
  sleep(200);
  NodeUtils.clickTextContains("WPA/WPA2/FT PSK");
  id("password").findOne().setText(pwd);
  id("btn_wifi_connect").findOne().click();

  sleep(2000);

  connectWifiP10();
}

function connectWifiMI8() {
  log("开始连接Wi-Fi");

  if (getNetworkState()) {
    log("当前wifi已链接");
    return;
  }

  launch("com.android.settings");
  sleep(500);
  NodeUtils.clickTextContains("WLAN");

  let isOpenWifi = false;
  while (!isOpenWifi) {
    const btn = className("android.widget.CheckBox").findOne();

    isOpenWifi = btn.checked();
    if (!isOpenWifi) {
      log("打开Wi-Fi");
      const rect = btn.bounds();
      if (click(rect.centerX(), rect.centerY())) {
        sleep(1000);
      } else {
        toastLog("开启wifi失败");
        throw new Error("开启wifi失败");
      }
    } else {
      toastLog("打开wifi成功");
    }
  }

  if (getNetworkState()) {
    log("当前wifi已链接");
    return;
  }

  log("开始向下滑动");
  while (!NodeUtils.waitNodeMatchExists("text", "添加网络", 500)) {
    GestureUtils.swipeToBottom();
    sleep(1000);
  }

  log("开始添加网络");
  NodeUtils.clickTextContains("添加网络");
  sleep(1000);

  id("ssid").findOne().setText(wifiName);
  sleep(500);
  // 点击安全性
  click(620, 788);
  sleep(500);
  NodeUtils.clickTextContains("WPA/WPA2-Personal");
  sleep(500);

  id("password").findOne().setText(pwd);

  sleep(500);

  // 点击确定
  click(970, 166);
  sleep(2000);

  connectWifiMI8();
}

function getNetworkState() {
  importClass(android.net.ConnectivityManager);
  var cm = context.getSystemService(context.CONNECTIVITY_SERVICE);
  var net = cm.getActiveNetworkInfo();

  if (isNetworkUsable(net)) {
    toastLog("网络连接可用!");
    return true;
  }

  var wifiNet = cm.getNetworkInfo(ConnectivityManager.TYPE_WIFI);
  if (isNetworkUsable(wifiNet)) {
    toastLog("Wi-Fi 网络连接可用!");
    return true;
  }

  var mobileNet = cm.getNetworkInfo(ConnectivityManager.TYPE_MOBILE);
  if (isNetworkUsable(mobileNet)) {
    toastLog("移动网络连接可用!");
    return true;
  }

  toastLog("网络连接不可用!");
  return false;
}

function isNetworkUsable(net) {
  return net != null && (net.isConnected() || net.isConnectedOrConnecting() || net.isAvailable());
}
