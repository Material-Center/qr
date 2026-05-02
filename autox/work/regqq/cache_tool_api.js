function normalizeBaseURL(baseURL) {
  const url = String(baseURL || "").trim();
  if (!url) {
    throw new Error("cache tool api baseURL is required");
  }
  return url.replace(/\/+$/, "");
}

function parseResponse(res) {
  const statusCode = res.statusCode;
  const bodyText = res.body ? res.body.string() : "";
  if (statusCode !== 200) {
    throw new Error(
      "cache tool request failed: status=" + statusCode + " body=" + bodyText
    );
  }

  if (!bodyText) {
    return {};
  }

  let payload = {};
  try {
    payload = JSON.parse(bodyText);
  } catch (err) {
    throw new Error("cache tool invalid json response: " + err.message);
  }

  if (typeof payload.code === "number" && payload.code !== 0) {
    throw new Error(payload.msg || "cache tool upstream business error");
  }

  if (payload.status && payload.status !== "success") {
    throw new Error(payload.message || "cache tool business error");
  }

  return payload;
}

function safeGet(baseURL, path) {
  try {
    return parseResponse(http.get(baseURL + path));
  } catch (err) {
    return {
      ok: false,
      error: err,
    };
  }
}

function createCacheToolApiClient(options) {
  const baseURL = normalizeBaseURL(options && options.baseURL);

  function get(path) {
    return parseResponse(http.get(baseURL + path));
  }

  function post(path, data) {
    return parseResponse(http.postJson(baseURL + path, data || {}));
  }

  return {
    getStatus() {
      return get("/status");
    },
    getStatusSafe() {
      return safeGet(baseURL, "/status");
    },
    ensureServiceRunning() {
      const status = get("/status");
      if (!status || status.running !== true) {
        throw new Error("cache tool service is not running");
      }
      return status;
    },
    pushPhoneRegisterCache(payload) {
      return post("/phoneRegister/pushCache", payload);
    },
  };
}

module.exports = {
  createCacheToolApiClient,
};
