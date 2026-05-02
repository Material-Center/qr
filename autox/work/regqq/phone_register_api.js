/**
 * @file phone_register_api.js
 * @description regqq 业务专用的手机号注册任务服务端 API client，仅封装接口调用。
 * @see 依赖 Auto.js 全局：http
 */

function normalizeBaseURL(baseURL) {
  const url = String(baseURL || "").trim();
  if (!url) {
    throw new Error("phone register api baseURL is required");
  }
  return url.replace(/\/+$/, "");
}

function encodeQuery(params) {
  const pairs = [];
  Object.keys(params || {}).forEach((key) => {
    const value = params[key];
    if (value === undefined || value === null || value === "") {
      return;
    }
    pairs.push(
      encodeURIComponent(key) + "=" + encodeURIComponent(String(value))
    );
  });
  return pairs.join("&");
}

function parseServerResponse(res) {
  const statusCode = res.statusCode;
  const bodyText = res.body ? res.body.string() : "";
  if (statusCode !== 200) {
    throw new Error(
      "request failed: status=" + statusCode + " body=" + bodyText
    );
  }
  let payload = {};
  try {
    payload = bodyText ? JSON.parse(bodyText) : {};
  } catch (err) {
    throw new Error("invalid json response: " + err.message);
  }
  if (payload.code !== 0) {
    throw new Error(payload.msg || "server business error");
  }
  return payload.data;
}

function createPhoneRegisterApiClient(options) {
  const baseURL = normalizeBaseURL(options && options.baseURL);

  function buildURL(path, query) {
    const url = baseURL + path;
    const queryString = encodeQuery(query);
    if (!queryString) {
      return url;
    }
    return url + "?" + queryString;
  }

  function get(path, query) {
    return parseServerResponse(http.get(buildURL(path, query)));
  }

  function post(path, data) {
    return parseServerResponse(http.postJson(buildURL(path), data || {}));
  }

  return {
    pollTask(deviceId) {
      return post("/phoneRegisterTask/device/poll", { deviceId });
    },

    getTask(deviceId) {
      return get("/phoneRegisterTask/device/task", { deviceId });
    },

    getTaskByPost(deviceId) {
      return post("/phoneRegisterTask/device/task", { deviceId });
    },

    heartbeat(deviceId) {
      return post("/phoneRegisterTask/device/heartbeat", { deviceId });
    },

    report(deviceId, action, message, statusCode) {
      const payload = {
        deviceId,
        action,
        message,
      };
      if (statusCode !== undefined && statusCode !== null) {
        payload.statusCode = statusCode;
      }
      return post("/phoneRegisterTask/device/report", payload);
    },

    getConfig(deviceId) {
      return get("/phoneRegisterTask/device/config", { deviceId });
    },
  };
}

module.exports = {
  createPhoneRegisterApiClient,
};
