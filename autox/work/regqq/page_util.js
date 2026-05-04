const { RegisterFailureAction } = require("./constants");
const { createExceptionDecision } = require("./error");
const { NodeUtils } = require("../../common/node_util");

const DEFAULT_VERIFY_CODE_NEXT_STAGE_SPECS = [
  { kind: "text", value: "设置账号信息", match: "contains" },
  { kind: "desc", value: "设置头像", match: "contains" },
  { kind: "text", value: "注册并登录", match: "contains" },
];

function getVerifyCodeNextStageSpecs(ctx) {
  const config = (ctx && ctx.config && ctx.config.verifyCodeStage) || {};
  if (Array.isArray(config.nextStageSelectors) && config.nextStageSelectors.length) {
    return config.nextStageSelectors;
  }
  return DEFAULT_VERIFY_CODE_NEXT_STAGE_SPECS;
}

function ensureAgreementChecked(ctx, timeoutMs) {
  const waitMs = Number(timeoutMs || 2000) || 2000;
  let tryCount = 0;

  // eslint-disable-next-line no-constant-condition
  while (true) {
    click(device.width / 2, 100);
    tryCount += 1;

    const agreeNode = desc("同意协议").findOne(waitMs);
    if (!agreeNode) {
      if (tryCount > 3) {
        throw createExceptionDecision(RegisterFailureAction.FAIL_FLOW, {
          message: "未找到同意协议控件",
          shouldReport: true,
          shouldReset: true,
        });
      }
      continue;
    }

    if (agreeNode.checked()) {
      ctx.log("同意协议已勾选");
      return true;
    }

    agreeNode.click();
    sleep(300);
  }
}

function isSecurityVerifyPage(timeoutMs) {
  const result = NodeUtils.waitPageChanged({
    newPage: [
      { kind: "text", value: "安全验证", match: "contains" },
      { kind: "text", value: "确定", match: "contains" },
      { kind: "text", value: "换一组", match: "contains" },
    ],
    newPageMode: "all",
    timeoutMs: Number(timeoutMs || 3000) || 3000,
    intervalMs: 50,
  });
  return result.changed && result.reason === "new_page_ready";
}

function isInputVerifyCodePage(timeoutMs) {
  const result = NodeUtils.waitPageChanged({
    newPage: [
      { kind: "text", value: "输入短信验证码", match: "contains" },
      { kind: "desc", value: "第1位", match: "contains" },
      { kind: "desc", value: "第6位", match: "contains" },
      { kind: "text", value: "收不到短信验证码", match: "contains" },
    ],
    newPageMode: "all",
    timeoutMs: Number(timeoutMs || 500) || 500,
    intervalMs: 50,
  });
  return result.changed && result.reason === "new_page_ready";
}

function isSendVerifyCodeManualPage(timeoutMs) {
  const result = NodeUtils.waitPageChanged({
    newPage: [
      { kind: "text", value: "去发送短信", match: "contains" },
      { kind: "text", value: "我已发送", match: "contains" },
      { kind: "text", value: "短信验证", match: "contains" },
    ],
    newPageMode: "all",
    timeoutMs: Number(timeoutMs || 1000) || 1000,
    intervalMs: 50,
  });
  return result.changed && result.reason === "new_page_ready";
}

function hasVerifyCodeNextStageFeature(ctx, timeoutMs) {
  const specs = getVerifyCodeNextStageSpecs(ctx);
  if (!specs.length) {
    return false;
  }
  const result = NodeUtils.waitPageChanged({
    newPage: specs,
    newPageMode: "any",
    timeoutMs: Number(timeoutMs || 1000) || 1000,
    intervalMs: 50,
  });
  return result.changed && result.reason === "new_page_ready";
}

function isLoginSuccessPage(timeoutMs) {
  const result = NodeUtils.waitPageChanged({
    newPage: [
      { kind: "text", value: "消息", match: "contains" },
      { kind: "text", value: "联系人", match: "contains" }
    ],
    newPageMode: "all",
    timeoutMs: Number(timeoutMs || 1000) || 1000,
    intervalMs: 50,
  });
  return result.changed && result.reason === "new_page_ready";
}

module.exports = {
  ensureAgreementChecked,
  isSecurityVerifyPage,
  isInputVerifyCodePage,
  isSendVerifyCodeManualPage,
  hasVerifyCodeNextStageFeature,
  isLoginSuccessPage,
};
