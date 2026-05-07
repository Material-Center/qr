const { NodeUtils } = require("../../common/node_util");
const { AppUtils } = require("../../common/app_util");
const { RegisterFailureAction } = require("./constants");
const {
  ensureAgreementChecked,
  isSecurityVerifyPage,
  isInputVerifyCodePage,
  isSendVerifyCodeManualPage,
  hasVerifyCodeNextStageFeature,
  isLoginSuccessPage,
} = require("./page_util");
const { createTodoError, createExceptionDecision } = require("./error");
const {
  getVerifyCodeStagePolicy,
  isVerifyCodeStagePassed,
} = require("./verify_code_flow");

const PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL = 1001;
const PHONE_REGISTER_STATUS_CODE_VERIFY_CODE_TIMEOUT = 1002;

function createStageFailure(message, statusCode) {
  return createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
    message: message,
    statusCode: statusCode,
    shouldReport: true,
    shouldReset: true,
  });
}

function clickTextButton(pattern, timeoutMs, failureMessage, clickLabel) {
  const button = textContains(pattern).findOne(timeoutMs);
  if (!button) {
    throw createStageFailure(
      failureMessage,
      PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
    );
  }
  if (!NodeUtils.clickUiObject(button, false)) {
    NodeUtils.clickByElement(button);
  }
  if (clickLabel) {
    log(clickLabel);
  }
}

function readVerifyCodeStageState(ctx) {
  return {
    isInputVerifyCodePage: isInputVerifyCodePage(100),
    isSendVerifyCodeManualPage: isSendVerifyCodeManualPage(100),
    hasNextStageFeature: hasVerifyCodeNextStageFeature(ctx, 100),
  };
}

function isVerifyCodeStagePassedNow(ctx) {
  return isVerifyCodeStagePassed(readVerifyCodeStageState(ctx));
}

function waitForVerifyCodeStagePassed(ctx, timeoutMs) {
  const maxWaitMs = Number(timeoutMs || 0) || 0;
  const startedAt = Date.now();

  while (Date.now() - startedAt < maxWaitMs) {
    if (isVerifyCodeStagePassedNow(ctx)) {
      return true;
    }
    sleep(Math.min(1000, Math.max(maxWaitMs - (Date.now() - startedAt), 50)));
  }

  return isVerifyCodeStagePassedNow(ctx);
}

function clickAgreeAndContinueUntilGone(ctx, timeoutMs) {
  const buttonText = "同意并继续";
  const maxWaitMs = Number(timeoutMs || 8000) || 8000;
  const startedAt = Date.now();
  let clickCount = 0;

  while (Date.now() - startedAt < maxWaitMs) {
    const agreeNode = text(buttonText).findOne(clickCount === 0 ? 1000 : 300);
    if (!agreeNode) {
      return {
        success: clickCount > 0,
        clicked: clickCount,
      };
    }

    if (!NodeUtils.clickUiObject(agreeNode, false)) {
      NodeUtils.clickByElement(agreeNode);
    }
    clickCount += 1;
    ctx.log("同意并继续按钮已点击，第" + clickCount + "次");

    if (NodeUtils.waitNodeGone("text", buttonText, 800)) {
      return {
        success: true,
        clicked: clickCount,
      };
    }
  }

  return {
    success: false,
    clicked: clickCount,
  };
}

function openOtherPhoneRegisterIfNeeded(ctx, timeoutMs) {
  const inputPhoneText = "输入手机号码";
  const otherPhoneText = "其他手机号注册";
  const maxWaitMs = Number(timeoutMs || 5000) || 5000;
  const startedAt = Date.now();
  let clickCount = 0;

  while (Date.now() - startedAt < maxWaitMs) {
    if (text(inputPhoneText).findOne(300)) {
      return true;
    }

    const otherPhoneRegisterNode = text(otherPhoneText).findOne(500);
    if (!otherPhoneRegisterNode) {
      sleep(200);
      continue;
    }

    clickCount += 1;
    ctx.log("其他手机号注册按钮已找到，准备点击，第" + clickCount + "次");
    const clicked =
      NodeUtils.clickUiObject(otherPhoneRegisterNode, false) ||
      NodeUtils.clickByElement(otherPhoneRegisterNode);
    if (!clicked) {
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "其他手机号注册按钮点击失败",
        shouldReport: true,
        shouldReset: true,
      });
    }

    if (NodeUtils.waitNodeExists("text", inputPhoneText, 1500)) {
      ctx.log("已进入手机号输入页面");
      return true;
    }
  }

  return text(inputPhoneText).findOne(300) != null;
}

function ensureManualVerifyCodePage(ctx) {
  if (isVerifyCodeStagePassedNow(ctx)) {
    return true;
  }
  if (isSendVerifyCodeManualPage(500)) {
    return true;
  }
  if (isInputVerifyCodePage(1000)) {
    clickTextButton("收不到短信验证码", 1000, "未找到收不到验证码按钮");
    ctx.log("已点击收不到验证码按钮");
  }
  if (!isSendVerifyCodeManualPage(2000)) {
    throw createStageFailure(
      "没有触发我已发送验证码页面",
      PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
    );
  }
  return true;
}

function clearAndInputVerifyCode(verifyCode) {
  const editTexts = className("android.widget.EditText").find();
  if (!editTexts || editTexts.length === 0) {
    throw createStageFailure(
      "未找到验证码输入框",
      PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
    );
  }
  const firstInput = editTexts.get(0);
  if (!NodeUtils.clickUiObject(firstInput, false)) {
    NodeUtils.clickByElement(firstInput);
  }
  for (let i = 0; i < editTexts.length; i++) {
    const input = editTexts.get(i);
    if (input && typeof input.setText === "function") {
      input.setText("");
    }
  }
  sleep(300);
  NodeUtils.inputNumber(String(verifyCode || ""));
}

function clickResendVerifyCode(ctx) {
  clickTextButton("重新发送", 2000, "未找到重新发送按钮");
  ctx.log("重新发送按钮已点击");
}

function handlePlatformSendVerifyCode(ctx, policy) {
  if (!isInputVerifyCodePage(2000)) {
    throw createStageFailure(
      "没有触发输入验证码页面",
      PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
    );
  }

  ctx.report("enter_waiting_code", "已进入验证码等待阶段");
  ctx.log("输入验证码页面已进入，开始等待地推提交验证码");

  let lastSubmittedCode = "";
  for (
    let roundIndex = 0;
    roundIndex < policy.waitRounds.length;
    roundIndex++
  ) {
    const roundWaitMs = policy.waitRounds[roundIndex];
    const roundDeadlineAt = Date.now() + roundWaitMs;
    ctx.log(
      "验证码等待第 " +
        (roundIndex + 1) +
        " 轮，最长等待 " +
        roundWaitMs +
        "ms",
    );

    while (Date.now() < roundDeadlineAt) {
      if (isVerifyCodeStagePassedNow(ctx)) {
        ctx.report("consume_code_success", "验证码阶段已通过");
        return;
      }

      const code = ctx.waitForPendingVerifyCode(
        Math.min(roundDeadlineAt - Date.now(), 3000),
        {
          pollIntervalMs: 1000,
          shouldStop: function () {
            return isVerifyCodeStagePassedNow(ctx);
          },
        },
      );

      if (!code) {
        continue;
      }
      if (code === lastSubmittedCode) {
        continue;
      }

      ctx.log("检测到新验证码，准备输入");
      clearAndInputVerifyCode(code);
      lastSubmittedCode = code;

      if (
        waitForVerifyCodeStagePassed(
          ctx,
          Math.min(Math.max(roundDeadlineAt - Date.now(), 0), 15000),
        )
      ) {
        ctx.report("consume_code_success", "验证码已消费并进入下一阶段");
        return;
      }

      ctx.log("当前验证码未推进到下一阶段，继续等待新的验证码");
    }

    if (roundIndex < policy.resendCount) {
      clickResendVerifyCode(ctx);
    }
  }

  throw createStageFailure(
    "验证码等待超时",
    PHONE_REGISTER_STATUS_CODE_VERIFY_CODE_TIMEOUT,
  );
}

function handleUserSentToTXVerifyCode(ctx, policy) {
  ctx.log("用户已发送验证码模式，准备点击我已发送");
  const waitSeconds = Math.floor(policy.manualSubmitIntervalMs / 1000);
  for (let attempt = 1; attempt <= policy.manualSubmitMaxAttempts; attempt++) {
    if (isVerifyCodeStagePassedNow(ctx)) {
      return;
    }

    ensureManualVerifyCodePage(ctx);
    clickTextButton("我已发送", 2000, "未找到我已发送按钮");
    ctx.log(
      "我已发送按钮已点击，attempt=" + attempt + "，等待" + waitSeconds + "秒",
    );

    if (waitForVerifyCodeStagePassed(ctx, policy.manualSubmitIntervalMs)) {
      return;
    }
  }

  throw createStageFailure("未发", PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL);
}

const RegisterUIActions = {
  handleAuthorizeDialog(ctx) {
    if (!NodeUtils.waitNodeExists("text", "用户协议及隐私政策概要", 1000)) {
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "未找到用户隐私弹窗",
        shouldReport: true,
        shouldReset: false,
      });
    }
    if (!NodeUtils.clickBySelector("id", "dialogRightBtn")) {
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "同意按钮点击失败",
        shouldReport: true,
        shouldReset: false,
      });
    }
    if (!NodeUtils.waitNodeExists("id", "btn_register", 2000)) {
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "注册按钮未找到",
        shouldReport: true,
        shouldReset: false,
      });
    }
    ctx.log("用户隐私弹窗已同意");
  },

  openRegisterPage(ctx) {
    ctx.log("准备打开 QQ 注册页面");
    if (!NodeUtils.waitNodeExists("id", "btn_register", 2000)) {
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "注册按钮未找到",
        shouldReport: true,
        shouldReset: true,
      });
    }

    if (!NodeUtils.clickBySelector("id", "btn_register")) {
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "注册按钮点击失败",
        shouldReport: true,
        shouldReset: false,
      });
    }

    ensureAgreementChecked(ctx, 3000);
    if (!openOtherPhoneRegisterIfNeeded(ctx, 5000)) {
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "未进入手机号输入页面",
        shouldReport: true,
        shouldReset: true,
      });
    }
  },

  inputPhone(ctx) {
    ctx.log("准备输入手机号: " + ctx.getTaskPhone());

    const phoneNode = text("输入手机号码").findOne(1000);
    if (!phoneNode) {
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "未找到输入手机号码控件",
        shouldReport: true,
        shouldReset: true,
      });
    }
    if (!NodeUtils.clickUiObject(phoneNode, false)) {
      NodeUtils.clickByElement(phoneNode);
    }
    sleep(500);
    NodeUtils.inputNumber(ctx.getTaskPhone(), 100);
    sleep(500);

    // 点一下键盘失焦
    click(device.width / 2, 100);

    ctx.log("手机号输入完成");

    const agreeResult = clickAgreeAndContinueUntilGone(ctx, 8000);
    if (!agreeResult.clicked) {
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "未找到同意并继续按钮",
        shouldReport: true,
        shouldReset: true,
      });
    }
    if (!agreeResult.success) {
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "同意并继续按钮点击后未消失",
        shouldReport: true,
        shouldReset: true,
      });
    }

    // 有些手机号是注册失败的，需要处理
    if (NodeUtils.waitNodeMatchExists("text", "注册失败", 1000)) {
      if (NodeUtils.waitNodeMatchExists("text", "更换手机号码后重试", 1000)) {
        throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
          message: "注册失败，更换手机号码后重试",
          shouldReport: true,
          shouldReset: false,
        });
      }
      throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
        message: "注册失败",
        shouldReport: true,
        shouldReset: true,
      });
    }
  },

  securityVerify(ctx) {
    ctx.log("准备处理安全验证");

    if (!isSecurityVerifyPage()) {
      ctx.log("当前页面未出现安全验证，跳过该阶段");
      return;
    }

    let tryCount = 0;

    while (!isInputVerifyCodePage() && !isSendVerifyCodeManualPage()) {
      if (tryCount > 3) {
        throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
          message: "安全验证处理失败",
          shouldReport: true,
          shouldReset: true,
        });
      }

      tryCount++;

      const challenge = ctx.solveImageVerification("框出正确位置", {});
      ctx.log("安全验证截图已生成 path=" + challenge.debugPath);
      challenge.click();

      const confirmNode = textMatches("确定").findOne(1000);
      if (!confirmNode) {
        throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
          message: "未找到确定按钮",
          shouldReport: true,
          shouldReset: true,
        });
      }
      if (!NodeUtils.clickUiObject(confirmNode, false)) {
        NodeUtils.clickByElement(confirmNode);
      }

      ctx.log("确定按钮已点击");
      sleep(3000);
    }

    ctx.log("安全验证处理完成");
  },

  waitOrSubmitVerifyCode(ctx) {
    const mode = ctx.getSmsReceiveMode();
    ctx.log("准备处理验证码，模式: " + mode);
    const policy = getVerifyCodeStagePolicy(mode);
    if (!policy) {
      throw createStageFailure(
        "未找到验证码模式",
        PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
      );
    }
    if (mode === "PLATFORM_SEND") {
      handlePlatformSendVerifyCode(ctx, policy);
      return;
    }
    if (mode === "USER_SENT_TO_TX") {
      handleUserSentToTXVerifyCode(ctx, policy);
      return;
    }
    throw createStageFailure(
      "不支持的验证码模式",
      PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
    );
  },

  completeProfile(ctx) {
    ctx.log("准备填写昵称、用户名等资料");
    if (!hasVerifyCodeNextStageFeature(ctx)) {
      throw createStageFailure(
        "没有触发设置账号信息页面",
        PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
      );
    }

    // 先清空数据，再输入数据
    descContains("清空")
      .find()
      .forEach((item) => {
        if (!NodeUtils.clickUiObject(item, false)) {
          NodeUtils.clickByElement(item);
        }
      });

    const editTexts = className("android.widget.EditText").find();
    if (editTexts && editTexts.size() < 2) {
      throw createStageFailure(
        "未找到昵称、用户名输入框",
        PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
      );
    }

    const nicknameEdit = editTexts.get(0);
    if (!NodeUtils.clickUiObject(nicknameEdit, false)) {
      NodeUtils.clickByElement(nicknameEdit);
    }
    NodeUtils.inputText(ctx.getTaskNickname());

    const usernameEdit = editTexts.get(1);
    if (!NodeUtils.clickUiObject(usernameEdit, false)) {
      NodeUtils.clickByElement(usernameEdit);
    }
    NodeUtils.inputText(ctx.getTaskUsername());
    sleep(500);

    const registerButton = text("注册并登录").findOne(1000);
    if (!registerButton) {
      throw createStageFailure(
        "未找到注册并登录按钮",
        PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
      );
    }
    if (!NodeUtils.clickUiObject(registerButton, false)) {
      NodeUtils.clickByElement(registerButton);
    }
    sleep(500);

    if (!isLoginSuccessPage()) {
      throw createStageFailure(
        "未跳转到登录成功页面",
        PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
      );
    }
    ctx.log("注册并登录按钮已点击");
  },

  waitLoginSuccess(ctx) {
    if (!isLoginSuccessPage()) {
      throw createStageFailure(
        "未跳转到登录成功页面",
        PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
      );
    }
    ctx.reportRegisterSuccessIfNeeded("注册成功，等待上传缓存");
    ctx.log("登录成功页面已进入");
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

  handleOpenRegisterPageException(ctx, exceptionState) {
    ctx.log(
      "注册页面异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message,
    );
    return null;
  },

  handleAuthorizeDialogException(ctx, exceptionState) {
    ctx.log(
      "待实现授权弹窗异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message,
    );
    return null;
  },

  handleInputPhoneException(ctx, exceptionState) {
    ctx.log(
      "待实现手机号输入异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message,
    );
    return null;
  },

  handleSecurityVerifyException(ctx, exceptionState) {
    ctx.log(
      "待实现安全验证异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message,
    );
    return null;
  },

  handleVerifyCodeException(ctx, exceptionState) {
    ctx.log(
      "待实现验证码异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message,
    );
    return null;
  },

  handleCompleteProfileException(ctx, exceptionState) {
    ctx.log(
      "待实现资料填写异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message,
    );
    return null;
  },

  handleWaitLoginSuccessException(ctx, exceptionState) {
    ctx.log(
      "待实现登录完成异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message,
    );
    return null;
  },

  handleSubmitCacheException(ctx, exceptionState) {
    ctx.log(
      "待实现缓存提交异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message,
    );
    return null;
  },

  handleResetEnvironmentException(ctx, exceptionState) {
    ctx.log(
      "待实现环境重置异常处理 type=" +
        exceptionState.type +
        " message=" +
        exceptionState.message,
    );
    return null;
  },
};

module.exports = {
  RegisterUIActions,
};
