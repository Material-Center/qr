/**
 * @file http_util.js
 * @description 通用 HTTP 下载等能力，不包含业务后台接口地址。
 * @see 依赖 Auto.js 全局：http、files、console
 */

const HttpUtils = {
  /**
   * 通过 GET 下载二进制内容并写入本地路径（自动创建父目录）。
   * @param {string} url 完整下载地址。
   * @param {string} filePath 目标文件绝对路径或相对路径。
   * @returns {string} 写入成功后的 filePath（与入参相同）。
   * @throws {Error} HTTP 状态非 200、响应体为空或写入失败时抛出。
   */
  downloadFile(url, filePath) {
    const now = Date.now();
    console.log("start download file: " + url + " to " + filePath);
    const res = http.get(url);
    if (res.statusCode != 200) {
      console.error(
        "download file failed: code: " +
          res.statusCode +
          " msg: " +
          res.body.string()
      );
      throw new Error("download file failed");
    }
    const data = res.body.bytes();
    console.log(
      "download file success: code: " + res.statusCode + " size: " + data.length
    );
    if (data.length == 0) {
      console.error("download file failed: file content is empty");
      throw new Error("download file failed");
    }
    files.createWithDirs(filePath);
    files.writeBytes(filePath, data);
    console.log(
      "download file success: size: " +
        data.length +
        " time: " +
        (Date.now() - now) +
        "ms"
    );
    return filePath;
  },
};

module.exports = { HttpUtils };
