/**
 * @file common.js
 * @description 可选入口：一次性导出各 util 命名空间，便于 `require("./common")` 解构。
 * @property {object} DeviceUtils 设备与网络，见 device_util.js。
 * @property {object} NodeUtils 无障碍 UI，见 node_util.js。
 * @property {object} GestureUtils 滑动手势，见 gesture_util.js。
 * @property {object} AppUtils 应用与 Activity，见 app_util.js。
 * @property {object} HttpUtils 通用 HTTP，见 http_util.js。
 * @property {object} PermissionUtils Android 权限，见 permission_util.js。
 *
 * 推荐在业务脚本中直接 `require("./node_util")` 等子模块，再使用 `NodeUtils.xxx()` 调用。
 */
function getBuildTime() {
  return "__AUTOX_BUILD_TIME__";
}

module.exports = {
  DeviceUtils: require("./device_util").DeviceUtils,
  NodeUtils: require("./node_util").NodeUtils,
  GestureUtils: require("./gesture_util").GestureUtils,
  AppUtils: require("./app_util").AppUtils,
  HttpUtils: require("./http_util").HttpUtils,
  PermissionUtils: require("./permission_util").PermissionUtils,
  getBuildTime,
};
