# 配置 Runbook — item-rental server

服务配置来源:
1. **conf/app.conf** — beego ini 文件,由 `web.AppConfig` 读取(`AppConfig.DefaultString/DefaultInt/DefaultBool` 等)。
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
| 7 | `redisaddr` | — | `127.0.0.1:6379` | Redis 地址 | 否 | 启用 Redis 会话时必设 |
| 8 | `redispass` | — | `redis123456` | Redis 密码 | **是** | 启用 Redis 会话时必设 |
| 9 | `redis_enabled` | — | `false` | refresh 会话是否落 Redis;多实例部署必须 `true` | 否 | 多实例必设 `true` |
| 10 | `jwtsecret` | `ITEM_RENTAL_JWT_SECRET` | `change-me-in-prod-rental-secret` | JWT 签名密钥 | **是** | **是(必设)** — 代码对默认值 fail-fast panic(middleware/jwt.go) |
| 11 | `jwtttl` | — | `7200` | access token 有效期(秒) | 否 | 否 |
| 12 | `jwtrt_ttl` | — | `604800` | refresh token 有效期(秒) | 否 | 否 |
| 13 | `wechat_mock` | `WECHAT_MOCK` | 空(`""`,关闭) | 支付 mock 模式(=`1` 开启) | 否 | **是(设 `0` 或留空)** |
| 14 | `wechat_appid` | `WECHAT_APPID` | 空 | 微信支付 AppID | 否 | **是(必设)** |
| 15 | `wechat_mchid` | `WECHAT_MCHID` | 空 | 微信商户号 | 否 | **是(必设)** |
| 16 | `wechat_mchkey` | `WECHAT_MCHKEY` | 空 | 商户 API 密钥 | **是** | **是(必设)** |
| 17 | `wechat_notify_url` | `WECHAT_NOTIFY_URL` | 空 | 支付结果回调 URL | 否 | **是(必设)** |
| 18 | `wechat_sign_type` | — | `HMAC-SHA256` | 签名算法 | 否 | 否 |
| 19 | `wechat_timeout` | — | `10` | 下单 HTTP 超时(秒) | 否 | 否 |
| 20 | `wechat_cert_file` | `WECHAT_CERT_FILE` | 空 | 商户证书路径(退款等双向 TLS) | 否 | 使用退款功能时必设 |
| 21 | `wechat_cert_key` | `WECHAT_CERT_KEY` | 空 | 商户证书私钥路径 | **是** | 使用退款功能时必设 |
| 22 | `pii_key` | `ITEM_RENTAL_PII_KEY` | 空 | PII 加密密钥(AES-256,64 hex;phone sha256 + real_name 加密);env 优先 | **是** | **是(必设)** — 缺失/非 64 hex fail-fast panic(services/pii.go) |
| 23 | `cors_allow_origins` | — | `*` | CORS 允许来源(dev 联调用 *;**生产收紧为具体域名**) | 否 | 生产建议收紧 |
| 24 | `snowflake_node` | — | `1` | snowflake 节点号(0-1023);**多实例部署每实例须唯一**,否则跨实例同毫秒可能撞号 | 否 | 多实例部署必设(services/snowid.go) |

## 环境变量读取陷阱(重要)

- `redis_enabled=false` 时 refresh 会话不落 Redis(单机/本地开发足够);多实例部署必须设为 `true`,否则各实例的 refresh 会话不共享,单端登出/轮换只在当前实例生效。
- `jwtsecret` 与 `wechat_*` 支持环境变量覆盖;其余键仅从 app.conf 读取,改配置需改 app.conf 文件。
- 未配置微信商户参数时保持 `WECHAT_MOCK=1`(或空)走 mock 支付;真实支付需同时配置商户号/密钥/回调地址。

## real/prod 启动前必检清单

1. `BEEGO_RUNMODE=prod`(否则启动即自动建表,存在覆盖线上 schema 风险)。
2. `ITEM_RENTAL_JWT_SECRET` 强随机密钥(32+ 字节)。
3. `sqlconn` / `redispass` 替换开发默认口令。
4. `WECHAT_MOCK=0`,`WECHAT_APPID/MCHID/MCHKEY/NOTIFY_URL` 已配置。
5. 多实例部署设 `redis_enabled=true`;单实例可保持 `false`。
6. `ITEM_RENTAL_PII_KEY` 强随机 64 hex(缺失/非 64 hex 启动即 panic,services/pii.go)。
7. 生产 CORS 收紧为具体域名(`cors_allow_origins`)。

## 生产 Docker Compose(deploy/docker-compose.prod.yml)

生产栈与 dev 完全隔离:`name: item-rental-prod` → 容器 `rental-*-prod`、卷 `item-rental-prod_*`、网络 `item-rental-prod_default`,与 `docker-compose.yml`(dev 栈)互不冲突。两个栈共用宿主机端口时通过 host 端口覆盖错开(见下)。

### 必设密钥(缺失则 `docker compose up` 直接报错退出,不会用弱默认值)

| 环境变量 | 用途 | 注入点 |
|---|---|---|
| `MYSQL_ROOT_PASSWORD` | MySQL root 口令 | compose `mysql.environment`(必设,无默认值) |
| `MYSQL_PASSWORD` | 应用账号 `MYSQL_USER`(默认 `rental`)口令 | compose `mysql.environment`(必设) |
| `REDIS_PASSWORD` | Redis AUTH 口令(`redis-server --requirepass`) | compose `redis.command` 与 `redis.environment`(必设) |

> MySQL 口令注意 shell 转义:`MYSQL_ROOT_PASSWORD='Pa$s_w0rd'` 用单引号;含 `$` 的字符串在 `.env` 中需按 compose 插值规则转义。

### 可选覆盖(host 端口冲突时)

| 环境变量 | 默认 | 说明 |
|---|---|---|
| `MYSQL_HOST_PORT` | `3307` | 映射到容器 3306(避开宿主 3306 占用) |
| `REDIS_HOST_PORT` | `16379` | 映射到容器 6379 |

### 与 dev 的差异(仅 prod)

- MySQL/Redis 都带 healthcheck 与强口令;不再编排 OpenSearch。
- 微信商户证书:`deploy/docker-compose.prod.yml` 中 `app` 服务(默认注释、启用示例)以 `WECHAT_CERT_DIR:/certs:ro` 挂载证书目录,容器内路径为 `/certs/apiclient_cert.pem`、`/certs/apiclient_key.pem`,经 `WECHAT_CERT_FILE` / `WECHAT_CERT_KEY` 注入(payments/config.go 对应字段)。

### app 服务 env 注入点

prod compose 预留 `app` 服务(默认注释;如需启用,把 `APP_*` 键解开):`sqlconn` 指向 `mysql:3306`、`redisaddr` 指向 `redis:6379`、`redispass` 用 `${REDIS_PASSWORD}`;另需注入 `BEEGO_RUNMODE=prod`、`ITEM_RENTAL_JWT_SECRET`(32+ 字节强随机,代码对默认值 fail-fast panic)、`ITEM_RENTAL_PII_KEY`、`WECHAT_*` 全部键;多实例加 `redis_enabled=true`。

### 启动方式

```bash
cd deploy
export MYSQL_ROOT_PASSWORD='...' MYSQL_PASSWORD='...' REDIS_PASSWORD='...'
docker compose -f docker-compose.prod.yml up -d
docker compose -f docker-compose.prod.yml ps    # mysql/redis 均应为 healthy
docker compose -f docker-compose.prod.yml config # 校验插值
```

初始化表结构由 `deploy/initdb/001_schema.sql`(挂载到 `/docker-entrypoint-initdb.d`,仅首次建卷时执行)完成。
