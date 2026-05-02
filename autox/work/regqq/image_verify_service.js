function normalizeQuestion(question, config) {
  const q = String(question || "").trim();
  if (q) {
    return q;
  }
  return String((config && config.question) || "").trim();
}

function ensureImageVerifyEnabled(config) {
  if (!config || !config.endpoint) {
    throw new Error("图片验证 endpoint 未配置");
  }
  if (!config.keyCode) {
    throw new Error("图片验证 keyCode 未配置");
  }
}

function normalizePermissionTexts(config) {
  const rawTexts = (config && config.permissionAllowTexts) || [
    "立即开始",
    "允许",
    "开始",
  ];
  const texts = [];
  for (let i = 0; i < rawTexts.length; i++) {
    const textValue = String(rawTexts[i] || "").trim();
    if (textValue) {
      texts.push(textValue);
    }
  }
  return texts;
}

function saveImageForDebug(image, config) {
  const dir = (config && config.screenshotDir) || files.cwd();
  const filePath = dir + "/verify_" + Date.now() + ".jpg";
  files.createWithDirs(filePath);
  images.save(image, filePath, "jpg", 80);
  return filePath;
}

function imageToBase64(image) {
  if (typeof images.toBase64 === "function") {
    return images.toBase64(image, "jpg", 80);
  }

  const tmpPath = files.cwd() + "/capture/_tmp_verify_" + Date.now() + ".jpg";
  files.createWithDirs(tmpPath);
  images.save(image, tmpPath, "jpg", 80);
  const bytes = files.readBytes(tmpPath);
  files.remove(tmpPath);

  importClass(java.util.Base64);
  return Base64.encodeToString(bytes, Base64.NO_WRAP);
}

function tryParseJson(text) {
  if (!text) {
    return {};
  }
  return JSON.parse(text);
}

function normalizePointsFromUnknown(result) {
  if (!result) {
    return [];
  }

  if (Array.isArray(result.points)) {
    return result.points;
  }
  if (result.data && Array.isArray(result.data.points)) {
    return result.data.points;
  }
  if (result.data && Array.isArray(result.data)) {
    return result.data;
  }
  if (Array.isArray(result.result)) {
    return result.result;
  }
  return [];
}

function toPoint(item) {
  if (!item) {
    return null;
  }

  if (Array.isArray(item) && item.length >= 2) {
    return {
      x: Number(item[0]),
      y: Number(item[1]),
    };
  }

  if (typeof item === "string") {
    const parts = item.split(",");
    if (parts.length >= 2) {
      return {
        x: Number(parts[0]),
        y: Number(parts[1]),
      };
    }
  }

  if (item.x !== undefined && item.y !== undefined) {
    return {
      x: Number(item.x),
      y: Number(item.y),
    };
  }

  if (item.X !== undefined && item.Y !== undefined) {
    return {
      x: Number(item.X),
      y: Number(item.Y),
    };
  }

  if (item.left !== undefined && item.top !== undefined) {
    return {
      x: Number(item.left),
      y: Number(item.top),
    };
  }

  return null;
}

function createImageVerifyService(config) {
  const imageVerifyConfig = config || {};
  let screenCaptureReady = false;

  function startCapturePermissionWatcher() {
    const timeoutMs =
      Number(imageVerifyConfig.permissionDialogTimeoutMs || 8000) || 8000;
    const allowTexts = normalizePermissionTexts(imageVerifyConfig);
    if (!allowTexts.length) {
      return null;
    }
    return threads.start(function () {
      const deadline = Date.now() + timeoutMs;
      while (Date.now() < deadline) {
        for (let i = 0; i < allowTexts.length; i++) {
          const allowText = allowTexts[i];
          const selector = text(allowText);
          const button = selector.findOne(300);
          if (!button) {
            continue;
          }
          if (typeof button.click === "function" && button.click()) {
            return;
          }
          click(allowText);
          return;
        }
        sleep(200);
      }
    });
  }

  function ensureScreenCaptureReady() {
    if (screenCaptureReady) {
      return true;
    }
    let watcher = null;
    if (device.sdkInt > 28) {
      watcher = startCapturePermissionWatcher();
    }
    const ok = requestScreenCapture(false);
    if (watcher && watcher.isAlive()) {
      watcher.interrupt();
    }
    if (!ok) {
      throw new Error("请求截图权限失败");
    }
    screenCaptureReady = true;
    return true;
  }

  function captureImage(options) {
    ensureScreenCaptureReady();
    const screen = captureScreen();
    if (!screen) {
      throw new Error("截图失败");
    }

    const region = options && options.region;
    if (!region) {
      return {
        image: screen,
        region: null,
        recycle: false,
      };
    }

    const x = Number(region.x || 0);
    const y = Number(region.y || 0);
    const width = Number(region.width || 0);
    const height = Number(region.height || 0);
    if (width <= 0 || height <= 0) {
      throw new Error("截图区域无效");
    }
    return {
      image: images.clip(screen, x, y, width, height),
      region: {
        x: x,
        y: y,
        width: width,
        height: height,
      },
      recycle: true,
    };
  }

  function applyRegionOffset(points, region) {
    if (!region) {
      return points;
    }
    const offsetX = Number(region.x || 0);
    const offsetY = Number(region.y || 0);
    const normalized = [];
    for (let i = 0; i < points.length; i++) {
      normalized.push({
        x: points[i].x + offsetX,
        y: points[i].y + offsetY,
      });
    }
    return normalized;
  }

  function predictImage(base64Image, question, options) {
    ensureImageVerifyEnabled(imageVerifyConfig);
    const req = {
      base64Image: base64Image,
      modelName: String(imageVerifyConfig.modelName || "普通模型"),
      keyCode: String(imageVerifyConfig.keyCode || ""),
      question: normalizeQuestion(question, imageVerifyConfig),
      system: String(imageVerifyConfig.system || ""),
    };

    const endpoint =
      (options && options.endpoint) || String(imageVerifyConfig.endpoint);
    const response = http.postJson(endpoint, req);
    const bodyText = response.body ? response.body.string() : "";
    if (response.statusCode !== 200) {
      throw new Error(
        "图片验证请求失败: status=" + response.statusCode + " body=" + bodyText
      );
    }
    return tryParseJson(bodyText);
  }

  function parsePoints(result, region) {
    const rawPoints = normalizePointsFromUnknown(result);
    const points = [];
    for (let i = 0; i < rawPoints.length; i++) {
      const point = toPoint(rawPoints[i]);
      if (!point) {
        continue;
      }
      if (Number.isNaN(point.x) || Number.isNaN(point.y)) {
        continue;
      }
      points.push(point);
    }
    return applyRegionOffset(points, region);
  }

  function clickPoints(points, options) {
    if (!Array.isArray(points) || points.length === 0) {
      throw new Error("没有可点击的坐标点");
    }
    const pressDurationMs =
      (options && options.pressDurationMs) ||
      imageVerifyConfig.pressDurationMs ||
      50;
    const clickIntervalMs =
      (options && options.clickIntervalMs) ||
      imageVerifyConfig.clickIntervalMs ||
      300;

    for (let i = 0; i < points.length; i++) {
      const point = points[i];
      press(point.x, point.y, pressDurationMs);
      sleep(clickIntervalMs);
    }
    return points;
  }

  function capturePredictAndClick(question, options) {
    const capture = captureImage(options);
    try {
      const debugPath = saveImageForDebug(capture.image, imageVerifyConfig);
      const base64Image = imageToBase64(capture.image);
      const result = predictImage(base64Image, question, options);
      const points = parsePoints(result, capture.region);
      return {
        debugPath: debugPath,
        base64Image: base64Image,
        result: result,
        points: points,
        click: function () {
          return clickPoints(points, options);
        },
      };
    } finally {
      if (capture.recycle && capture.image && capture.image.recycle) {
        capture.image.recycle();
      }
    }
  }

  return {
    ensureScreenCaptureReady: ensureScreenCaptureReady,
    captureImage: captureImage,
    predictImage: predictImage,
    parsePoints: parsePoints,
    clickPoints: clickPoints,
    capturePredictAndClick: capturePredictAndClick,
  };
}

module.exports = {
  createImageVerifyService,
};
