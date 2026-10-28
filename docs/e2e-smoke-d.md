# E2E 冒烟记录（当前）

> 可执行真源:`scripts/e2e-smoke.sh`。历史阶段D/E/G 的详细记录见 git 提交历史。

## 前置

- MySQL 8 可连(`app.conf` 默认 `root:root123456@127.0.0.1:3306/rental`)。
- 环境变量:`ITEM_RENTAL_JWT_SECRET`、`ITEM_RENTAL_PII_KEY`(64 hex)、`WECHAT_MOCK=1`。
- Redis 可选(`redis_enabled=false` 时 refresh 会话不落 Redis)。
- 首次建表:`deploy/initdb/001_schema.sql`;脚本 `REBUILD_DB=1` 会 DROP/重建 rental 库。

## 断言范围

| 链 | 断言 |
| --- | --- |
| 链1 | 手机号登录(无验证码)、profile 带 token 200 / 无 token 真 401、物品多图上传与发布、owner 视图 4 语义、头像 png/gif、下单→签名 notify→pickup→return_request→return_confirm=4、real_name 加密后回读明文 |
| 链2a | cancel:支付→退款回 0→取消=5 |
| 链2b | breach:支付→pickup→退款 409 守卫→return_request→breach=6 |

> 当前脚本不包含押金/信用分、站内消息、搜索/geo、security-go 断言(相关功能已在开源版裁剪)。

## 运行

```bash
# 起后端(server/ 目录;静态/上传以 CWD 为基准)
cd server
ITEM_RENTAL_JWT_SECRET=rental-dev-smoke-2026 \
ITEM_RENTAL_PII_KEY=<64hex> \
WECHAT_MOCK=1 go run .

# 另开终端
bash scripts/e2e-smoke.sh
# 或重建库后跑:
REBUILD_DB=1 bash scripts/e2e-smoke.sh
```

## 说明

- 业务失败为 HTTP 200 + body `code`;JWT 中间件失败为真 HTTP 401。
- 冒烟结束如临时改过 MySQL root 密码,请还原。
- 上传文件落在 `server/static/uploads/`;脚本退出不会自动清理,dev 环境可接受。
