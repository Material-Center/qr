const RegisterAction = {
  OPEN_REGISTER_PAGE: "open_register_page",
  HANDLE_AUTHORIZE_DIALOG: "handle_authorize_dialog",
  INPUT_PHONE: "input_phone",
  SECURITY_VERIFY: "security_verify",
  WAIT_OR_SUBMIT_VERIFY_CODE: "wait_or_submit_verify_code",
  COMPLETE_PROFILE: "complete_profile",
  WAIT_LOGIN_SUCCESS: "wait_login_success",
  SUBMIT_CACHE: "submit_cache",
  RESET_ENVIRONMENT: "reset_environment",
  HANDLE_STAGE_EXCEPTION: "handle_stage_exception",
  STAGE_RETRY: "stage_retry",
  STAGE_SKIPPED: "stage_skipped",
  FLOW_FAILED: "flow_failed",
};

const RegisterExceptionType = {
  TODO: "todo",
  UNKNOWN: "unknown",
  PAGE_UNEXPECTED: "page_unexpected",
  AUTHORIZE_DIALOG_BLOCKED: "authorize_dialog_blocked",
  PHONE_INPUT_INVALID: "phone_input_invalid",
  VERIFY_CODE_TIMEOUT: "verify_code_timeout",
  VERIFY_CODE_REJECTED: "verify_code_rejected",
  RISK_CONTROL_PAGE: "risk_control_page",
  IMAGE_CHALLENGE_FAILED: "image_challenge_failed",
  PROFILE_SUBMIT_FAILED: "profile_submit_failed",
  LOGIN_RESULT_UNKNOWN: "login_result_unknown",
  CACHE_UPLOAD_FAILED: "cache_upload_failed",
  ENV_RESET_FAILED: "env_reset_failed",
};

const RegisterFailureAction = {
  FAIL_FLOW: "fail_flow",
  RETRY_STAGE: "retry_stage",
  CONTINUE: "continue",
  RESET_AND_FAIL: "reset_and_fail",
};

module.exports = {
  RegisterAction,
  RegisterExceptionType,
  RegisterFailureAction,
};
