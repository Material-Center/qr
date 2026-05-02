const {
  RegisterAction,
  RegisterFailureAction,
} = require("./register_constants");
const {
  createExceptionDecision,
  normalizeRegisterError,
} = require("./register_error");
const { RegisterUIActions } = require("./register_ui_actions");

const STAGE_HANDLER_MAP = {};
STAGE_HANDLER_MAP[RegisterAction.OPEN_LOGIN_PAGE] =
  "handleOpenLoginPageException";
STAGE_HANDLER_MAP[RegisterAction.HANDLE_AUTHORIZE_DIALOG] =
  "handleAuthorizeDialogException";
STAGE_HANDLER_MAP[RegisterAction.INPUT_PHONE] = "handleInputPhoneException";
STAGE_HANDLER_MAP[RegisterAction.WAIT_OR_SUBMIT_VERIFY_CODE] =
  "handleVerifyCodeException";
STAGE_HANDLER_MAP[RegisterAction.COMPLETE_PROFILE] =
  "handleCompleteProfileException";
STAGE_HANDLER_MAP[RegisterAction.WAIT_LOGIN_SUCCESS] =
  "handleWaitLoginSuccessException";
STAGE_HANDLER_MAP[RegisterAction.SUBMIT_CACHE] = "handleSubmitCacheException";
STAGE_HANDLER_MAP[RegisterAction.RESET_ENVIRONMENT] =
  "handleResetEnvironmentException";

function resolveExceptionConfig(ctx) {
  const flowConfig = (ctx.config && ctx.config.exceptionFlow) || {};
  return {
    defaultRetryLimit: Number(flowConfig.defaultRetryLimit || 0) || 0,
    retryIntervalMs: Number(flowConfig.retryIntervalMs || 1000) || 1000,
    reportStageException: flowConfig.reportStageException !== false,
    resetOnFailure: flowConfig.resetOnFailure !== false,
    stageRetryLimits: flowConfig.stageRetryLimits || {},
  };
}

function resolveStageRetryLimit(ctx, stageAction) {
  const config = resolveExceptionConfig(ctx);
  const stageRetryLimits = config.stageRetryLimits || {};
  if (stageRetryLimits[stageAction] !== undefined) {
    return Number(stageRetryLimits[stageAction] || 0) || 0;
  }
  return config.defaultRetryLimit;
}

function resolveDefaultDecision(stageError) {
  return createExceptionDecision(
    stageError.retryable
      ? RegisterFailureAction.RETRY_STAGE
      : RegisterFailureAction.FAIL_FLOW,
    {
      message: stageError.message,
      statusCode: stageError.statusCode,
      shouldReport: stageError.shouldReport,
      shouldReset: stageError.shouldReset,
    }
  );
}

function mergeDecision(baseDecision, overrideDecision) {
  if (!overrideDecision) {
    return baseDecision;
  }
  const merged = {
    action: overrideDecision.action || baseDecision.action,
    message:
      overrideDecision.message !== undefined
        ? overrideDecision.message
        : baseDecision.message,
    statusCode:
      overrideDecision.statusCode !== undefined
        ? overrideDecision.statusCode
        : baseDecision.statusCode,
    shouldReport:
      overrideDecision.shouldReport !== undefined
        ? overrideDecision.shouldReport
        : baseDecision.shouldReport,
    shouldReset:
      overrideDecision.shouldReset !== undefined
        ? overrideDecision.shouldReset
        : baseDecision.shouldReset,
    retryDelayMs:
      overrideDecision.retryDelayMs !== undefined
        ? overrideDecision.retryDelayMs
        : baseDecision.retryDelayMs,
  };
  return merged;
}

function buildExceptionState(stageAction, stageError) {
  return {
    stageAction: stageAction,
    attempt: stageError.attempt || 1,
    type: stageError.type,
    message: stageError.message,
    statusCode: stageError.statusCode,
    retryable: stageError.retryable === true,
    shouldReport: stageError.shouldReport !== false,
    shouldReset: stageError.shouldReset !== false,
    isTodo: stageError.isTodo === true,
    details: stageError.details || null,
    error: stageError,
  };
}

function callHandler(handlerName, ctx, exceptionState) {
  if (!handlerName) {
    return null;
  }
  const handler = RegisterUIActions[handlerName];
  if (typeof handler !== "function") {
    return null;
  }
  return handler(ctx, exceptionState);
}

function resolveDecisionByHandlers(ctx, stageAction, stageError) {
  const exceptionState = buildExceptionState(stageAction, stageError);
  let decision = resolveDefaultDecision(stageError);

  decision = mergeDecision(
    decision,
    callHandler("handleCommonException", ctx, exceptionState)
  );
  decision = mergeDecision(
    decision,
    callHandler(STAGE_HANDLER_MAP[stageAction], ctx, exceptionState)
  );
  return {
    decision: decision,
    exceptionState: exceptionState,
  };
}

function safeReport(ctx, action, message, statusCode) {
  try {
    ctx.report(action, message, statusCode);
  } catch (err) {
    ctx.log(
      "异常流程上报失败 action=" +
        action +
        " message=" +
        (err && err.message ? err.message : String(err))
    );
  }
}

function safeReset(ctx, reason) {
  try {
    ctx.log("异常流程触发环境重置 reason=" + reason);
    ctx.resetEnvironment();
  } catch (err) {
    ctx.log(
      "异常流程环境重置失败: " +
        (err && err.message ? err.message : String(err))
    );
  }
}

function handleStageException(ctx, stageAction, stageError) {
  const config = resolveExceptionConfig(ctx);
  const resolved = resolveDecisionByHandlers(ctx, stageAction, stageError);
  const decision = resolved.decision;
  const exceptionState = resolved.exceptionState;

  ctx.log(
    "阶段异常 stage=" +
      stageAction +
      " type=" +
      exceptionState.type +
      " attempt=" +
      exceptionState.attempt +
      " message=" +
      exceptionState.message
  );

  if (config.reportStageException && decision.shouldReport !== false) {
    safeReport(
      ctx,
      RegisterAction.HANDLE_STAGE_EXCEPTION,
      stageAction + ":" + exceptionState.message,
      decision.statusCode
    );
  }

  if (
    decision.action === RegisterFailureAction.RESET_AND_FAIL &&
    config.resetOnFailure
  ) {
    safeReset(ctx, stageAction + ":" + exceptionState.type);
    stageError.failureResetApplied = true;
  }

  stageError.decision = decision;
  return decision;
}

function runStageWithExceptionHandling(ctx, stageAction, execute) {
  const config = resolveExceptionConfig(ctx);
  const retryLimit = resolveStageRetryLimit(ctx, stageAction);
  let attempt = 0;

  while (true) {
    attempt += 1;
    ctx.report(stageAction, "start");

    try {
      execute();
      ctx.report(stageAction, "done");
      return;
    } catch (err) {
      const stageError = normalizeRegisterError(stageAction, err, { attempt });
      const decision = handleStageException(ctx, stageAction, stageError);

      if (
        decision.action === RegisterFailureAction.RETRY_STAGE &&
        attempt <= retryLimit
      ) {
        const retryMessage =
          decision.message || "异常流程要求重试 stage=" + stageAction;
        safeReport(ctx, RegisterAction.STAGE_RETRY, retryMessage);
        sleep(decision.retryDelayMs || config.retryIntervalMs);
        continue;
      }

      if (decision.action === RegisterFailureAction.CONTINUE) {
        safeReport(
          ctx,
          RegisterAction.STAGE_SKIPPED,
          decision.message || "异常流程允许跳过 stage=" + stageAction,
          decision.statusCode
        );
        return;
      }

      throw stageError;
    }
  }
}

function handleFlowFailure(ctx, err) {
  const config = resolveExceptionConfig(ctx);
  const normalized = normalizeRegisterError("flow", err);
  normalized.decision = normalized.decision || resolveDefaultDecision(normalized);

  if (!normalized.isTodo && normalized.shouldReport !== false) {
    safeReport(
      ctx,
      RegisterAction.FLOW_FAILED,
      normalized.message,
      normalized.statusCode
    );
  }

  if (
    config.resetOnFailure &&
    normalized.shouldReset !== false &&
    !normalized.isTodo &&
    !normalized.failureResetApplied
  ) {
    safeReset(ctx, "flow_failed");
  }

  return normalized;
}

module.exports = {
  runStageWithExceptionHandling,
  handleFlowFailure,
};
