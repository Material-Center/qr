const { NodeUtils } = require("../../common/node_util");
const { RegisterFailureAction } = require("./constants");
const {
  ensureAgreementChecked,
  isSecurityVerifyPage,
  isInputVerifyCodePage,
  isSendVerifyCodeManualPage,
  hasVerifyCodeNextStageFeature,
  isLoginSuccessPage,
} = require("./page_util");
const { createExceptionDecision } = require("./error");
const {
  getVerifyCodeStagePolicy,
  isVerifyCodeStagePassed,
} = require("./verify_code_flow");

const PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL = 1001;
const PHONE_REGISTER_STATUS_CODE_VERIFY_CODE_TIMEOUT = 1002;
const CLICK_AFTER_DELAY_MS = 1500;

function createStageFailure(message, statusCode) {
  return createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
    message: message,
    statusCode: statusCode,
    shouldReport: true,
    shouldReset: true,
  });
}

function clickMatchedButton(selector, timeoutMs, failureMessage, clickLabel) {
  const match = NodeUtils.findClickableMatch(selector, timeoutMs);
  if (!match) {
    throw createStageFailure(
      failureMessage,
      PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
    );
  }
  const target = match.node || match.fallbackNode;
  if (!NodeUtils.clickUiObject(target, false)) {
    NodeUtils.clickByElement(match.fallbackNode || target);
  }
  if (clickLabel) {
    log(clickLabel);
  }
  return true;
}

function clickTextButton(pattern, timeoutMs, failureMessage, clickLabel) {
  clickMatchedButton(text(pattern), timeoutMs, failureMessage, clickLabel);
}

function clickTextContainsButton(
  pattern,
  timeoutMs,
  failureMessage,
  clickLabel,
) {
  clickMatchedButton(
    textContains(pattern),
    timeoutMs,
    failureMessage,
    clickLabel,
  );
}

function clickRegisterAndLogin(ctx) {
  const maxClickCount = 3;
  let hasButton = false;

  for (let i = 0; i < maxClickCount; i++) {
    try {
      clickTextContainsButton("注册并登录", 2000, "未找到注册并登录按钮");
      hasButton = true;
    } catch (e) {
      sleep(500);
      continue;
    }

    sleep(CLICK_AFTER_DELAY_MS);

    if (isLoginSuccessPage(5000)) {
      ctx.log("已点击注册并登录");
      return;
    }

    if (!textContains("注册并登录").exists()) {
      ctx.log("已提交注册资料");
      return;
    }
  }

  if (!hasButton) {
    throw createStageFailure(
      "未找到注册并登录按钮",
      PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
    );
  }

  throw createStageFailure(
    "注册失败",
    PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
  );
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
    handleCheckPhoneBindLimit(ctx, 100);
    if (isVerifyCodeStagePassedNow(ctx)) {
      return true;
    }
    sleep(Math.min(300, Math.max(maxWaitMs - (Date.now() - startedAt), 50)));
  }

  handleCheckPhoneBindLimit(ctx, 100);
  return isVerifyCodeStagePassedNow(ctx);
}

function clickAgreeAndContinueUntilGone(ctx, timeoutMs) {
  const buttonText = "同意并继续";
  const maxWaitMs = Number(timeoutMs || 8000) || 8000;
  const startedAt = Date.now();
  let clickCount = 0;

  while (Date.now() - startedAt < maxWaitMs) {
    const agreeNode = textContains(buttonText).findOne(
      clickCount === 0 ? 1000 : 300,
    );
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
    sleep(CLICK_AFTER_DELAY_MS);

    if (NodeUtils.waitNodeGone("text", buttonText, 3000)) {
      return {
        success: true,
        clicked: clickCount,
      };
    }

    sleep(1000);
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
    clickTextContainsButton("收不到短信验证码", 1000, "未找到收不到验证码按钮");
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

function handlePlatformSendVerifyCode(ctx, policy) {
  if (!isInputVerifyCodePage(2000)) {
    throw createStageFailure(
      "没有触发输入验证码页面",
      PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
    );
  }

  ctx.report("enter_waiting_code", "已进入验证码等待阶段");
  ctx.log("验证码阶段: 等待地推提交验证码");

  let lastSubmittedCode = "";
  for (
    let roundIndex = 0;
    roundIndex < policy.waitRounds.length;
    roundIndex++
  ) {
    const roundWaitMs = policy.waitRounds[roundIndex];
    const roundDeadlineAt = Date.now() + roundWaitMs;
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

      ctx.log("验证码阶段: 收到验证码，准备输入");
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
    }

    if (roundIndex < policy.resendCount) {
      if (roundIndex > 0) {
        handleCheckPhoneBindLimit(ctx, 500);
      }
      clickTextButton("重新发送", 2000, "未找到重新发送按钮");
    }
  }

  throw createStageFailure(
    "验证码等待超时",
    PHONE_REGISTER_STATUS_CODE_VERIFY_CODE_TIMEOUT,
  );
}

function handleUserSentToTXVerifyCode(ctx, policy) {
  ctx.log("验证码阶段: 我已发码模式");
  const waitSeconds = Math.floor(policy.manualSubmitIntervalMs / 1000);
  for (let attempt = 1; attempt <= policy.manualSubmitMaxAttempts; attempt++) {
    if (isVerifyCodeStagePassedNow(ctx)) {
      return;
    }

    ensureManualVerifyCodePage(ctx);
    clickTextContainsButton("我已发送", 2000, "未找到我已发送按钮");
    ctx.log(
      "验证码阶段: 已点击我已发送 " +
        attempt +
        "/" +
        policy.manualSubmitMaxAttempts +
        "，等待" +
        waitSeconds +
        "秒",
    );
    sleep(CLICK_AFTER_DELAY_MS);

    if (waitForVerifyCodeStagePassed(ctx, policy.manualSubmitIntervalMs)) {
      return;
    }
  }

  throw createStageFailure("未发", PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL);
}

function handleCheckPhoneBindLimit(ctx, timeoutMs = 2000) {
  // 一个手机号只能绑定 5 个 QQ
  if (NodeUtils.waitNodeMatchExists("text", "手机号绑定名额已满", timeoutMs)) {
    ctx.log("手机号绑定名额已满");
    throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
      message: "手机号绑定名额已满",
      shouldReport: true,
      shouldReset: true,
    });
  }
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
    ctx.log("授权弹窗已处理");
  },

  openRegisterPage(ctx) {
    ctx.log("打开QQ注册页");
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
    ctx.log("输入手机号");

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

    const agreeResult = clickAgreeAndContinueUntilGone(ctx, 15 * 1000);
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
          shouldReset: true,
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
    if (!isSecurityVerifyPage()) {
      return;
    }
    ctx.log("处理安全验证");

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
      sleep(3000);
    }

    ctx.log("安全验证处理完成");
  },

  waitOrSubmitVerifyCode(ctx) {
    handleCheckPhoneBindLimit(ctx);
    const mode = ctx.getSmsReceiveMode();
    ctx.log("处理验证码 mode=" + mode);
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
    ctx.log("填写账号资料");
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
        ctx.log("清空数据: " + item.text());
      });

    const editTexts = className("android.widget.EditText").find();
    if (editTexts && editTexts.size() < 2) {
      throw createStageFailure(
        "未找到昵称、密码输入框",
        PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
      );
    }

    ctx.log("username: " + ctx.getTaskUsername());

    const nicknameEdit = editTexts.get(0);
    if (!NodeUtils.clickUiObject(nicknameEdit, false)) {
      NodeUtils.clickByElement(nicknameEdit);
    }
    nicknameEdit.setText(ctx.getTaskNickname());
    sleep(500);

    const passwordEdit = editTexts.get(1);
    if (!NodeUtils.clickUiObject(passwordEdit, false)) {
      NodeUtils.clickByElement(passwordEdit);
    }
    passwordEdit.setText(ctx.getQQPassword());
    sleep(500);

    // 点一下键盘失焦
    click(device.width / 2, 100);

    clickRegisterAndLogin(ctx);
  },

  waitLoginSuccess(ctx) {
    if (!isLoginSuccessPage(15000)) {
      throw createStageFailure(
        "未跳转到登录成功页面",
        PHONE_REGISTER_STATUS_CODE_DEVICE_EXEC_FAIL,
      );
    }
    ctx.reportRegisterSuccessIfNeeded("注册成功，等待上传缓存");
    ctx.log("登录成功");
  },

  handleCommonException(ctx, exceptionState) {
    handleCheckPhoneBindLimit(ctx);
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
    return null;
  },

  handleAuthorizeDialogException(ctx, exceptionState) {
    return null;
  },

  handleInputPhoneException(ctx, exceptionState) {
    return null;
  },

  handleSecurityVerifyException(ctx, exceptionState) {
    return null;
  },

  handleVerifyCodeException(ctx, exceptionState) {
    return null;
  },

  handleCompleteProfileException(ctx, exceptionState) {
    return null;
  },

  handleWaitLoginSuccessException(ctx, exceptionState) {
    return null;
  },

  handleSubmitCacheException(ctx, exceptionState) {
    return null;
  },

  handleResetEnvironmentException(ctx, exceptionState) {
    return null;
  },
};

module.exports = {
  RegisterUIActions,
};
