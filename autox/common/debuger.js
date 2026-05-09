/**
 * @file debuger.js
 * @description 无障碍节点树调试：打印层级、按关键字搜集相似节点（类名拼写为 debuger 保持文件名一致）。
 * @see 依赖 Auto.js 全局：auto、log
 */

/**
 * 节点树调试器。
 */
class NodeDebugger {
  constructor() {
    /** @type {boolean} 为 false 时 dumpNodeTree 不输出。 */
    this.debugMode = true;
  }

  /**
   * 自根节点向下打印控件树（类名、部分 text/id/desc、bounds）。
   * @param {number} [maxDepth=3] 最大递归深度，根为 0。
   * @returns {void} 无法取根或无权限时打日志后返回。
   */
  dumpNodeTree(maxDepth = 3, logger) {
    if (!this.debugMode) return;

    const output = typeof logger === "function" ? logger : log;
    let root = auto.root;
    if (!root) {
      output("无法获取根节点");
      return;
    }

    this._dumpNode(root, 0, maxDepth, output);
  }

  /**
   * @param {object} node UiObject
   * @param {number} depth 当前深度
   * @param {number} maxDepth 最大深度
   * @returns {void}
   * @private
   */
  _dumpNode(node, depth, maxDepth, logger) {
    if (depth > maxDepth) return;

    let indent = "  ".repeat(depth);
    let info = `${indent}${node.className()}`;

    if (node.text()) info += ` text:${node.text().substring(0, 20)}`;
    if (node.id()) info += ` id:${node.id()}`;
    if (node.desc()) info += ` desc:${node.desc().substring(0, 20)}`;
    if (node.bounds()) info += ` bounds:${node.bounds()}`;

    logger(info);

    for (let i = 0; i < node.childCount(); i++) {
      this._dumpNode(node.child(i), depth + 1, maxDepth, logger);
    }
  }

  /**
   * 深度优先遍历，收集 className/text/desc 拼接串中包含关键字的节点。
   * @param {string} partialInfo 不区分大小写的子串。
   * @returns {Array<{node: object, className: string, text: string, desc: string, bounds: object}>} 匹配项列表。
   */
  findSimilarNodes(partialInfo) {
    let results = [];
    this._findSimilar(auto.root, partialInfo, results);
    return results;
  }

  /**
   * @param {object|null} node
   * @param {string} partialInfo
   * @param {Array} results 就地追加
   * @returns {void}
   * @private
   */
  _findSimilar(node, partialInfo, results) {
    if (!node) return;

    let nodeInfo =
      node.className() + " " + (node.text() || "") + " " + (node.desc() || "");
    if (nodeInfo.toLowerCase().includes(partialInfo.toLowerCase())) {
      results.push({
        node: node,
        className: node.className(),
        text: node.text(),
        desc: node.desc(),
        bounds: node.bounds(),
      });
    }

    for (let i = 0; i < node.childCount(); i++) {
      this._findSimilar(node.child(i), partialInfo, results);
    }
  }
}

module.exports = NodeDebugger;
