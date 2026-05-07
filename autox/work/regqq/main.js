const { RegQQConfig } = require("./regqq_config");
const { RegisterRunner } = require("./runner");
const { RegisterUIActions } = require("./ui_actions");
const { PermissionUtils } = require("../../common/permission_util");

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
  auto.waitFor();

  if (!PermissionUtils.ensureExternalStoragePermission({ timeoutMs: 15000 })) {
    throw new Error("外部文件读写权限未授权");
  }

  const runner = new RegisterRunner(RegQQConfig);

  try {
    runEntry(runner);
  } catch (error) {
    console.error(error);
  }
}

function runEntry(runner) {
  const devConfig = (RegQQConfig && RegQQConfig.dev) || {};
  const entry = String(devConfig.entry || "worker")
    .trim()
    .toLowerCase();

  if (entry === "worker") {
    runner.runForever();
    return;
  }

  if (entry === "once") {
    runner.runOnce();
    return;
  }

  if (entry === "upload_cache") {
    bindMockTaskIfNeeded(runner, devConfig);
    runner.ctx.uploadCurrentCache(String(devConfig.uploadCacheQQPwd || ""));
    return;
  }

  if (entry === "image_verify") {
    runner.ctx.solveImageVerification(devConfig.imageVerifyQuestion, {});
    return;
  }

  if (entry === "custom") {
    runCustomDevEntry(runner, devConfig);
    return;
  }

  throw new Error("unknown regqq dev entry: " + entry);
}

function bindMockTaskIfNeeded(runner, devConfig) {
  if (runner.ctx.getTaskId()) {
    return;
  }
  const mockTask = devConfig && devConfig.mockTask;
  if (mockTask && (mockTask.id || mockTask.taskId)) {
    runner.ctx.bindTask(mockTask);
  }
}

function runCustomDevEntry(runner, devConfig) {
  bindMockTaskIfNeeded(runner, devConfig);

  // const nodeDebugger = new NodeDebugger();
  // nodeDebugger.dumpNodeTree(4);
  runner.ctx.refreshTaskRuntimeConfig();
  runner.ctx.prepareQQProfileDraft(false);

  runner.ctx.ensureQQReady();
  RegisterUIActions.handleAuthorizeDialog(runner.ctx);
  RegisterUIActions.openRegisterPage(runner.ctx);
  RegisterUIActions.inputPhone(runner.ctx);
  RegisterUIActions.securityVerify(runner.ctx);
  RegisterUIActions.waitLoginSuccess(runner.ctx);
}
