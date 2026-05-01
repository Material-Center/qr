const { DeviceUtils } = require("../../common/device_util");

const deviceId = DeviceUtils.getSerialNum();

setTimeout(() => {
  main();
}, 1000);

function main() {}
