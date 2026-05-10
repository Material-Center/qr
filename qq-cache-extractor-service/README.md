# QQ Cache Extractor Service

独立的 QQ 缓存提取服务，用于在服务端复刻 CacheTool 的缓存读取能力。

## 构建

```bash
cd qq-cache-extractor-service
./gradlew fatJar
```

输出：

```text
qq-cache-extractor-service/build/libs/extra-1.0.0.jar
```

## 启动服务

```bash
java -jar build/libs/extra-1.0.0.jar --port=19091
```

## Ubuntu 安装

在 Ubuntu 服务器上安装 Java、写入 systemd 服务并启动：

```bash
sudo INSTALL_DIR=/opt/extra \
  SERVICE_NAME=extra \
  SERVICE_PORT=19091 \
  JAR_SOURCE=/tmp/extra.jar \
  ./deploy/install-ubuntu.sh
```

安装后服务名：

```bash
systemctl status extra
```

## 仓库部署脚本

根目录部署脚本已支持单独部署：

```bash
./deploy.sh extra
```

`./deploy.sh all` 也会部署该服务。

可在根目录 `deploy.sh` 中调整：

```text
REMOTE_QQ_CACHE_EXTRACTOR_DIR=/opt/extra
QQ_CACHE_EXTRACTOR_SERVICE_NAME=extra
QQ_CACHE_EXTRACTOR_PORT=19091
QQ_CACHE_EXTRACTOR_INSTALL_JAVA=1
```

## 接口

健康检查：

```bash
curl http://127.0.0.1:19091/health
```

按服务端本地路径提取：

```bash
curl -X POST 'http://127.0.0.1:19091/extract' \
  -H 'Content-Type: application/json' \
  -d '{"inputPath":"/tmp/qq_cache.zip","clientId":"android-id","deviceInfo":"device"}'
```

直接上传 zip：

```bash
curl -X POST 'http://127.0.0.1:19091/extract?clientId=android-id' \
  -H 'Content-Type: application/zip' \
  --data-binary '@/tmp/qq_cache.zip'
```

CLI 模式：

```bash
java -jar build/libs/extra-1.0.0.jar \
  --input=/tmp/qq_cache.zip \
  --output=/tmp/qq_cache_result.json
```

## 输入文件

zip 或目录内需要包含：

- `wlogin_device.dat`
- `tk_file`
- `mobileQQ.xml` 或 `Properties`
- `uid` 目录，建议包含
- `uifa.xml`，可选
- `mmkv/qq_uin_uid_map`，可选兜底

服务会递归查找这些文件。
