const {
  RegisterExceptionType,
  RegisterFailureAction,
} = require("./constants");

function createRegisterError(type, message, options) {
  const err = new Error(message || "register flow error");
  const extra = options || {};
  err.name = "RegisterError";
  err.isRegisterError = true;
  err.type = type || RegisterExceptionType.UNKNOWN;
  err.statusCode = extra.statusCode;
  err.retryable = extra.retryable === true;
  err.shouldReport = extra.shouldReport !== false;
  err.shouldReset = extra.shouldReset !== false;
  err.isTodo = extra.isTodo === true;
  err.stageAction = extra.stageAction || "";
  err.details = extra.details || null;
  err.cause = extra.cause;
  return err;
}

function createTodoError(action, message, options) {
  const extra = options || {};
  return createRegisterError(
    extra.type || RegisterExceptionType.TODO,
    "[TODO][" + action + "] " + message,
    {
      stageAction: action,
      statusCode: extra.statusCode,
      retryable: false,
      shouldReport: false,
      shouldReset: false,
      isTodo: true,
      details: extra.details,
      cause: extra.cause,
    }
  );
}

function createExceptionDecision(action, options) {
  const extra = options || {};
  return {
    action: action || RegisterFailureAction.FAIL_FLOW,
    message: extra.message || "",
    statusCode: extra.statusCode,
    shouldReport: extra.shouldReport,
    shouldReset: extra.shouldReset,
    retryDelayMs: extra.retryDelayMs,
  };
}

function normalizeRegisterError(stageAction, err, options) {
  const extra = options || {};
  if (err && err.isRegisterError) {
    err.stageAction = err.stageAction || stageAction;
    err.attempt = extra.attempt || err.attempt || 1;
    return err;
  }

  if (err && err.action) {
    const normalizedDecisionError = createRegisterError(
      RegisterExceptionType.UNKNOWN,
      err.message || "register flow error",
      {
        stageAction: stageAction,
        statusCode: err.statusCode,
        retryable: err.action === RegisterFailureAction.RETRY_STAGE,
        shouldReport: err.shouldReport !== false,
        shouldReset: err.shouldReset !== false,
        isTodo: false,
        cause: err,
      }
    );
    normalizedDecisionError.attempt = extra.attempt || 1;
    normalizedDecisionError.decision = err;
    return normalizedDecisionError;
  }

  const message = err && err.message ? err.message : String(err);
  const normalized = createRegisterError(
    RegisterExceptionType.UNKNOWN,
    message,
    {
      stageAction: stageAction,
      retryable: false,
      shouldReport: !(err && err.isTodo),
      shouldReset: !(err && err.isTodo),
      isTodo: !!(err && err.isTodo),
      cause: err,
    }
  );
  normalized.attempt = extra.attempt || 1;
  return normalized;
}

module.exports = {
  createRegisterError,
  createTodoError,
  createExceptionDecision,
  normalizeRegisterError,
};
