# 配置 Runbook — item-rental server

服务配置来源:
1. **conf/app.conf** — beego ini 文件,由 `web.AppConfig` 读取(`AppConfig.DefaultString/DefaultInt` 等)。
2. **进程环境变量** — 代码中显式 `os.Getenv` 的键(见下表"环境变量"列)。

覆盖优先级:**环境变量 > app.conf**。未设置的键使用代码内默认值(表内"默认值"列)。

## 全键环境变量覆盖表

| # | 配置键 (app.conf) | 环境变量 | 默认值 | 用途 | Secret? | real/prod 必须显式设置 |
|---|-------------------|----------|--------|------|---------|------------------------|
| 1 | `appname` | — | `item-rental` | beego 应用名 | 否 | 否 |
| 2 | `httpport` | — | `8080` | HTTP 监听端口 | 否 | 是(按部署改端口) |
| 3 | `runmode` | `BEEGO_RUNMODE` | `dev` | beego 运行模式 | 否 | **是(必设 `prod`)** — dev 模式触发 `models.InitORM` 自动建表 `RunSyncdb`(见 models/orm.go) |
| 4 | `autorender` | — | `false` | 禁用模板自动渲染 | 否 | 否 |
| 5 | `copyrequestbody` | — | `true` | 复制请求体供中间件读取 | 否 | 否 |
| 6 | `sqlconn` | — | `root:root123456@tcp(127.0.0.1:3306)/rental?...` | MySQL DSN(含密码) | **是** | **是(必设)** — 当前为开发弱口令 |
| 7 | `redisaddr` | — | `127.0.0.1:6379` | Redis 地址 | 否 | **是(必设)** |
| 8 | `redispass` | — | `redis123456` | Redis 密码 | **是** | **是(必设)** |
| 9 | `jwtsecret` | `ITEM_RENTAL_JWT_SECRET` | `change-me-in-prod-rental-secret` | JWT 签名密钥 | **是** | **是(必设)** — 代码对默认值 fail-fast panic(middleware/jwt.go) |
| 10 | `jwtttl` | — | `7200` | access token 有效期(秒) | 否 | 否 |
| 11 | `jwtrt_ttl` | — | `604800` | refresh token 有效期(秒) | 否 | 否 |
| 12 | `SCOUT_DRIVER` | `SCOUT_DRIVER` | 未设→搜索降级为 null 引擎 | 搜索驱动 | 否 | **是(设 `opensearch`)** — ⚠ 仅读环境变量,app.conf 中的键无效(见下) |
| 13 | `OPENSEARCH_HTTP_HOST` | `OPENSEARCH_HTTP_HOST` | (go-scout 库内默认) | OpenSearch 地址 | 否 | **是(必设)** — ⚠ 仅读环境变量 |
| 14 | `OPENSEARCH_USERNAME` | `OPENSEARCH_USERNAME` | `admin` | OpenSearch 用户 | 否 | **是(必设)** — ⚠ 仅读环境变量 |
| 15 | `OPENSEARCH_PASSWORD` | `OPENSEARCH_PASSWORD` | `admin` | OpenSearch 密码 | **是** | **是(必设)** — ⚠ 仅读环境变量 |
| 16 | `SCOUT_SOFT_DELETE` | `SCOUT_SOFT_DELETE` | `false` | 搜索软删除 | 否 | 否 — ⚠ 仅读环境变量 |
| 17 | `ipban_threshold` | — | `5` | IP 封禁触发阈值(窗口内攻击次数) | 否 | 否 |
| 18 | `ipban_window` | — | `60` | 封禁统计窗口(秒) | 否 | 否 |
| 19 | `ipban_duration` | — | `900` | 封禁时长(秒) | 否 | 否 |
| 20 | `body_size_limit` | — | `10485760` | 请求体大小上限(字节,10MB) | 否 | 否 |
| 21 | `content_types` | — | `application/json,application/x-www-form-urlencoded,multipart/form-data` | 允许的 Content-Type 白名单 | 否 | 否 |
| 22 | `sms_provider` | — | `mock` | 短信渠道;`mock` 时 refresh 会话绕过 Redis(services/refresh.go) | 否 | **是(接真实网关后必设)** |
| 23 | `wechat_mock` | `WECHAT_MOCK` | 空(`""`,关闭) | 支付 mock 模式(=`1` 开启) | 否 | **是(设 `0` 或留空)** |
| 24 | `wechat_appid` | `WECHAT_APPID` | 空 | 微信支付 AppID | 否 | **是(必设)** |
| 25 | `wechat_mchid` | `WECHAT_MCHID` | 空 | 微信商户号 | 否 | **是(必设)** |
| 26 | `wechat_mchkey` | `WECHAT_MCHKEY` | 空 | 商户 API 密钥 | **是** | **是(必设)** |
| 27 | `wechat_notify_url` | `WECHAT_NOTIFY_URL` | 空 | 支付结果回调 URL | 否 | **是(必设)** |
| 28 | `wechat_sign_type` | — | `HMAC-SHA256` | 签名算法 | 否 | 否 |
| 29 | `wechat_timeout` | — | `10` | 下单 HTTP 超时(秒) | 否 | 否 |
| 30 | `wechat_cert_file` | `WECHAT_CERT_FILE` | 空 | 商户证书路径(退款等双向 TLS) | 否 | 使用退款功能时必设 |
| 31 | `wechat_cert_key` | `WECHAT_CERT_KEY` | 空 | 商户证书私钥路径 | **是** | 使用退款功能时必设 |
| 32 | (无 app.conf 键) | `ITEM_RENTAL_TRUSTED_PROXY` | 空 | 可信反向代理 CIDR 列表(逗号分隔;置空则不信任 X-Forwarded-For) | 否 | 位于反代后时必设(middleware/security_middleware.go) |
| 33 | `pii_key` | `ITEM_RENTAL_PII_KEY` | 空 | PII 加密密钥(AES-256,64 hex;phone sha256 + real_name 加密);env 优先 | **是** | **是(必设)** — 缺失/非 64 hex fail-fast panic(services/pii.go) |
| 34 | `ipban_file` | — | `data/ipban.json` | IP 封禁记录持久化文件(security-go storage.NewFile,30s 自动落盘) | 否 | 否(重启丢失可接受则留空也可;默认已持久化) |
| 35 | `cors_allow_origins` | — | `*` | CORS 允许来源(dev 联调用 *;**生产收紧为具体域名**) | 否 | 生产建议收紧 |

## 环境变量读取陷阱(重要)

- `SCOUT_DRIVER` / `OPENSEARCH_*` / `SCOUT_SOFT_DELETE`:虽然列在 `conf/app.conf`,**代码与 go-scout 库只读进程环境变量**(见 search/init.go),app.conf 中的同名键**不生效**。real/prod 必须通过部署层(systemd/k8s/docker `environment:`)注入真实环境变量,或删除 app.conf 中的同名行以免误导。
- 未接真实 OpenSearch 时,`SCOUT_DRIVER` 留空:服务自动降级为 null 空引擎,搜索返回空结果、不阻塞(search/init.go)。
- `jwtsecret` 与 `wechat_*` 支持环境变量覆盖;其余键仅从 app.conf 读取,改配置需改 app.conf 文件。

## real/prod 启动前必检清单

1. `BEEGO_RUNMODE=prod`(否则启动即自动建表,存在覆盖线上 schema 风险)。
2. `ITEM_RENTAL_JWT_SECRET` 强随机密钥(32+ 字节)。
3. `sqlconn` / `redispass` 替换开发默认口令。
4. `SCOUT_DRIVER=opensearch` + `OPENSEARCH_HTTP_HOST/USERNAME/PASSWORD`(或留空接受降级)。
5. `WECHAT_MOCK=0`,`WECHAT_APPID/MCHID/MCHKEY/NOTIFY_URL` 已配置。
6. `sms_provider` 切离 `mock`;反代场景配 `ITEM_RENTAL_TRUSTED_PROXY`。
7. `ITEM_RENTAL_PII_KEY` 强随机 64 hex(缺失/非 64 hex 启动即 panic,services/pii.go)。

## 生产 Docker Compose(deploy/docker-compose.prod.yml)

生产栈与 dev 完全隔离:`name: item-rental-prod` → 容器 `rental-*-prod`、卷 `item-rental-prod_*`、网络 `item-rental-prod_default`,与 `docker-compose.yml`(dev 栈)互不冲突。**dev 栈未改动**;两个栈共用宿主机端口时通过 host 端口覆盖错开(见下)。

### 必设密钥(缺失则 `docker compose up` 直接报错退出,不会用弱默认值)

| 环境变量 | 用途 | 注入点 |
|---|---|---|
| `MYSQL_ROOT_PASSWORD` | MySQL root 口令 | compose `mysql.environment`(必设,无默认值) |
| `MYSQL_PASSWORD` | 应用账号 `MYSQL_USER`(默认 `rental`)口令 | compose `mysql.environment`(必设) |
| `REDIS_PASSWORD` | Redis AUTH 口令(`redis-server --requirepass`) | compose `redis.command` 与 `redis.environment`(必设) |
| `OPENSEARCH_ADMIN_PASSWORD` | OpenSearch 初始 admin 口令(`OPENSEARCH_INITIAL_ADMIN_PASSWORD`) | compose `opensearch.environment`(必设) |

> 密码强度:OpenSearch security plugin 启用 zxcvbn 校验,**拒绝弱口令/与用户名相似口令**(冒烟中 `SmokeTestAdmin_1` 因"与用户名相似"被拒)。选强随机口令,勿用 `admin`/含 `admin` 的变体。
>
> MySQL 口令注意 shell 转义:`MYSQL_ROOT_PASSWORD='Pa$s_w0rd'` 用单引号;含 `$` 的字符串在 `.env` 中需按 compose 插值规则转义。

### 可选覆盖(host 端口冲突时)

| 环境变量 | 默认 | 说明 |
|---|---|---|
| `MYSQL_HOST_PORT` | `3307` | 映射到容器 3306(避开宿主 3306 占用) |
| `REDIS_HOST_PORT` | `16379` | 映射到容器 6379 |
| `OPENSEARCH_PORT` | `9200` | 映射到容器 9200 |

### 与 dev 的差异(仅 prod)

- OpenSearch **不再**重复设置 `plugins.security.disabled`;dev 栈同时设 `plugins.security.disabled=true` + `DISABLE_SECURITY_PLUGIN=true` 正是崩溃根因(两处同键冲突)。prod 设 `DISABLE_SECURITY_PLUGIN=false`,security ON + admin bootstrap,镜像内 `curl -k -u admin:$OPENSEARCH_ADMIN_PASSWORD` 才能访问,healthcheck 与 app 端都必须带认证。
- MySQL/Redis 都带 healthcheck;OpenSearch healthcheck 走 HTTPS(自签证书,`curl -skf`)。
- 微信商户证书:`deploy/docker-compose.prod.yml` 中 `app` 服务(默认注释、启用示例)以 `WECHAT_CERT_DIR:/certs:ro` 挂载证书目录,容器内路径为 `/certs/apiclient_cert.pem`、`/certs/apiclient_key.pem`,经 `WECHAT_CERT_FILE` / `WECHAT_CERT_KEY` 注入(payments/config.go 对应字段)。

### app 服务 env 注入点

prod compose 预留 `app` 服务(默认注释;如需启用,把 `APP_*` 键解开):`sqlconn` 指向 `mysql:3306`、`redisaddr` 指向 `redis:6379`、`redispass` 用 `${REDIS_PASSWORD}`;另需注入 `BEEGO_RUNMODE=prod`、`ITEM_RENTAL_JWT_SECRET`(32+ 字节强随机,代码对默认值 fail-fast panic)、`SCOUT_DRIVER=opensearch` + `OPENSEARCH_*`、`WECHAT_*` 全部键、`sms_provider` 相关、`ITEM_RENTAL_TRUSTED_PROXY`。`OPENSEARCH_HTTP_HOST` 用 `https://opensearch:9200`(带认证)。

### 启动方式

```bash
cd deploy
export MYSQL_ROOT_PASSWORD='...' MYSQL_PASSWORD='...' REDIS_PASSWORD='...' OPENSEARCH_ADMIN_PASSWORD='...'
docker compose -f docker-compose.prod.yml up -d
docker compose -f docker-compose.prod.yml ps    # 三个服务均应为 healthy
docker compose -f docker-compose.prod.yml config # 校验插值
```

初始化表结构由 `deploy/initdb/001_schema.sql`(挂载到 `/docker-entrypoint-initdb.d`,仅首次建卷时执行)完成。
