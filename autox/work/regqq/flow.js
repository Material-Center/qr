const { RegisterAction } = require("./constants");
const { RegisterUIActions } = require("./ui_actions");
const { runStageWithExceptionHandling } = require("./exception_flow");

function executeRegisterFlow(ctx) {
  ctx.ensureTask();
  ctx.ensureQQReady();

  runStageWithExceptionHandling(ctx, RegisterAction.HANDLE_AUTHORIZE_DIALOG, function () {
    RegisterUIActions.handleAuthorizeDialog(ctx);
  });

  runStageWithExceptionHandling(ctx, RegisterAction.OPEN_REGISTER_PAGE, function () {
    RegisterUIActions.openRegisterPage(ctx);
  });

  runStageWithExceptionHandling(ctx, RegisterAction.INPUT_PHONE, function () {
    RegisterUIActions.inputPhone(ctx);
  });

  runStageWithExceptionHandling(ctx, RegisterAction.SECURITY_VERIFY, function () {
    RegisterUIActions.securityVerify(ctx);
  });

  runStageWithExceptionHandling(ctx, RegisterAction.WAIT_OR_SUBMIT_VERIFY_CODE, function () {
    RegisterUIActions.waitOrSubmitVerifyCode(ctx);
  });

  runStageWithExceptionHandling(ctx, RegisterAction.COMPLETE_PROFILE, function () {
    RegisterUIActions.completeProfile(ctx);
  });

  runStageWithExceptionHandling(ctx, RegisterAction.WAIT_LOGIN_SUCCESS, function () {
    RegisterUIActions.waitLoginSuccess(ctx);
  });

  runStageWithExceptionHandling(ctx, RegisterAction.SUBMIT_CACHE, function () {
    ctx.reportRegisterSuccessIfNeeded("注册成功，等待上传缓存");
    ctx.uploadCurrentCache("");
  });

  if (ctx.config.resetAfterSuccess) {
    runStageWithExceptionHandling(ctx, RegisterAction.RESET_ENVIRONMENT, function () {
      ctx.resetEnvironment();
    });
  }
}

module.exports = {
  executeRegisterFlow,
};
