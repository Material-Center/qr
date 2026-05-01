/**
 * @file gesture_util.js
 * @description 屏幕滑动与手势轨迹：横向/纵向滑动、贝塞尔曲线滑动、匀速/平滑曲线滑动等。
 * @see 依赖 Auto.js 全局：swipe、gesture、device、random
 */
/* global swipe, gesture, device, random */

/**
 * 三阶贝塞尔曲线插值（内部使用）。
 * @param {{x:number,y:number}[]} cp 四个控制点。
 * @param {number} t 参数 [0,1]。
 * @returns {{x:number,y:number}} 插值坐标。
 */
function bezierCurves(cp, t) {
  var cx = 3.0 * (cp[1].x - cp[0].x);
  var bx = 3.0 * (cp[2].x - cp[1].x) - cx;
  var ax = cp[3].x - cp[0].x - cx - bx;
  var cy = 3.0 * (cp[1].y - cp[0].y);
  var by = 3.0 * (cp[2].y - cp[1].y) - cy;
  var ay = cp[3].y - cp[0].y - cy - by;

  var tSquared = t * t;
  var tCubed = tSquared * t;
  return {
    x: ax * tCubed + bx * tSquared + cx * t + cp[0].x,
    y: ay * tCubed + by * tSquared + cy * t + cp[0].y,
  };
}

const GestureUtils = {
  /**
   * 闭区间 [min, max] 上的随机整数（使用 Math.random，与 Auto.js 全局 random 无关）。
   * @param {number} min 下限（含）。
   * @param {number} max 上限（含）。
   * @returns {number} 随机整数。
   */
  randomInt(min, max) {
    return Math.floor(Math.random() * (max - min + 1)) + min;
  },

  /**
   * 从屏幕偏右滑向偏左（约 0.8→0.2 屏宽），带小幅随机抖动。
   * @returns {void}
   */
  swipeToRight() {
    swipe(
      device.width * 0.8 + random(-20, 10),
      device.height * 0.5 + random(-20, 10),
      device.width * 0.2 + random(-20, 10),
      device.height * 0.5 + random(-20, 10),
      1
    );
  },

  /**
   * 从屏幕偏左滑向偏右（约 0.2→0.8 屏宽），带小幅随机抖动。
   * @returns {void}
   */
  swipeToLeft() {
    swipe(
      device.width * 0.2 + random(-20, 10),
      device.height * 0.5 + random(-20, 10),
      device.width * 0.8 + random(-20, 10),
      device.height * 0.5 + random(-20, 10),
      1
    );
  },

  /**
   * 从屏幕偏下方向上滑（列表向下滚动的常见手势）。
   * @param {number} [duration=500] 滑动持续时间（毫秒）。
   * @returns {void}
   */
  swipeToBottom(duration = 500) {
    swipe(
      device.width / 2 + random(-20, 10),
      device.height * 0.8 + random(-20, 10),
      device.width / 2 + random(-20, 10),
      device.height * 0.2 + random(-20, 10),
      duration
    );
  },

  /**
   * 从屏幕偏上方向下滑。
   * @param {number} [duration=500] 滑动持续时间（毫秒）。
   * @returns {void}
   */
  swipeToTop(duration = 500) {
    swipe(
      device.width / 2 + random(-20, 10),
      device.height * 0.2 + random(-20, 10),
      device.width / 2 + random(-20, 10),
      device.height * 0.8 + random(-20, 10),
      duration
    );
  },

  /**
   * 沿随机控制点的贝塞尔轨迹，从 (qx,qy) 滑到 (zx,zy)，总时长 time 毫秒。
   * @param {number} qx 起点 x。
   * @param {number} qy 起点 y。
   * @param {number} zx 终点 x。
   * @param {number} zy 终点 y。
   * @param {number} time 手势总时长（毫秒）。
   * @returns {void}
   */
  swipeRandom(qx, qy, zx, zy, time) {
    var xxy = [time];
    var point = [];
    var dx0 = { x: qx, y: qy };
    var dx1 = { x: random(qx - 100, qx + 100), y: random(qy, qy + 50) };
    var dx2 = { x: random(zx - 100, zx + 100), y: random(zy, zy + 50) };
    var dx3 = { x: zx, y: zy };
    point.push(dx0, dx1, dx2, dx3);
    for (let i = 0; i < 1.2; i += 0.08) {
      var p = bezierCurves(point, i);
      xxy.push([parseInt(p.x, 10), parseInt(p.y, 10)]);
    }
    gesture.apply(null, xxy);
  },

  /**
   * 直线匀速多点手势（由步数均分起点到终点）。
   * @param {number} startX 起点 x。
   * @param {number} startY 起点 y。
   * @param {number} endX 终点 x。
   * @param {number} endY 终点 y。
   * @param {number} duration 总时长（毫秒）。
   * @returns {void}
   */
  uniformSlide(startX, startY, endX, endY, duration) {
    let points = [];
    let steps = Math.max(Math.floor(duration / 20), 2);
    for (let i = 0; i <= steps; i++) {
      let progress = i / steps;
      let x = startX + (endX - startX) * progress;
      let y = startY + (endY - startY) * progress;
      points.push([x, y]);
    }
    gesture(duration, points);
  },

  /**
   * 带轻微弧度的三次贝塞尔轨迹手势（比 uniformSlide 更接近人手滑动）。
   * @param {number} startX 起点 x。
   * @param {number} startY 起点 y。
   * @param {number} endX 终点 x。
   * @param {number} endY 终点 y。
   * @param {number} duration 总时长（毫秒）。
   * @returns {void}
   */
  smoothUniformSlide(startX, startY, endX, endY, duration) {
    let controlX1 = startX + (endX - startX) * 0.3;
    let controlY1 = startY - 50;
    let controlX2 = startX + (endX - startX) * 0.7;
    let controlY2 = startY - 30;
    let points = [];
    let steps = Math.max(Math.floor(duration / 20), 10);
    points.push([startX, startY]);
    for (let i = 0; i <= steps; i++) {
      let progress = i / steps;
      let x =
        Math.pow(1 - progress, 3) * startX +
        3 * Math.pow(1 - progress, 2) * progress * controlX1 +
        3 * (1 - progress) * Math.pow(progress, 2) * controlX2 +
        Math.pow(progress, 3) * endX;
      let y =
        Math.pow(1 - progress, 3) * startY +
        3 * Math.pow(1 - progress, 2) * progress * controlY1 +
        3 * (1 - progress) * Math.pow(progress, 2) * controlY2 +
        Math.pow(progress, 3) * endY;
      points.push([x, y]);
    }
    gesture(duration, points);
  },
};

module.exports = { GestureUtils };
