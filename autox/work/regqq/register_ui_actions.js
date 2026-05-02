const { AppUtils } = require("../../common/app_util");
const { RegisterFailureAction } = require("./register_constants");
const {
  createTodoError,
  createExceptionDecision,
} = require("./register_error");

const RegisterUIActions = {
  openLoginPage(ctx) {
    ctx.log("准备打开 QQ 登录页");
    AppUtils.waitPkgLaunched(ctx.config.qqPackageName);
    throw createTodoError(
      "openLoginPage",
      "实现进入登录页、处理冷启动和首页分流逻辑"
    );
  },

  handleAuthorizeDialog(ctx) {
    ctx.log("准备处理授权弹窗");
    throw createTodoError(
      "handleAuthorizeDialog",
      "实现授权/协议弹窗识别与点击同意逻辑"
    );
  },

  inputPhone(ctx) {
    ctx.log("准备输入手机号: " + ctx.getTaskPhone());
    throw createTodoError(
      "inputPhone",
      "实现手机号输入、下一步提交、异常提示处理"
    );
  },

  waitOrSubmitVerifyCode(ctx) {
    ctx.log("准备处理验证码，模式: " + ctx.getSmsReceiveMode());
    throw createTodoError(
      "waitOrSubmitVerifyCode",
      "实现等待验证码 / 我已发送验证码 / 二次提交重试逻辑"
    );
  },

  completeProfile(ctx) {
    ctx.log("准备填写昵称、用户名等资料");
    // 示例：
    // const challenge = ctx.solveImageVerification("框出正确位置", {
    //   region: { x: 0, y: 300, width: device.width, height: 900 },
    // });
    // ctx.log("图片验证截图: " + challenge.debugPath);
    // challenge.click();
    throw createTodoError(
      "completeProfile",
      "实现昵称用户名生成、表单填写、校验失败重试逻辑；若出现图片验证页可调用 ctx.solveImageVerification()"
    );
  },

  waitLoginSuccess(ctx) {
    ctx.log("准备等待登录完成");
    throw createTodoError(
      "waitLoginSuccess",
      "实现登录成功页识别、首页落地判断、异常分支处理"
    );
  },

  handleCommonException(ctx, exceptionState) {
    // 在这里统一处理全局异常弹窗、风控页、网络异常页、崩溃重启等。
    // 可返回：
    // 1. createExceptionDecision(RegisterFailureAction.RETRY_STAGE, {...})
    // 2. createExceptionDecision(RegisterFailureAction.CONTINUE, {...})
    // 3. createExceptionDecision(RegisterFailureAction.RESET_AND_FAIL, {...})
    if (exceptionState.isTodo) {
      return createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: exceptionState.message,
        shouldReport: false,
        shouldReset: false,
      });
    }
    return null;
  },

  handleOpenLoginPageException(ctx, exceptionState) {
    ctx.log(
      "待实现登录页异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message
    );
    return null;
  },

  handleAuthorizeDialogException(ctx, exceptionState) {
    ctx.log(
      "待实现授权弹窗异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message
    );
    return null;
  },

  handleInputPhoneException(ctx, exceptionState) {
    ctx.log(
      "待实现手机号输入异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message
    );
    return null;
  },

  handleVerifyCodeException(ctx, exceptionState) {
    ctx.log(
      "待实现验证码异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message
    );
    return null;
  },

  handleCompleteProfileException(ctx, exceptionState) {
    ctx.log(
      "待实现资料填写异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message
    );
    return null;
  },

  handleWaitLoginSuccessException(ctx, exceptionState) {
    ctx.log(
      "待实现登录完成异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message
    );
    return null;
  },

  handleSubmitCacheException(ctx, exceptionState) {
    ctx.log(
      "待实现缓存提交异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message
    );
    return null;
  },

  handleResetEnvironmentException(ctx, exceptionState) {
    ctx.log(
      "待实现环境重置异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message
    );
    return null;
  },
};

module.exports = {
  RegisterUIActions,
};
