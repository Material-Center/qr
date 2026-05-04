const VERIFY_CODE_STAGE_POLICY = {
  PLATFORM_SEND: {
    waitRounds: [120000, 120000],
    resendCount: 1,
    manualSubmitMaxAttempts: 0,
    manualSubmitIntervalMs: 0,
  },
  USER_SENT_TO_TX: {
    waitRounds: [],
    resendCount: 0,
    manualSubmitMaxAttempts: 3,
    manualSubmitIntervalMs: 10000,
  },
};

function clonePolicy(policy) {
  return {
    waitRounds: (policy.waitRounds || []).slice(),
    resendCount: Number(policy.resendCount || 0) || 0,
    manualSubmitMaxAttempts:
      Number(policy.manualSubmitMaxAttempts || 0) || 0,
    manualSubmitIntervalMs:
      Number(policy.manualSubmitIntervalMs || 0) || 0,
  };
}

function getVerifyCodeStagePolicy(mode) {
  const key = String(mode || "").trim();
  const policy = VERIFY_CODE_STAGE_POLICY[key];
  if (!policy) {
    return null;
  }
  return clonePolicy(policy);
}

function isVerifyCodeStagePassed(state) {
  const pageState = state || {};
  return (
    pageState.isInputVerifyCodePage !== true &&
    pageState.isSendVerifyCodeManualPage !== true &&
    pageState.hasNextStageFeature === true
  );
}

function normalizeDeviceTask(task) {
  if (!task || typeof task !== "object") {
    return null;
  }
  const normalized = {};
  Object.keys(task).forEach(function (key) {
    normalized[key] = task[key];
  });
  const id = Number(task.id || task.taskId || 0) || 0;
  normalized.id = id;
  normalized.taskId = id;
  return normalized;
}

module.exports = {
  getVerifyCodeStagePolicy,
  isVerifyCodeStagePassed,
  normalizeDeviceTask,
};
