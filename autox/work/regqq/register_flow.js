const { RegisterAction } = require("./register_constants");
const { RegisterUIActions } = require("./register_ui_actions");
const { runStageWithExceptionHandling } = require("./register_exception_flow");

function executeRegisterFlow(ctx) {
  ctx.ensureTask();

  runStageWithExceptionHandling(ctx, RegisterAction.OPEN_LOGIN_PAGE, function () {
    RegisterUIActions.openLoginPage(ctx);
  });

  runStageWithExceptionHandling(ctx, RegisterAction.HANDLE_AUTHORIZE_DIALOG, function () {
    RegisterUIActions.handleAuthorizeDialog(ctx);
  });

  runStageWithExceptionHandling(ctx, RegisterAction.INPUT_PHONE, function () {
    RegisterUIActions.inputPhone(ctx);
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
