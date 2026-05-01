/**
 * @file node_util.js
 * @description 无障碍 UI 自动化：选择器存在性、点击、等待、数字输入等，封装 Auto.js 控件 API。
 * @see 依赖 Auto.js 全局：text、className、desc、id、textContains、descContains、descMatches、textMatches、device、click、sleep、KeyCode、log
 */
/* global text, className, desc, id, textContains, descContains, descMatches, textMatches, device, click, sleep, KeyCode, log */

/** 数字字符到 Android KeyEvent 键码的映射（内部使用）。 */
const numberKeyCodeMap = {
  0: 7,
  1: 8,
  2: 9,
  3: 10,
  4: 11,
  5: 12,
  6: 13,
  7: 14,
  8: 15,
  9: 16,
};

/**
 * 将 kind 转为 Auto.js 选择器对象。
 * @param {"text"|"name"|"desc"|"id"} kind text=文本精确；name=类名 className；desc=描述；id=资源 id。
 * @param {string} value 匹配字符串。
 * @returns {object|null} UiSelector 或无法识别时 null。
 */
function selectorForKind(kind, value) {
  if (kind === "text") {
    return text(value);
  }
  if (kind === "name") {
    return className(value);
  }
  if (kind === "desc") {
    return desc(value);
  }
  if (kind === "id") {
    return id(value);
  }
  return null;
}

const NodeUtils = {
  /**
   * 在给定超时内判断选择器是否能找到控件。
   * @param {object} selector Auto.js UiSelector。
   * @param {number} [timeout=1000] 等待毫秒数。
   * @returns {boolean} 找到为 true，超时或异常为 false。
   */
  quickExists(selector, timeout = 1000) {
    try {
      return selector.findOne(timeout) !== null;
    } catch (e) {
      return false;
    }
  },

  /**
   * 对选择器结果执行中心坐标点击，支持失败重试。
   * @param {object} selector Auto.js UiSelector。
   * @param {{timeout?: number, maxRetries?: number}} [options] timeout 单次查找超时；maxRetries 最大尝试次数。
   * @returns {boolean} 至少一次点击成功为 true，否则 false。
   */
  safeClick(selector, options = {}) {
    const { timeout = 3000, maxRetries = 2 } = options;
    for (let i = 0; i < maxRetries; i++) {
      try {
        let element = selector.findOne(timeout);
        if (element) {
          const rect = element.bounds();
          click(rect.centerX(), rect.centerY());
          return true;
        }
      } catch (e) {
        console.log(`点击尝试${i + 1},失败`);
        if (i < maxRetries - 1) {
          sleep(500);
        }
      }
    }
    return false;
  },

  /**
   * 点击已有 UiObject 的中心坐标。
   * @param {object|null} element Auto.js 控件实例。
   * @returns {boolean} 成功发起点击为 true；element 为空为 false。
   */
  clickByElement(element) {
    if (!element) {
      return false;
    }
    const rect = element.bounds();
    click(rect.centerX(), rect.centerY());
    return true;
  },

  /**
   * 批量检测多个选择器在超时内是否存在。
   * @param {Object.<string, object>} selectors 键为自定义名称，值为 UiSelector。
   * @param {number} [timeout=2000] 每个选择器的等待毫秒数。
   * @returns {Object.<string, boolean>} 与入参键一致的存在性结果表。
   */
  checkMultiple(selectors, timeout = 2000) {
    let results = {};
    for (let [key, selector] of Object.entries(selectors)) {
      try {
        results[key] = NodeUtils.quickExists(selector, timeout);
      } catch (e) {
        results[key] = false;
      }
      sleep(100);
    }
    return results;
  },

  /**
   * 按类型精确匹配后点击控件（使用控件自带 click()）。
   * @param {"text"|"name"|"desc"|"id"} kind 选择器类型。
   * @param {string} value 匹配文案或类名或 id。
   * @param {number} [waitMs=1000] findOne 超时毫秒数。
   * @returns {boolean} 找到并点击成功为 true；未找到或 kind 非法为 false。
   */
  clickBySelector(kind, value, waitMs = 1000) {
    const sel = selectorForKind(kind, value);
    if (!sel) {
      return false;
    }
    const el = sel.findOne(waitMs);
    if (!el) {
      return false;
    }
    el.click();
    return true;
  },

  /**
   * 按「文字包含」查找控件，并点击其 bounds 中心（避免误用 click 传入字符串）。
   * @param {string} textFragment 界面显示文本的子串。
   * @param {number} [waitMs=1000] findOne 超时毫秒数。
   * @returns {boolean} Auto.js click() 的返回值；未找到控件时为 false。
   */
  clickTextContains(textFragment, waitMs = 1000) {
    if (!textFragment) {
      return false;
    }
    const el = textContains(textFragment).findOne(waitMs);
    if (!el) {
      return false;
    }
    const rect = el.bounds();
    return click(rect.centerX(), rect.centerY());
  },

  /**
   * 精确匹配控件是否存在于当前界面。
   * @param {"text"|"name"|"desc"|"id"} kind 选择器类型。
   * @param {string} value 匹配值。
   * @returns {boolean} exists() 结果；kind 非法时为 false。
   */
  nodeExists(kind, value) {
    const sel = selectorForKind(kind, value);
    return sel ? sel.exists() : false;
  },

  /**
   * 「包含」匹配控件是否存在（仅支持 text、desc 两类）。
   * @param {"text"|"desc"} kind textContains 或 descContains。
   * @param {string} value 子串。
   * @returns {boolean} 存在为 true；不支持的 kind 为 false。
   */
  nodeMatchExists(kind, value) {
    if (kind === "text") {
      return textContains(value).exists();
    }
    if (kind === "desc") {
      return descContains(value).exists();
    }
    return false;
  },

  /**
   * 在全屏范围内判断 descMatches 或 textMatches 正则是否命中任一控件。
   * @param {string|RegExp} reg 正则表达式或字符串形式正则。
   * @returns {boolean} 命中为 true，否则 false。
   */
  fullscreenRegexExists(reg) {
    if (
      descMatches(reg).boundsInside(0, 0, device.width, device.height).exists()
    ) {
      console.log("文本-->", reg, descMatches(reg).findOne().desc());
      return true;
    }
    if (
      textMatches(reg).boundsInside(0, 0, device.width, device.height).exists()
    ) {
      console.log("文本-->", reg, textMatches(reg).findOne().text());
      return true;
    }
    return false;
  },

  /**
   * 轮询直到 nodeExists 为真或超时。
   * @param {"text"|"name"|"desc"|"id"} kind 选择器类型。
   * @param {string} value 匹配值。
   * @param {number} maxWaitMs 最长等待毫秒数（约每 50ms 轮询一次）。
   * @returns {boolean} 在时限内出现为 true，超时为 false。
   */
  waitNodeExists(kind, value, maxWaitMs) {
    var n = 0;
    while (n < maxWaitMs) {
      sleep(50);
      if (NodeUtils.nodeExists(kind, value)) {
        return true;
      }
      n += 50;
    }
    return false;
  },

  /**
   * 轮询直到 nodeMatchExists 为真或超时。
   * @param {"text"|"desc"} kind 仅 text / desc。
   * @param {string} value 子串。
   * @param {number} maxWaitMs 最长等待毫秒数。
   * @returns {boolean} 在时限内出现为 true，超时为 false。
   */
  waitNodeMatchExists(kind, value, maxWaitMs) {
    var n = 0;
    while (n < maxWaitMs) {
      sleep(50);
      if (NodeUtils.nodeMatchExists(kind, value)) {
        return true;
      }
      n += 50;
    }
    return false;
  },

  /**
   * 短超时查找可见控件，可选点击。
   * @param {object} selector UiSelector（建议链式 visibleToUser 等）。
   * @param {boolean} isclick 为 true 时对找到控件执行 .click()。
   * @returns {boolean} 找到控件为 true（与是否点击成功取决于控件）；异常或未找到为 false。
   */
  findNode(selector, isclick) {
    let g_ret = null;
    try {
      g_ret = selector.visibleToUser(true).findOne(10);
      if (g_ret != null) {
        if (isclick) {
          g_ret.click();
        }
        return true;
      }
    } catch (e) {
      log(e);
    }
    return false;
  },

  /**
   * 短超时查找可见控件，可选点击中心坐标（相对 findNode 更贴近模拟点击）。
   * @param {object} selector UiSelector。
   * @param {boolean} isclick 为 true 时点击 bounds 中心。
   * @returns {boolean} 找到为 true；否则 false。
   */
  findNodeXY(selector, isclick) {
    let g_ret = null;
    try {
      g_ret = selector.visibleToUser(true).findOne(10);
      if (g_ret != null) {
        if (isclick) {
          g_ret = g_ret.bounds();
          sleep(20);
          click(g_ret.centerX(), g_ret.centerY());
        }
        return true;
      }
    } catch (e) {
      log(e);
    }
    return false;
  },

  /**
   * 自当前节点向上查找第一个 clickable 的祖先节点。
   * @param {object|null} node UiObject。
   * @returns {object|null} 可点击祖先；不存在或无父链时 null。
   */
  findClickableAncestor(node) {
    if (!node) {
      return null;
    }
    if (node.clickable()) {
      return node;
    }
    if (!node.parent()) {
      return null;
    }
    return NodeUtils.findClickableAncestor(node.parent());
  },

  /**
   * 自节点向上直到遇到 clickable 节点（与 findClickableAncestor 类似，原地沿 parent 爬升）。
   * @param {object|null} e 起始 UiObject。
   * @returns {object|null} 第一个可点击节点；无则返回 null（当 e 已为 null 时）。
   */
  topClickedView(e) {
    while (e && !e.clickable()) {
      e = e.parent();
    }
    return e;
  },

  /**
   * 点击 UiObject：可选强制坐标点击，否则优先 .click()，不可点击则递归父节点。
   * @param {object|null} uiObject 目标控件。
   * @param {boolean} useCoordinate 为 true 时始终点击 bounds 中心。
   * @returns {boolean} click 系列 API 的布尔结果；无有效对象时为 false。
   */
  clickUiObject(uiObject, useCoordinate) {
    if (!uiObject) {
      return false;
    }
    if (useCoordinate) {
      var rect = uiObject.bounds();
      return click(rect.centerX(), rect.centerY());
    }
    if (uiObject.clickable()) {
      return uiObject.click();
    }
    return NodeUtils.clickUiObject(uiObject.parent(), useCoordinate);
  },

  /**
   * 通过 KeyCode 依次输入数字字符串（如密码纯数字）。
   * @param {string|number} numstr 仅含 0-9 的字符串或数字。
   * @returns {void} 无返回值；非法字符无映射时 KeyCode(undefined) 行为由运行时决定。
   */
  inputNumber(numstr) {
    const arr = String(numstr).split("");
    for (let i = 0; i < arr.length; i++) {
      KeyCode(numberKeyCodeMap[arr[i]]);
      sleep(50);
    }
  },
};

module.exports = { NodeUtils };
