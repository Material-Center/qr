const { RegisterContext } = require("./register_context");
const { executeRegisterFlow } = require("./register_flow");
const { handleFlowFailure } = require("./register_exception_flow");

function RegisterRunner(config) {
  this.config = config || {};
  this.ctx = new RegisterContext(config);
}

RegisterRunner.prototype.runOnce = function () {
  const ctx = this.ctx;
  ctx.log("开始轮询手机号注册任务");

  const task = ctx.pollTask();
  if (!task || !task.id) {
    ctx.log("当前没有待执行任务");
    return {
      ok: true,
      idle: true,
    };
  }

  ctx.log(
    "领取到任务 id=" +
      task.id +
      " phone=" +
      ctx.getTaskPhone() +
      " smsMode=" +
      ctx.getSmsReceiveMode()
  );

  try {
    ctx.heartbeat();
    executeRegisterFlow(ctx);
    ctx.log("任务流程执行完成 id=" + task.id);
    return {
      ok: true,
      taskId: task.id,
    };
  } catch (err) {
    const normalized = handleFlowFailure(ctx, err);
    ctx.log("任务流程失败: " + normalized.message);
    throw normalized;
  }
};

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
