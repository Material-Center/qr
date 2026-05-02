const { RegQQConfig } = require("./regqq_config");
const { RegisterRunner } = require("./register_runner");

threads.start(function () {
  // eslint-disable-next-line no-constant-condition
  while (true) {
    const pkgName = currentPackage();
    if (
      pkgName === "com.miui.securitycenter" ||
      pkgName === "com.lbe.security.miui" ||
      pkgName === "com.android.systemui" ||
      pkgName === "com.lbe.security.miui" ||
      pkgName === "com.android.vpndialogs" ||
      pkgName === "miuix.appcompat.app.AlertDialog"
    ) {
      const btn = textMatches(
        /允许|同意|继续|始终允许|立即开始|仅在使用中允许|知道了/,
      ).findOne(1000);
      if (btn) {
        btn.click();
        console.log("弹窗已点击允许");
      }
    }

    const btn = textMatches(/暂不更新/).findOne(500);
    if (btn) {
      btn.click();
    }

    device.wakeUpIfNeeded();
  }
});

setTimeout(function () {
  main();
}, 1000);

function main() {
  const runner = new RegisterRunner(RegQQConfig);
  runner.runForever();
}
