const { RegisterContext } = require("./context");
const { executeRegisterFlow } = require("./flow");
const { handleFlowFailure } = require("./exception_flow");

function RegisterRunner(config) {
  this.config = config || {};
  this.ctx = new RegisterContext(config);
}

RegisterRunner.prototype.runOnce = function () {
  const ctx = this.ctx;
  ctx.ensureStartupGuardReady();
  ctx.log("开始轮询手机号注册任务");

  const task = ctx.pollTask();
  const taskId = ctx.getTaskId();
  if (!task || !taskId) {
    ctx.log("当前没有待执行任务");
    return {
      ok: true,
      idle: true,
    };
  }

  ctx.log(
    "领取到任务 id=" +
      taskId +
      " phone=" +
      ctx.getTaskPhone() +
      " smsMode=" +
      ctx.getSmsReceiveMode()
  );

  try {
    ctx.refreshTaskRuntimeConfig();
    ctx.prepareQQProfileDraft(false);
    ctx.heartbeat();
    ctx.startTaskHeartbeatLoop();
    executeRegisterFlow(ctx);
    ctx.log("任务流程执行完成 id=" + taskId);
    return {
      ok: true,
      taskId: taskId,
    };
  } catch (err) {
    const normalized = handleFlowFailure(ctx, err);
    ctx.log("任务流程失败: " + normalized.message);
    throw normalized;
  } finally {
    ctx.stopTaskHeartbeatLoop();
    openCurrentAutoXApp();
  }
};

function openCurrentAutoXApp() {
  const packageName =
    typeof context !== "undefined" &&
    context &&
    typeof context.getPackageName === "function"
      ? context.getPackageName()
      : "";
  if (!packageName) {
    console.warn("未获取到当前 AutoX 包名，无法打开应用");
    return false;
  }
  console.log("runOnce 结束，打开当前 AutoX app: " + packageName);
  return launch(packageName);
}

RegisterRunner.prototype.runForever = function () {
  const ctx = this.ctx;
  const pollIntervalMs =
    Number(this.config.taskPollIntervalMs || 5000) || 5000;
  const errorRetryIntervalMs =
    Number(this.config.taskErrorRetryIntervalMs || pollIntervalMs) ||
    pollIntervalMs;

  ctx.log(
    "启动自动轮询 worker pollIntervalMs=" +
      pollIntervalMs +
      " errorRetryIntervalMs=" +
      errorRetryIntervalMs
  );

  // eslint-disable-next-line no-constant-condition
  while (true) {
    try {
      const result = this.runOnce();
      if (result && result.idle) {
        sleep(pollIntervalMs);
        continue;
      }
      sleep(1000);
    } catch (err) {
      const message = err && err.message ? err.message : String(err);
      ctx.log("本轮执行异常，等待后重试: " + message);
      sleep(errorRetryIntervalMs);
    }
  }
};

module.exports = {
  RegisterRunner,
};
